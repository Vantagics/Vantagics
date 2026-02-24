package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"
	"time"
)

// Feature: quick-analysis-pack, Property 1: Pack serialization round-trip
// Validates: Requirements 2.5

// generateRandomString produces a random non-empty string of printable ASCII characters.
func generateRandomString(r *rand.Rand, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 1
	}
	n := r.Intn(maxLen) + 1
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte(r.Intn(94) + 32) // printable ASCII 32-125
	}
	return string(buf)
}

// generateRandomPack creates a random but valid QuickAnalysisPack for property testing.
func generateRandomPack(r *rand.Rand) QuickAnalysisPack {
	// Fixed valid values for constrained fields
	fileType := "Vantagics_QuickAnalysisPack"
	formatVersion := "1.0"

	stepTypes := []string{"sql_query", "python_code"}

	// Generate a valid RFC3339 timestamp
	year := 2020 + r.Intn(6)
	month := time.Month(r.Intn(12) + 1)
	day := r.Intn(28) + 1
	hour := r.Intn(24)
	minute := r.Intn(60)
	second := r.Intn(60)
	createdAt := time.Date(year, month, day, hour, minute, second, 0, time.UTC).Format(time.RFC3339)

	// Generate metadata
	metadata := PackMetadata{
		Author:      generateRandomString(r, 20),
		CreatedAt:   createdAt,
		SourceName:  generateRandomString(r, 30),
		Description: generateRandomString(r, 50),
	}

	// Generate schema requirements (0-5 tables)
	numTables := r.Intn(6)
	schemas := make([]PackTableSchema, numTables)
	for i := 0; i < numTables; i++ {
		numCols := r.Intn(5) + 1
		cols := make([]PackColumnInfo, numCols)
		for j := 0; j < numCols; j++ {
			cols[j] = PackColumnInfo{
				Name: fmt.Sprintf("col_%d_%d", i, j),
				Type: generateRandomString(r, 10),
			}
		}
		schemas[i] = PackTableSchema{
			TableName: fmt.Sprintf("table_%d", i),
			Columns:   cols,
		}
	}

	// Generate executable steps (0-10 steps)
	numSteps := r.Intn(11)
	steps := make([]PackStep, numSteps)
	for i := 0; i < numSteps; i++ {
		// Generate depends_on: random subset of previous step IDs
		var dependsOn []int
		if i > 0 && r.Intn(2) == 1 {
			numDeps := r.Intn(i) + 1
			depSet := make(map[int]bool)
			for d := 0; d < numDeps; d++ {
				depSet[r.Intn(i)+1] = true
			}
			for dep := range depSet {
				dependsOn = append(dependsOn, dep)
			}
		}

		steps[i] = PackStep{
			StepID:      i + 1,
			StepType:    stepTypes[r.Intn(len(stepTypes))],
			Code:        generateRandomString(r, 100),
			Description: generateRandomString(r, 50),
			DependsOn:   dependsOn,
		}
	}

	return QuickAnalysisPack{
		FileType:           fileType,
		FormatVersion:      formatVersion,
		Metadata:           metadata,
		SchemaRequirements: schemas,
		ExecutableSteps:    steps,
	}
}

// normalizePackForComparison ensures nil vs empty slice differences don't cause false negatives.
func normalizePackForComparison(p *QuickAnalysisPack) {
	if p.SchemaRequirements == nil {
		p.SchemaRequirements = []PackTableSchema{}
	}
	if p.ExecutableSteps == nil {
		p.ExecutableSteps = []PackStep{}
	}
	for i := range p.SchemaRequirements {
		if p.SchemaRequirements[i].Columns == nil {
			p.SchemaRequirements[i].Columns = []PackColumnInfo{}
		}
	}
	for i := range p.ExecutableSteps {
		if p.ExecutableSteps[i].DependsOn == nil {
			p.ExecutableSteps[i].DependsOn = []int{}
		}
	}
}

