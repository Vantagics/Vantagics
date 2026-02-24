package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"vantagics/agent"
	"vantagics/i18n"
	
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// TableSchemaInfo 表结构信�?
type TableSchemaInfo struct {
	TableName  string              `json:"table_name"`
	Columns    []ColumnInfo        `json:"columns"`
	SampleData []map[string]any    `json:"sample_data"`
}

// ColumnInfo 字段信息
type ColumnInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

// SemanticOptimizationResult LLM 返回的优化结�?
type SemanticOptimizationResult struct {
	Tables []TableOptimization `json:"tables"`
}

// TableOptimization 表的优化结果
type TableOptimization struct {
	OriginalTableName  string          `json:"original_table_name"`
	OptimizedTableName string          `json:"optimized_table_name"`
	ColumnMappings     []ColumnMapping `json:"column_mappings"`
	Description        string          `json:"description"`
}

// ColumnMapping 字段映射
type ColumnMapping struct {
	OriginalName  string `json:"original_name"`
	OptimizedName string `json:"optimized_name"`
	Description   string `json:"description"`
}

// SemanticOptimizeDataSource 对数据源进行语义优化
func (a *App) SemanticOptimizeDataSource(sourceID string) error {
	a.Log("Starting semantic optimization for data source: " + sourceID)

	// 1. 获取原数据源信息
	sources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		return fmt.Errorf("failed to load data sources: %w", err)
	}

	var originalSource *agent.DataSource
	for _, s := range sources {
		if s.ID == sourceID {
			originalSource = &s
			break
		}
	}

	if originalSource == nil {
		return fmt.Errorf("data source not found: %s", sourceID)
	}

	// 发送进度事�?
	a.sendSemanticOptimizeProgress("正在分析表结�?..")

	// 2. 收集所有表的结构和样本数据
	schemas, err := a.collectTableSchemas(sourceID)
	if err != nil {
		return fmt.Errorf("failed to collect table schemas: %w", err)
	}

	if len(schemas) == 0 {
		return fmt.Errorf("no tables found in data source")
	}

	a.sendSemanticOptimizeProgress("正在生成优化方案...")

	// 3. 调用 LLM 生成优化方案
	cfg, err := a.GetConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	optimization, err := a.generateSemanticOptimization(schemas, cfg.Language)
	if err != nil {
		return fmt.Errorf("failed to generate optimization: %w", err)
	}

	a.sendSemanticOptimizeProgress("正在创建新数据源...")

	// 4. 创建新数据源（返回新数据源和完整数据库路径）
	// 传�?schemas 以便使用已收集的列类型信�?
	newSource, newDBFullPath, err := a.createOptimizedDataSource(originalSource, optimization, schemas)
	if err != nil {
		return fmt.Errorf("failed to create optimized data source: %w", err)
	}

	a.sendSemanticOptimizeProgress("正在迁移数据...")

	// 5. 迁移数据（使用完整路径）
	err = a.migrateDataWithOptimization(originalSource, newDBFullPath, optimization)
	if err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}

	a.sendSemanticOptimizeProgress("完成")

	// 7. 发送完成事�?
	runtime.EventsEmit(a.ctx, "semantic-optimize-completed", map[string]interface{}{
		"original_id": sourceID,
		"new_id":      newSource.ID,
		"new_name":    newSource.Name,
	})

	a.Log("Semantic optimization completed successfully")
	return nil
}

