package main

import (
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	_ "github.com/marcboeker/go-duckdb"
	xlsReader "github.com/shakinm/xlsReader/xls"
	"golang.org/x/text/encoding/charmap"
)

func toUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	decoded, err := charmap.Windows1252.NewDecoder().String(s)
	if err != nil {
		return strings.ToValidUTF8(s, "\uFFFD")
	}
	return decoded
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: xls_debug <file.xls>")
		os.Exit(1)
	}

	workbook, err := xlsReader.OpenFile(os.Args[1])
	if err != nil {
		fmt.Printf("Error opening xls: %v\n", err)
		os.Exit(1)
	}

	dbPath := "test_import.duckdb"
	defer os.Remove(dbPath)
	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		fmt.Printf("Error opening duckdb: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	for si := 0; si < workbook.GetNumberSheets(); si++ {
		sheet, err := workbook.GetSheet(si)
		if err != nil {
			continue
		}
		name := sheet.GetName()
		fmt.Printf("\n=== Sheet: %s ===\n", name)

		var rows [][]string
		maxCols := 0
		for r := 0; r <= sheet.GetNumberRows(); r++ {
			row, err := sheet.GetRow(r)
			if err != nil {
				continue
			}
			cols := row.GetCols()
			rd := make([]string, len(cols))
			hasData := false
			for c, cell := range cols {
				rd[c] = toUTF8(cell.GetString())
				if !hasData && strings.TrimSpace(rd[c]) != "" {
					hasData = true
				}
			}
			if !hasData {
				continue
			}
			if len(rd) > maxCols {
				maxCols = len(rd)
			}
			rows = append(rows, rd)
		}
		for i, row := range rows {
			if len(row) < maxCols {
				p := make([]string, maxCols)
				copy(p, row)
				rows[i] = p
			}
		}
		if len(rows) == 0 {
			fmt.Println("  (empty)")
			continue
		}
		fmt.Printf("  %d rows, %d cols\n", len(rows), maxCols)

		// Infer types
		colTypes := make([]string, maxCols)
		for i := range colTypes {
			colTypes[i] = "TEXT"
		}
		for i := 0; i < maxCols; i++ {
			ct := ""
			for r := 1; r < len(rows) && r <= 100; r++ {
				if i >= len(rows[r]) || rows[r][i] == "" {
					continue
				}
				if _, err := strconv.ParseInt(rows[r][i], 10, 64); err == nil {
					if ct == "" {
						ct = "INTEGER"
					}
				} else if _, err := strconv.ParseFloat(rows[r][i], 64); err == nil {
					if ct == "" || ct == "INTEGER" {
						ct = "REAL"
					}
				} else {
					ct = "TEXT"
					break
				}
			}
			if ct != "" {
				colTypes[i] = ct
			}
		}
		headers := make([]string, maxCols)
		for i := 0; i < maxCols; i++ {
			if i < len(rows[0]) && rows[0][i] != "" {
				headers[i] = strings.ReplaceAll(strings.ReplaceAll(rows[0][i], " ", "_"), "-", "_")
			} else {
				headers[i] = fmt.Sprintf("col_%d", i)
			}
		}
		tn := strings.ReplaceAll(name, " ", "_")

		// CREATE TABLE
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf(`CREATE TABLE "%s" (`, tn))
		phs := make([]string, maxCols)
		for i, h := range headers {
			if i > 0 {
				sb.WriteString(", ")
			}
			sb.WriteString(fmt.Sprintf(`"%s" %s`, h, colTypes[i]))
			phs[i] = "?"
		}
		sb.WriteString(");")
		if _, err := db.Exec(sb.String()); err != nil {
			fmt.Printf("  CREATE error: %v\n", err)
			continue
		}

		// Batch INSERT (mirrors processSheet logic)
		const maxParams = 2000
		batchSize := maxParams / maxCols
		if batchSize < 1 {
			batchSize = 1
		}
		if batchSize > 500 {
			batchSize = 500
		}
		singlePH := "(" + strings.Join(phs, ",") + ")"
		dataRows := rows[1:]
		inserted := 0
		for bs := 0; bs < len(dataRows); bs += batchSize {
			be := bs + batchSize
			if be > len(dataRows) {
				be = len(dataRows)
			}
			batch := dataRows[bs:be]
			allVals := make([]interface{}, len(batch)*maxCols)
			rowPHs := make([]string, len(batch))
			for ri, row := range batch {
				base := ri * maxCols
				for j := 0; j < maxCols; j++ {
					if j >= len(row) || row[j] == "" {
						allVals[base+j] = nil
					} else if colTypes[j] == "INTEGER" {
						if iv, e := strconv.ParseInt(row[j], 10, 64); e == nil {
							allVals[base+j] = iv
						} else if fv, e := strconv.ParseFloat(row[j], 64); e == nil {
							allVals[base+j] = int64(fv)
						} else {
							allVals[base+j] = nil
						}
					} else if colTypes[j] == "REAL" {
						if fv, e := strconv.ParseFloat(row[j], 64); e == nil {
							allVals[base+j] = fv
						} else {
							allVals[base+j] = nil
						}
					} else {
						allVals[base+j] = row[j]
					}
				}
				rowPHs[ri] = singlePH
			}
			q := fmt.Sprintf(`INSERT INTO "%s" VALUES %s`, tn, strings.Join(rowPHs, ","))
			if _, err := db.Exec(q, allVals...); err != nil {
				fmt.Printf("  INSERT error at row %d (batch=%d): %v\n", bs+2, len(batch), err)
				break
			}
			inserted += len(batch)
		}
		fmt.Printf("  Inserted %d/%d rows\n", inserted, len(dataRows))
		var count int
		db.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM "%s"`, tn)).Scan(&count)
		fmt.Printf("  COUNT(*) = %d\n", count)
	}
}