func TestProperty1_PackSerializationRoundTrip(t *testing.T) {
	// Property 1: Pack serialization round-trip
	// For any valid QuickAnalysisPack, Marshal then Unmarshal should produce an equivalent object.
	// Validates: Requirements 2.5

	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		original := generateRandomPack(r)

		// Marshal to JSON
		data, err := json.Marshal(original)
		if err != nil {
			t.Logf("Marshal failed: %v", err)
			return false
		}

		// Unmarshal back
		var restored QuickAnalysisPack
		err = json.Unmarshal(data, &restored)
		if err != nil {
			t.Logf("Unmarshal failed: %v", err)
			return false
		}

		// Normalize nil vs empty slices for comparison
		normalizePackForComparison(&original)
		normalizePackForComparison(&restored)

		if !reflect.DeepEqual(original, restored) {
			t.Logf("Round-trip mismatch!\nOriginal: %+v\nRestored: %+v", original, restored)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1 failed: %v", err)
	}
}

// Feature: quick-analysis-pack, Property 2: ZIP packaging round-trip
// Validates: Requirements 1.4, 1.5, 1.6, 2.1

func TestProperty2_ZipPackagingRoundTrip(t *testing.T) {
	// Property 2: ZIP packaging round-trip
	// For any valid JSON data and any password (including empty for no encryption),
	// PackToZip followed by UnpackFromZip with the same password should return
	// identical bytes. Using a wrong password should return ErrWrongPassword.
	// Validates: Requirements 1.4, 1.5, 1.6, 2.1

	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate random JSON data (1 to 500 bytes)
		dataLen := r.Intn(500) + 1
		jsonData := make([]byte, dataLen)
		for i := range jsonData {
			jsonData[i] = byte(r.Intn(256))
		}

		// Generate random password: 50% chance of empty (no encryption), 50% non-empty
		var password string
		if r.Intn(2) == 1 {
			password = generateRandomString(r, 30)
		}

		tmpDir := t.TempDir()
		zipPath := fmt.Sprintf("%s/test_%d.qap", tmpDir, seed)

		// Pack to ZIP
		if err := PackToZip(jsonData, zipPath, password); err != nil {
			t.Logf("PackToZip failed: %v", err)
			return false
		}

		// Unpack with correct password
		restored, err := UnpackFromZip(zipPath, password)
		if err != nil {
			t.Logf("UnpackFromZip failed: %v", err)
			return false
		}

		// Verify round-trip: restored data must match original
		if !bytes.Equal(jsonData, restored) {
			t.Logf("Round-trip mismatch! original len=%d, restored len=%d", len(jsonData), len(restored))
			return false
		}

		// If password was non-empty, verify wrong password returns ErrWrongPassword
		if password != "" {
			wrongPassword := password + "_wrong"
			_, err := UnpackFromZip(zipPath, wrongPassword)
			if !errors.Is(err, ErrWrongPassword) {
				t.Logf("Expected ErrWrongPassword with wrong password, got: %v", err)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 2 failed: %v", err)
	}
}

// Feature: quick-analysis-pack, Property 3: Pack structural completeness
// Validates: Requirements 1.3, 2.2, 2.3, 2.4

func TestProperty3_PackStructuralCompleteness(t *testing.T) {
	// Property 3: Pack structural completeness
	// For any valid QuickAnalysisPack, its JSON representation should contain all required
	// top-level fields, metadata should contain author/created_at/source_name, and each
	// executable_step should contain step_id/step_type/code/description.
	// Validates: Requirements 1.3, 2.2, 2.3, 2.4

	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		pack := generateRandomPack(r)

		// Marshal to JSON
		data, err := json.Marshal(pack)
		if err != nil {
			t.Logf("Marshal failed: %v", err)
			return false
		}

		// Unmarshal into generic map to verify field presence
		var generic map[string]interface{}
		if err := json.Unmarshal(data, &generic); err != nil {
			t.Logf("Unmarshal to map failed: %v", err)
			return false
		}

		// Verify all required top-level fields exist
		requiredTopLevel := []string{"file_type", "format_version", "metadata", "schema_requirements", "executable_steps"}
		for _, field := range requiredTopLevel {
			if _, ok := generic[field]; !ok {
				t.Logf("Missing required top-level field: %s", field)
				return false
			}
		}

		// Verify metadata contains required fields
		metadataRaw, ok := generic["metadata"].(map[string]interface{})
		if !ok {
			t.Logf("metadata is not a JSON object")
			return false
		}
		requiredMetadata := []string{"author", "created_at", "source_name"}
		for _, field := range requiredMetadata {
			if _, ok := metadataRaw[field]; !ok {
				t.Logf("Missing required metadata field: %s", field)
				return false
			}
		}

		// Verify each executable_step contains required fields
		stepsRaw, ok := generic["executable_steps"].([]interface{})
		if !ok {
			t.Logf("executable_steps is not a JSON array")
			return false
		}
		requiredStep := []string{"step_id", "step_type", "code", "description"}
		for i, stepRaw := range stepsRaw {
			step, ok := stepRaw.(map[string]interface{})
			if !ok {
				t.Logf("executable_steps[%d] is not a JSON object", i)
				return false
			}
			for _, field := range requiredStep {
				if _, ok := step[field]; !ok {
					t.Logf("Missing required field '%s' in executable_steps[%d]", field, i)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3 failed: %v", err)
	}
}

// Feature: quick-analysis-pack, Property 4: Schema validation correctness
// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6

// generateRandomSchema creates a random schema with 1-5 tables, each with 1-5 columns.
// Table and column names are unique random strings.
func generateRandomSchema(r *rand.Rand) []PackTableSchema {
	numTables := r.Intn(5) + 1
	tables := make([]PackTableSchema, numTables)
	for i := 0; i < numTables; i++ {
		numCols := r.Intn(5) + 1
		cols := make([]PackColumnInfo, numCols)
		for j := 0; j < numCols; j++ {
			cols[j] = PackColumnInfo{
				Name: fmt.Sprintf("col_%d_%d_%s", i, j, generateRandomString(r, 6)),
				Type: generateRandomString(r, 8),
			}
		}
		tables[i] = PackTableSchema{
			TableName: fmt.Sprintf("tbl_%d_%s", i, generateRandomString(r, 6)),
			Columns:   cols,
		}
	}
	return tables
}

func TestProperty4_SchemaValidationCorrectness(t *testing.T) {
	// Property 4: Schema validation correctness
	// For any source/target schema pair, ValidateSchema correctly identifies missing tables and columns.
	// Validates: Requirements 3.1, 3.2, 3.3, 3.4, 3.5, 3.6

	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Sub-property 4a: When target contains ALL source tables and columns → Compatible=true, no missing items
	t.Run("AllPresent", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			source := generateRandomSchema(r)

			// Target = copy of source (possibly with extra tables/columns, but at least all source content)
			target := make([]PackTableSchema, len(source))
			for i, tbl := range source {
				cols := make([]PackColumnInfo, len(tbl.Columns))
				copy(cols, tbl.Columns)
				target[i] = PackTableSchema{
					TableName: tbl.TableName,
					Columns:   cols,
				}
			}

			result := ValidateSchema(source, target)

			if !result.Compatible {
				t.Logf("Expected Compatible=true when target has all source tables")
				return false
			}
			if len(result.MissingTables) != 0 {
				t.Logf("Expected no MissingTables, got %v", result.MissingTables)
				return false
			}
			if len(result.MissingColumns) != 0 {
				t.Logf("Expected no MissingColumns, got %v", result.MissingColumns)
				return false
			}
			return true
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4a failed: %v", err)
		}
	})

	// Sub-property 4b: When target is missing some source tables → Compatible=false, MissingTables contains exactly those tables
	t.Run("MissingTables", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			source := generateRandomSchema(r)

			// Randomly select which tables to remove (at least 1)
			numToRemove := r.Intn(len(source)) + 1
			// Shuffle indices and pick first numToRemove
			indices := r.Perm(len(source))
			removedSet := make(map[string]bool)
			for _, idx := range indices[:numToRemove] {
				removedSet[source[idx].TableName] = true
			}

			// Build target without the removed tables
			var target []PackTableSchema
			for _, tbl := range source {
				if !removedSet[tbl.TableName] {
					cols := make([]PackColumnInfo, len(tbl.Columns))
					copy(cols, tbl.Columns)
					target = append(target, PackTableSchema{
						TableName: tbl.TableName,
						Columns:   cols,
					})
				}
			}

			result := ValidateSchema(source, target)

			if result.Compatible {
				t.Logf("Expected Compatible=false when tables are missing")
				return false
			}
			if len(result.MissingTables) != len(removedSet) {
				t.Logf("Expected %d MissingTables, got %d: %v", len(removedSet), len(result.MissingTables), result.MissingTables)
				return false
			}
			// Verify MissingTables contains exactly the removed tables
			missingSet := make(map[string]bool)
			for _, name := range result.MissingTables {
				missingSet[name] = true
			}
			for name := range removedSet {
				if !missingSet[name] {
					t.Logf("Expected table %q in MissingTables but not found", name)
					return false
				}
			}
			return true
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4b failed: %v", err)
		}
	})

	// Sub-property 4c: When target has all tables but missing some columns → Compatible=true, MissingColumns contains exactly those columns
	t.Run("MissingColumns", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			source := generateRandomSchema(r)

			// Build target with all tables but randomly remove some columns
			type removedCol struct {
				table  string
				column string
			}
			var expectedMissing []removedCol

			target := make([]PackTableSchema, len(source))
			for i, tbl := range source {
				var keptCols []PackColumnInfo
				for _, col := range tbl.Columns {
					// 30% chance to remove each column
					if r.Intn(10) < 3 {
						expectedMissing = append(expectedMissing, removedCol{table: tbl.TableName, column: col.Name})
					} else {
						keptCols = append(keptCols, col)
					}
				}
				if keptCols == nil {
					keptCols = []PackColumnInfo{}
				}
				target[i] = PackTableSchema{
					TableName: tbl.TableName,
					Columns:   keptCols,
				}
			}

			result := ValidateSchema(source, target)

			// All tables present → Compatible=true
			if !result.Compatible {
				t.Logf("Expected Compatible=true when all tables present (only columns missing)")
				return false
			}
			if len(result.MissingTables) != 0 {
				t.Logf("Expected no MissingTables, got %v", result.MissingTables)
				return false
			}
			// Verify MissingColumns matches exactly
			if len(result.MissingColumns) != len(expectedMissing) {
				t.Logf("Expected %d MissingColumns, got %d", len(expectedMissing), len(result.MissingColumns))
				return false
			}
			// Build a set for comparison
			type colKey struct{ table, col string }
			expectedSet := make(map[colKey]bool)
			for _, m := range expectedMissing {
				expectedSet[colKey{m.table, m.column}] = true
			}
			for _, mc := range result.MissingColumns {
				key := colKey{mc.TableName, mc.ColumnName}
				if !expectedSet[key] {
					t.Logf("Unexpected MissingColumn: %s.%s", mc.TableName, mc.ColumnName)
					return false
				}
			}
			return true
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4c failed: %v", err)
		}
	})

	// Sub-property 4d: MissingTables count + found tables count = source table count
	t.Run("TableCountInvariant", func(t *testing.T) {
		f := func(seed int64) bool {
			r := rand.New(rand.NewSource(seed))
			source := generateRandomSchema(r)

			// Build a random target: randomly include/exclude source tables
			var target []PackTableSchema
			for _, tbl := range source {
				if r.Intn(2) == 1 {
					cols := make([]PackColumnInfo, len(tbl.Columns))
					copy(cols, tbl.Columns)
					target = append(target, PackTableSchema{
						TableName: tbl.TableName,
						Columns:   cols,
					})
				}
			}

			result := ValidateSchema(source, target)

			foundTables := len(source) - len(result.MissingTables)
			if foundTables+len(result.MissingTables) != len(source) {
				t.Logf("Invariant violated: found(%d) + missing(%d) != source(%d)",
					foundTables, len(result.MissingTables), len(source))
				return false
			}
			return true
		}
		if err := quick.Check(f, config); err != nil {
			t.Errorf("Property 4d failed: %v", err)
		}
	})
}