// collectTableSchemas 收集表结构信�?
func (a *App) collectTableSchemas(sourceID string) ([]TableSchemaInfo, error) {
	// 获取数据源信�?
	sources, err := a.dataSourceService.LoadDataSources()
	if err != nil {
		a.Log(fmt.Sprintf("[SEMANTIC] Error loading data sources: %v", err))
		return nil, err
	}

	var ds *agent.DataSource
	for _, s := range sources {
		if s.ID == sourceID {
			ds = &s
			break
		}
	}

	if ds == nil {
		a.Log(fmt.Sprintf("[SEMANTIC] Data source not found: %s", sourceID))
		return nil, fmt.Errorf("data source not found: %s", sourceID)
	}

	a.Log(fmt.Sprintf("[SEMANTIC] Data source info - ID: %s, Name: %s, Type: %s, DBPath: %s",
		ds.ID, ds.Name, ds.Type, ds.Config.DBPath))

	// 首先尝试�?analysis.schema 获取表信�?
	if ds.Analysis != nil && len(ds.Analysis.Schema) > 0 {
		a.Log(fmt.Sprintf("[SEMANTIC] Using analysis.schema, found %d tables", len(ds.Analysis.Schema)))
		schemas := make([]TableSchemaInfo, 0, len(ds.Analysis.Schema))
		
		for _, tableSchema := range ds.Analysis.Schema {
			// 尝试获取真实的列类型信息
			columnInfos := make([]ColumnInfo, 0, len(tableSchema.Columns))
			realColumns, err := a.dataSourceService.GetTableColumns(sourceID, tableSchema.TableName)
			if err == nil && len(realColumns) > 0 {
				// 成功获取真实列信�?
				columnTypeMap := make(map[string]string)
				for _, col := range realColumns {
					columnTypeMap[col.Name] = col.Type
				}
				for _, colName := range tableSchema.Columns {
					colType := columnTypeMap[colName]
					if colType == "" {
						colType = "TEXT"
					}
					columnInfos = append(columnInfos, ColumnInfo{
						Name:     colName,
						Type:     colType,
						Nullable: true,
					})
				}
			} else {
				// 无法获取真实列信息，使用默认类型
				a.Log(fmt.Sprintf("[SEMANTIC] Could not get real column types for %s, using TEXT: %v", tableSchema.TableName, err))
				for _, colName := range tableSchema.Columns {
					columnInfos = append(columnInfos, ColumnInfo{
						Name:     colName,
						Type:     "TEXT",
						Nullable: true,
					})
				}
			}
			
			// 尝试获取样本数据
			sampleData, err := a.getSampleData(sourceID, tableSchema.TableName, 2)
			if err != nil {
				a.Log(fmt.Sprintf("Failed to get sample data for table %s: %v", tableSchema.TableName, err))
				sampleData = []map[string]any{}
			}
			
			schemas = append(schemas, TableSchemaInfo{
				TableName:  tableSchema.TableName,
				Columns:    columnInfos,
				SampleData: sampleData,
			})
		}
		
		return schemas, nil
	}

	// 获取所有表�?
	a.Log(fmt.Sprintf("[SEMANTIC] Getting tables for source: %s", sourceID))
	tables, err := a.dataSourceService.GetTables(sourceID)
	if err != nil {
		a.Log(fmt.Sprintf("[SEMANTIC] Error getting tables: %v", err))
		return nil, err
	}
	a.Log(fmt.Sprintf("[SEMANTIC] Found %d tables: %v", len(tables), tables))

	// 如果 GetTables 返回空，尝试使用 GetDataSourceTables
	if len(tables) == 0 {
		a.Log("[SEMANTIC] GetTables returned empty, trying GetDataSourceTables...")
		tables, err = a.dataSourceService.GetDataSourceTables(sourceID)
		if err != nil {
			a.Log(fmt.Sprintf("[SEMANTIC] Error getting tables via GetDataSourceTables: %v", err))
			return nil, err
		}
		a.Log(fmt.Sprintf("[SEMANTIC] GetDataSourceTables found %d tables: %v", len(tables), tables))
	}

	schemas := make([]TableSchemaInfo, 0, len(tables))

	for _, tableName := range tables {
		// 获取表结�?
		columns, err := a.dataSourceService.GetTableColumns(sourceID, tableName)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to get columns for table %s: %v", tableName, err))
			continue
		}

		columnInfos := make([]ColumnInfo, 0, len(columns))
		for _, col := range columns {
			columnInfos = append(columnInfos, ColumnInfo{
				Name:     col.Name,
				Type:     col.Type,
				Nullable: col.Nullable,
			})
		}

		// 获取前两行样本数�?
		sampleData, err := a.getSampleData(sourceID, tableName, 2)
		if err != nil {
			a.Log(fmt.Sprintf("Failed to get sample data for table %s: %v", tableName, err))
			sampleData = []map[string]any{} // 继续处理，即使没有样本数�?
		}

		schemas = append(schemas, TableSchemaInfo{
			TableName:  tableName,
			Columns:    columnInfos,
			SampleData: sampleData,
		})
	}

	return schemas, nil
}

// getSampleData 获取表的样本数据
func (a *App) getSampleData(sourceID, tableName string, limit int) ([]map[string]any, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", tableName, limit)
	return a.dataSourceService.ExecuteQuery(sourceID, query)
}

// generateSemanticOptimization 调用 LLM 生成优化方案
func (a *App) generateSemanticOptimization(schemas []TableSchemaInfo, language string) (*SemanticOptimizationResult, error) {
	// 构建 prompt
	prompt := a.buildSemanticOptimizationPrompt(schemas, language)

	// 获取配置并创�?LLM 服务
	cfg, err := a.GetEffectiveConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	llm := agent.NewLLMService(cfg, a.Log)

	// 调用 LLM
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	response, err := llm.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析 JSON 响应
	var result SemanticOptimizationResult
	
	// 尝试提取 JSON（可能包含在 markdown 代码块中�?
	content := response
	
	// 先尝试找 ```json 代码�?
	if idx := strings.Index(content, "```json"); idx != -1 {
		// 找到 ```json 后的内容
		start := idx + 7
		// 跳过可能的换行符
		for start < len(content) && (content[start] == '\n' || content[start] == '\r') {
			start++
		}
		// 找到结束�?```
		remaining := content[start:]
		if endIdx := strings.Index(remaining, "```"); endIdx != -1 {
			content = remaining[:endIdx]
		} else {
			content = remaining
		}
	} else if idx := strings.Index(content, "```"); idx != -1 {
		// 找到普�?``` 代码�?
		start := idx + 3
		// 跳过可能的换行符
		for start < len(content) && (content[start] == '\n' || content[start] == '\r') {
			start++
		}
		remaining := content[start:]
		if endIdx := strings.Index(remaining, "```"); endIdx != -1 {
			content = remaining[:endIdx]
		} else {
			content = remaining
		}
	}

	content = strings.TrimSpace(content)
	
	// 确保内容�?{ 开头（找到第一�?{ 的位置）
	if idx := strings.Index(content, "{"); idx > 0 {
		content = content[idx:]
	}
	
	// 确保内容�?} 结尾（找到最后一�?} 的位置）
	if idx := strings.LastIndex(content, "}"); idx != -1 && idx < len(content)-1 {
		content = content[:idx+1]
	}

	err = json.Unmarshal([]byte(content), &result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w\nResponse: %s", err, content)
	}

	return &result, nil
}