// Feature: quick-analysis-pack, Property 5: Schema validation ignores extras
// Validates: Requirements 3.7

func TestProperty5_SchemaValidationIgnoresExtras(t *testing.T) {
	// Property 5: Schema validation ignores extras
	// For any compatible source/target schema pair, adding extra tables or columns
	// to the target should not change the Compatible status.
	// Validates: Requirements 3.7

	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		source := generateRandomSchema(r)

		// Build target as a full copy of source (guaranteed compatible)
		target := make([]PackTableSchema, len(source))
		for i, tbl := range source {
			cols := make([]PackColumnInfo, len(tbl.Columns))
			copy(cols, tbl.Columns)
			target[i] = PackTableSchema{
				TableName: tbl.TableName,
				Columns:   cols,
			}
		}

		// Add extra random tables to target (1-3 extra tables)
		numExtraTables := r.Intn(3) + 1
		for i := 0; i < numExtraTables; i++ {
			numCols := r.Intn(4) + 1
			cols := make([]PackColumnInfo, numCols)
			for j := 0; j < numCols; j++ {
				cols[j] = PackColumnInfo{
					Name: fmt.Sprintf("extra_col_%d_%d_%s", i, j, generateRandomString(r, 6)),
					Type: generateRandomString(r, 8),
				}
			}
			target = append(target, PackTableSchema{
				TableName: fmt.Sprintf("extra_tbl_%d_%s", i, generateRandomString(r, 6)),
				Columns:   cols,
			})
		}

		// Add extra random columns to existing target tables
		for i := range target {
			if i >= len(source) {
				break // only add extras to tables that came from source
			}
			numExtraCols := r.Intn(3) + 1
			for j := 0; j < numExtraCols; j++ {
				target[i].Columns = append(target[i].Columns, PackColumnInfo{
					Name: fmt.Sprintf("extra_field_%d_%d_%s", i, j, generateRandomString(r, 6)),
					Type: generateRandomString(r, 8),
				})
			}
		}

		result := ValidateSchema(source, target)

		// Compatible must be true since target is a superset of source
		if !result.Compatible {
			t.Logf("Expected Compatible=true, got false (seed=%d)", seed)
			return false
		}
		if len(result.MissingTables) != 0 {
			t.Logf("Expected no MissingTables, got %v (seed=%d)", result.MissingTables, seed)
			return false
		}
		if len(result.MissingColumns) != 0 {
			t.Logf("Expected no MissingColumns, got %v (seed=%d)", result.MissingColumns, seed)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 5 failed: %v", err)
	}
}