// buildSemanticOptimizationPrompt 构建语义优化 prompt
func (a *App) buildSemanticOptimizationPrompt(schemas []TableSchemaInfo, language string) string {
	// 序列化表结构信息
	schemasJSON, _ := json.MarshalIndent(schemas, "", "  ")

	langInstruction := "Use English for field names"
	if language == "简体中�? {
		langInstruction = "使用英文命名字段，但要考虑中文业务语义"
	}

	prompt := fmt.Sprintf(`# 数据源语义优化任�?

你是一个数据库设计专家。请分析以下数据表的结构和样本数据，为每个字段生成更有意义、更易理解的字段名�?

## 要求

1. **字段名规�?*�?
   - 使用小写字母和下划线（snake_case�?
   - 名称要简洁但有意�?
   - 避免缩写，除非是通用缩写（如 id, url, qty�?
   - %s

2. **保持数据类型**�?
   - 不改变字段的数据类型
   - 保持字段的可空�?

3. **考虑业务语义**�?
   - 根据样本数据推断字段的业务含�?
   - 使用业务术语而非技术术�?
   - 考虑字段之间的关联关�?

4. **表名优化**�?
   - 如果表名不够清晰，也可以优化表名
   - 表名应该反映表的业务含义
   - 使用 snake_case 命名

## 输入数据

%s

## 输出格式

返回 JSON 格式，结构如下：

{
  "tables": [
    {
      "original_table_name": "原表�?,
      "optimized_table_name": "优化后的表名",
      "description": "表的业务描述",
      "column_mappings": [
        {
          "original_name": "原字段名",
          "optimized_name": "优化后的字段�?,
          "description": "字段的业务含�?
        }
      ]
    }
  ]
}

## 重要提示

- 只返�?JSON，不要包含其他说明文�?
- 确保所有原始字段都有对应的映射
- 如果字段名已经很好，可以保持不变
- 优化后的字段名必须是有效�?SQL 标识�?

请开始优化：`, langInstruction, string(schemasJSON))

	return prompt
}

// createOptimizedDataSource 创建优化后的数据�?
func (a *App) createOptimizedDataSource(originalSource *agent.DataSource, optimization *SemanticOptimizationResult, schemas []TableSchemaInfo) (*agent.DataSource, string, error) {
	// 创建新的数据源名�?
	newName := originalSource.Name + "_语义优化"

	// 创建新的 DuckDB 数据库文件（返回完整路径�?
	newDBFullPath, err := a.dataSourceService.CreateOptimizedDatabase(originalSource, newName)
	if err != nil {
		return nil, "", err
	}

	// 打开新数据库
	newDB, err := a.dataSourceService.DB.OpenNew(newDBFullPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open new database: %w", err)
	}
	defer newDB.Close()

	// 构建表名�?schema 的映射，用于获取列类�?
	schemaMap := make(map[string]TableSchemaInfo)
	for _, schema := range schemas {
		schemaMap[schema.TableName] = schema
	}

	// 创建优化后的表结�?
	for _, tableOpt := range optimization.Tables {
		// �?schemas 中获取原表的列类型信�?
		originalSchema, hasSchema := schemaMap[tableOpt.OriginalTableName]
		err = a.createOptimizedTable(newDB, tableOpt, originalSchema, hasSchema)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create table %s: %w", tableOpt.OptimizedTableName, err)
		}
	}

	// 从完整路径中提取相对路径（sources/{id}/data.db�?
	// 完整路径格式：{dataCacheDir}/sources/{id}/data.db
	cfg, err := a.GetConfig()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get config: %w", err)
	}
	
	// 计算相对路径
	relDBPath, err := filepath.Rel(cfg.DataCacheDir, newDBFullPath)
	if err != nil {
		// 如果无法计算相对路径，尝试从路径中提�?sources/{id}/data.db 部分
		parts := strings.Split(filepath.ToSlash(newDBFullPath), "/")
		for i, part := range parts {
			if part == "sources" && i+2 < len(parts) {
				relDBPath = filepath.Join("sources", parts[i+1], parts[i+2])
				break
			}
		}
		if relDBPath == "" {
			relDBPath = filepath.Base(newDBFullPath)
		}
	}

	a.Log(fmt.Sprintf("[SEMANTIC] New database relative path: %s", relDBPath))

	// 构建新数据源�?schema 信息
	newSchema := make([]agent.TableSchema, 0, len(optimization.Tables))
	for _, tableOpt := range optimization.Tables {
		columns := make([]string, 0, len(tableOpt.ColumnMappings))
		for _, mapping := range tableOpt.ColumnMappings {
			columns = append(columns, mapping.OptimizedName)
		}
		newSchema = append(newSchema, agent.TableSchema{
			TableName: tableOpt.OptimizedTableName,
			Columns:   columns,
		})
	}

	// 构建数据摘要描述
	var summaryParts []string
	for _, tableOpt := range optimization.Tables {
		if tableOpt.Description != "" {
			summaryParts = append(summaryParts, fmt.Sprintf("%s: %s", tableOpt.OptimizedTableName, tableOpt.Description))
		}
	}
	summary := strings.Join(summaryParts, "\n")
	if summary == "" {
		summary = i18n.T("datasource.semantic_opt_summary", len(optimization.Tables))
	}

	// 注册新数据源（使用相对路径，包含 schema 信息�?
	newSource := &agent.DataSource{
		ID:   generateID(),
		Name: newName,
		Type: "sqlite",
		Config: agent.DataSourceConfig{
			DBPath:    relDBPath,
			Optimized: true,
		},
		Analysis: &agent.DataSourceAnalysis{
			Summary: summary,
			Schema:  newSchema,
		},
	}

	err = a.dataSourceService.SaveDataSource(newSource)
	if err != nil {
		return nil, "", fmt.Errorf("failed to save new data source: %w", err)
	}

	// 返回新数据源和完整路径（用于数据迁移�?
	return newSource, newDBFullPath, nil
}

// validateAndFixSQL 使用 LLM 验证和修�?SQL 语句
func (a *App) validateAndFixSQL(sqlStmt string, tableName string) (string, error) {
	// 由于我们已经使用双引号包裹了所有标识符，SQL 应该是正确的
	// 直接返回原始 SQL，避�?LLM 添加额外内容导致语法错误
	a.Log(fmt.Sprintf("SQL for table %s: %s", tableName, sqlStmt))
	return sqlStmt, nil
}

// createOptimizedTable 创建优化后的�?
// 使用已收集的 schema 信息，避免再次查询原数据�?
func (a *App) createOptimizedTable(db *sql.DB, tableOpt TableOptimization, originalSchema TableSchemaInfo, hasSchema bool) error {
	// 构建列定义映射（从已收集�?schema 中获取类型）
	columnTypeMap := make(map[string]string)
	if hasSchema {
		for _, col := range originalSchema.Columns {
			columnTypeMap[col.Name] = col.Type
		}
	}

	// 构建 CREATE TABLE 语句
	var columnDefs []string
	for _, mapping := range tableOpt.ColumnMappings {
		colType := columnTypeMap[mapping.OriginalName]
		if colType == "" {
			colType = "TEXT" // 默认类型
		}
		// 使用双引号包裹列名，避免保留字问�?
		columnDefs = append(columnDefs, fmt.Sprintf(`"%s" %s`, mapping.OptimizedName, colType))
	}

	// 使用双引号包裹表�?
	createSQL := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS "%s" (%s)`,
		tableOpt.OptimizedTableName,
		strings.Join(columnDefs, ", "))

	// 使用 LLM 验证和修�?SQL
	fixedSQL, err := a.validateAndFixSQL(createSQL, tableOpt.OptimizedTableName)
	if err != nil {
		a.Log("SQL validation error: " + err.Error())
		// 继续使用原始 SQL
		fixedSQL = createSQL
	}

	a.Log(fmt.Sprintf("Creating table with SQL: %s", fixedSQL))

	_, err = db.Exec(fixedSQL)
	if err != nil {
		// 如果执行失败，记录详细错�?
		return fmt.Errorf("failed to create table %s: %w\nSQL: %s", tableOpt.OptimizedTableName, err, fixedSQL)
	}
	return nil
}

// migrateDataWithOptimization 使用优化方案迁移数据
func (a *App) migrateDataWithOptimization(originalSource *agent.DataSource, targetDBPath string, optimization *SemanticOptimizationResult) error {
	// 打开原数据库
	sourceDB, err := a.dataSourceService.GetConnection(originalSource.ID)
	if err != nil {
		return fmt.Errorf("failed to connect to source database: %w", err)
	}
	defer sourceDB.Close()

	// 打开新数据库（使用完整路径）
	a.Log(fmt.Sprintf("Opening target database: %s", targetDBPath))
	targetDB, err := a.dataSourceService.DB.OpenWritable(targetDBPath)
	if err != nil {
		return fmt.Errorf("failed to open target database: %w", err)
	}
	defer targetDB.Close()

	// 确定源数据库类型，用于正确的标识符引�?
	sourceDBType := originalSource.Type // "mysql", "sqlite", "excel", "csv", etc.
	
	// 迁移每个表的数据
	for _, tableOpt := range optimization.Tables {
		err = a.migrateTableData(sourceDB, targetDB, tableOpt, sourceDBType)
		if err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", tableOpt.OriginalTableName, err)
		}
	}

	return nil
}

// quoteIdentifier 根据数据库类型返回正确的标识符引�?
func quoteIdentifier(name string, dbType string) string {
	switch dbType {
	case "mysql", "doris":
		return fmt.Sprintf("`%s`", name)
	default:
		// DuckDB, PostgreSQL 使用双引�?
		return fmt.Sprintf(`"%s"`, name)
	}
}

// migrateTableData 迁移单个表的数据
func (a *App) migrateTableData(sourceDB, targetDB *sql.DB, tableOpt TableOptimization, sourceDBType string) error {
	// 构建字段列表
	originalCols := make([]string, 0, len(tableOpt.ColumnMappings))
	optimizedCols := make([]string, 0, len(tableOpt.ColumnMappings))

	for _, mapping := range tableOpt.ColumnMappings {
		// 原表列名使用源数据库的引用方�?
		originalCols = append(originalCols, quoteIdentifier(mapping.OriginalName, sourceDBType))
		// 新表列名使用 DuckDB 的引用方式（双引号）
		optimizedCols = append(optimizedCols, quoteIdentifier(mapping.OptimizedName, "duckdb"))
	}

	// 查询原表数据（使用源数据库的引用方式�?
	query := fmt.Sprintf(`SELECT %s FROM %s`,
		strings.Join(originalCols, ", "),
		quoteIdentifier(tableOpt.OriginalTableName, sourceDBType))

	a.Log(fmt.Sprintf("Querying source table: %s", query))

	rows, err := sourceDB.Query(query)
	if err != nil {
		return fmt.Errorf("failed to query source table: %w", err)
	}
	defer rows.Close()

	// 准备插入语句
	placeholders := make([]string, len(optimizedCols))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	insertQuery := fmt.Sprintf(`INSERT INTO "%s" (%s) VALUES (%s)`,
		tableOpt.OptimizedTableName,
		strings.Join(optimizedCols, ", "),
		strings.Join(placeholders, ", "))

	a.Log(fmt.Sprintf("Insert query: %s", insertQuery))

	// 开始事�?
	tx, err := targetDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// 在事务中准备语句
	stmt, err := tx.Prepare(insertQuery)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// 逐行迁移
	rowCount := 0
	for rows.Next() {
		values := make([]interface{}, len(originalCols))
		valuePtrs := make([]interface{}, len(originalCols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to scan row: %w", err)
		}

		if _, err := stmt.Exec(values...); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row: %w", err)
		}

		rowCount++
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return fmt.Errorf("error iterating rows: %w", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	a.Log(fmt.Sprintf("Migrated %d rows from %s to %s", rowCount, tableOpt.OriginalTableName, tableOpt.OptimizedTableName))
	return nil
}

// sendSemanticOptimizeProgress 发送进度事�?
func (a *App) sendSemanticOptimizeProgress(message string) {
	runtime.EventsEmit(a.ctx, "semantic-optimize-progress", map[string]interface{}{
		"message": message,
	})
}

// generateID 生成唯一 ID
func generateID() string {
	return fmt.Sprintf("ds_%d", time.Now().UnixNano())
}