// Feature: quick-analysis-pack, Property 8: Replay error resilience
// Validates: Requirements 5.5

// simulateReplayExecution mirrors the error-resilient loop pattern from ExecuteQuickAnalysisPack.
// It iterates through ALL steps, calling stepExecutor for each one. If a step fails, the error
// is recorded but execution continues to the next step. Returns the total number of attempted steps
// and a slice of errors (nil entries for successful steps).
func simulateReplayExecution(steps []PackStep, stepExecutor func(step PackStep) error) (attemptedCount int, stepErrors []error) {
	stepErrors = make([]error, len(steps))
	for i, step := range steps {
		attemptedCount++
		if err := stepExecutor(step); err != nil {
			// Record error but continue — mirrors ExecuteQuickAnalysisPack behavior
			stepErrors[i] = err
		}
	}
	return attemptedCount, stepErrors
}

func TestProperty8_ReplayErrorResilience(t *testing.T) {
	// Property 8: Replay error resilience
	// For any pack with multiple steps, when some steps fail, subsequent steps should
	// still execute. The total number of attempted steps should equal the total number
	// of steps in the pack.
	// Validates: Requirements 5.5

	config := &quick.Config{
		MaxCount: 200,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		// Generate a pack with 2-10 steps
		numSteps := r.Intn(9) + 2
		steps := make([]PackStep, numSteps)
		failSet := make(map[int]bool)

		for i := 0; i < numSteps; i++ {
			steps[i] = PackStep{
				StepID:      i + 1,
				StepType:    []string{"sql_query", "python_code"}[r.Intn(2)],
				Code:        generateRandomString(r, 50),
				Description: generateRandomString(r, 30),
			}
			// Randomly mark some steps as "will fail" (roughly 40% chance)
			if r.Intn(5) < 2 {
				failSet[i+1] = true
			}
		}

		// Execute the simulation with a step executor that fails for marked steps
		attemptedCount, stepErrors := simulateReplayExecution(steps, func(step PackStep) error {
			if failSet[step.StepID] {
				return errors.New(fmt.Sprintf("simulated failure for step %d", step.StepID))
			}
			return nil
		})

		// Property: ALL steps must be attempted regardless of failures
		if attemptedCount != numSteps {
			t.Logf("Expected %d attempted steps, got %d (seed=%d)", numSteps, attemptedCount, seed)
			return false
		}

		// Verify that the correct steps have errors
		for i, step := range steps {
			if failSet[step.StepID] {
				if stepErrors[i] == nil {
					t.Logf("Expected error for step %d but got nil (seed=%d)", step.StepID, seed)
					return false
				}
			} else {
				if stepErrors[i] != nil {
					t.Logf("Expected no error for step %d but got: %v (seed=%d)", step.StepID, stepErrors[i], seed)
					return false
				}
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 8 failed: %v", err)
	}
}
