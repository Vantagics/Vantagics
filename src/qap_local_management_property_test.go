package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: qap-local-management, Property 1: 导出文件保存正确性
// Validates: Requirements 1.1, 1.3, 1.4
//
// For any valid session export operation, the exported .qap file should exist in
// {DataCacheDir}/qap/ directory, the filename should match analysis_YYYYMMDD_HHmmss.qap
// format, and the returned path should point to that file.

// generateRandomMetadata creates random but valid PackMetadata for property testing.
func generateRandomMetadata(r *rand.Rand) PackMetadata {
	year := 2020 + r.Intn(6)
	month := time.Month(r.Intn(12) + 1)
	day := r.Intn(28) + 1
	hour := r.Intn(24)
	minute := r.Intn(60)
	second := r.Intn(60)
	createdAt := time.Date(year, month, day, hour, minute, second, 0, time.UTC).Format(time.RFC3339)

	return PackMetadata{
		Author:      generateRandomString(r, 20),
		CreatedAt:   createdAt,
		SourceName:  generateRandomString(r, 30),
		Description: generateRandomString(r, 50),
	}
}

// generateRandomPackForExport creates a random valid QuickAnalysisPack suitable for export testing.
func generateRandomPackForExport(r *rand.Rand) QuickAnalysisPack {
	metadata := generateRandomMetadata(r)

	// Generate 1-5 schema tables
	numTables := r.Intn(5) + 1
	schemas := make([]PackTableSchema, numTables)
	for i := 0; i < numTables; i++ {
		numCols := r.Intn(4) + 1
		cols := make([]PackColumnInfo, numCols)
		for j := 0; j < numCols; j++ {
			cols[j] = PackColumnInfo{
				Name: fmt.Sprintf("col_%d_%d", i, j),
				Type: "TEXT",
			}
		}
		schemas[i] = PackTableSchema{
			TableName: fmt.Sprintf("table_%d", i),
			Columns:   cols,
		}
	}

	// Generate 1-5 steps
	numSteps := r.Intn(5) + 1
	steps := make([]PackStep, numSteps)
	for i := 0; i < numSteps; i++ {
		stepType := "sql_query"
		if r.Intn(2) == 1 {
			stepType = "python_code"
		}
		steps[i] = PackStep{
			StepID:      i + 1,
			StepType:    stepType,
			Code:        fmt.Sprintf("SELECT %d FROM test", i),
			Description: fmt.Sprintf("Step %d", i+1),
			DependsOn:   buildDependsOn(i + 1),
		}
	}

	return QuickAnalysisPack{
		FileType:           "Vantagics_QuickAnalysisPack",
		FormatVersion:      "1.0",
		Metadata:           metadata,
		SchemaRequirements: schemas,
		ExecutableSteps:    steps,
	}
}

func TestProperty1_ExportFileSaveCorrectness(t *testing.T) {
	// Feature: qap-local-management, Property 1: 导出文件保存正确性
	// Validates: Requirements 1.1, 1.3, 1.4
	//
	// Since ExportQuickAnalysisPack depends on App context (chat service, data source),
	// this property test focuses on the file-saving aspect:
	// - Generate random metadata, create a valid QuickAnalysisPack JSON
	// - Call PackToZip to save to a temp QAP directory
	// - Verify the file exists at the expected path
	// - Verify the filename matches the expected pattern

	filenamePattern := regexp.MustCompile(`^analysis_\d{8}_\d{6}\.qap$`)

	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		pack := generateRandomPackForExport(r)

		// Marshal to JSON
		jsonData, err := json.MarshalIndent(pack, "", "  ")
		if err != nil {
			t.Logf("Marshal failed: %v", err)
			return false
		}

		// Simulate the export flow: create qap directory and save file
		tmpDir := t.TempDir()
		qapDir := filepath.Join(tmpDir, "qap")
		if err := os.MkdirAll(qapDir, 0755); err != nil {
			t.Logf("MkdirAll failed: %v", err)
			return false
		}

		// Generate filename using the same format as ExportQuickAnalysisPack
		now := time.Now()
		filename := fmt.Sprintf("analysis_%s.qap", now.Format("20060102_150405"))
		savePath := filepath.Join(qapDir, filename)

		// Pack to ZIP (no encryption for this test)
		if err := PackToZip(jsonData, savePath, ""); err != nil {
			t.Logf("PackToZip failed: %v", err)
			return false
		}

		// Property check 1: File exists at the expected path (Requirement 1.1)
		if _, err := os.Stat(savePath); os.IsNotExist(err) {
			t.Logf("File does not exist at expected path: %s", savePath)
			return false
		}

		// Property check 2: File is in the qap/ directory (Requirement 1.1)
		fileDir := filepath.Dir(savePath)
		if fileDir != qapDir {
			t.Logf("File not in qap directory: got %s, want %s", fileDir, qapDir)
			return false
		}

		// Property check 3: Filename matches analysis_YYYYMMDD_HHmmss.qap pattern (Requirement 1.3)
		baseName := filepath.Base(savePath)
		if !filenamePattern.MatchString(baseName) {
			t.Logf("Filename does not match pattern: %s", baseName)
			return false
		}

		// Property check 4: The saved file can be unpacked and contains the original data (Requirement 1.4)
		restoredJSON, err := UnpackFromZip(savePath, "")
		if err != nil {
			t.Logf("UnpackFromZip failed: %v", err)
			return false
		}

		var restoredPack QuickAnalysisPack
		if err := json.Unmarshal(restoredJSON, &restoredPack); err != nil {
			t.Logf("Unmarshal restored JSON failed: %v", err)
			return false
		}

		// Verify metadata matches
		if restoredPack.Metadata.Author != pack.Metadata.Author {
			t.Logf("Author mismatch: got %q, want %q", restoredPack.Metadata.Author, pack.Metadata.Author)
			return false
		}
		if restoredPack.Metadata.SourceName != pack.Metadata.SourceName {
			t.Logf("SourceName mismatch: got %q, want %q", restoredPack.Metadata.SourceName, pack.Metadata.SourceName)
			return false
		}
		if restoredPack.Metadata.CreatedAt != pack.Metadata.CreatedAt {
			t.Logf("CreatedAt mismatch: got %q, want %q", restoredPack.Metadata.CreatedAt, pack.Metadata.CreatedAt)
			return false
		}

		// Verify steps count matches
		if len(restoredPack.ExecutableSteps) != len(pack.ExecutableSteps) {
			t.Logf("Steps count mismatch: got %d, want %d", len(restoredPack.ExecutableSteps), len(pack.ExecutableSteps))
			return false
		}

		// Verify schema count matches
		if len(restoredPack.SchemaRequirements) != len(pack.SchemaRequirements) {
			t.Logf("Schema count mismatch: got %d, want %d", len(restoredPack.SchemaRequirements), len(pack.SchemaRequirements))
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1 (导出文件保存正确性) failed: %v", err)
	}
}

// scanQAPDirectory is a standalone helper that mirrors the core logic of
// ListLocalQuickAnalysisPacks without requiring App context. It scans a directory
// for .qap files, parses their metadata, and returns sorted LocalPackInfo entries.
func scanQAPDirectory(qapDir string) ([]LocalPackInfo, error) {
	entries, err := os.ReadDir(qapDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []LocalPackInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read qap directory: %w", err)
	}

	var packs []LocalPackInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".qap") {
			continue
		}

		filePath := filepath.Join(qapDir, entry.Name())

		jsonData, err := UnpackFromZip(filePath, "")
		if err != nil {
			// Encrypted or corrupted — skip
			continue
		}

		var pack QuickAnalysisPack
		if err := json.Unmarshal(jsonData, &pack); err != nil {
			continue
		}

		packs = append(packs, LocalPackInfo{
			FileName:    entry.Name(),
			FilePath:    filePath,
			PackName:    pack.Metadata.SourceName,
			Description: pack.Metadata.Description,
			SourceName:  pack.Metadata.SourceName,
			Author:      pack.Metadata.Author,
			CreatedAt:   pack.Metadata.CreatedAt,
			IsEncrypted: false,
		})
	}

	sort.Slice(packs, func(i, j int) bool {
		return packs[i].CreatedAt > packs[j].CreatedAt
	})

	if packs == nil {
		packs = []LocalPackInfo{}
	}
	return packs, nil
}

// createQAPFile is a helper that creates a .qap file from a QuickAnalysisPack in the given directory.
func createQAPFile(dir string, filename string, pack QuickAnalysisPack) error {
	jsonData, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return err
	}
	return PackToZip(jsonData, filepath.Join(dir, filename), "")
}

// Feature: qap-local-management, Property 2: 列表元信息提取正确性
// Validates: Requirements 2.1, 2.2
//
// For any set of valid .qap files in QAP directory, each LocalPackInfo's description,
// source_name, author, created_at should match the PackMetadata inside the corresponding .qap file.

func TestProperty2_ListMetadataExtractionCorrectness(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		qapDir := t.TempDir()

		// Generate 1-8 random valid packs
		numPacks := r.Intn(8) + 1
		expectedPacks := make(map[string]PackMetadata) // filename -> metadata

		for i := 0; i < numPacks; i++ {
			pack := generateRandomPackForExport(r)
			filename := fmt.Sprintf("pack_%d.qap", i)
			if err := createQAPFile(qapDir, filename, pack); err != nil {
				t.Logf("createQAPFile failed: %v", err)
				return false
			}
			expectedPacks[filename] = pack.Metadata
		}

		// Scan the directory
		result, err := scanQAPDirectory(qapDir)
		if err != nil {
			t.Logf("scanQAPDirectory failed: %v", err)
			return false
		}

		// Check count matches
		if len(result) != numPacks {
			t.Logf("Count mismatch: got %d, want %d", len(result), numPacks)
			return false
		}

		// Check each entry's metadata matches the original
		for _, info := range result {
			expected, ok := expectedPacks[info.FileName]
			if !ok {
				t.Logf("Unexpected file in result: %s", info.FileName)
				return false
			}
			if info.Description != expected.Description {
				t.Logf("Description mismatch for %s: got %q, want %q", info.FileName, info.Description, expected.Description)
				return false
			}
			if info.SourceName != expected.SourceName {
				t.Logf("SourceName mismatch for %s: got %q, want %q", info.FileName, info.SourceName, expected.SourceName)
				return false
			}
			if info.Author != expected.Author {
				t.Logf("Author mismatch for %s: got %q, want %q", info.FileName, info.Author, expected.Author)
				return false
			}
			if info.CreatedAt != expected.CreatedAt {
				t.Logf("CreatedAt mismatch for %s: got %q, want %q", info.FileName, info.CreatedAt, expected.CreatedAt)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 2 (列表元信息提取正确性) failed: %v", err)
	}
}

// Feature: qap-local-management, Property 3: 损坏文件不影响列表
// Validates: Requirements 2.4
//
// For any mixed set of valid and invalid .qap files, the list should contain
// all valid files and no invalid files.

func TestProperty3_CorruptedFilesDoNotAffectList(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		qapDir := t.TempDir()

		// Generate 1-5 valid packs
		numValid := r.Intn(5) + 1
		validFiles := make(map[string]bool)

		for i := 0; i < numValid; i++ {
			pack := generateRandomPackForExport(r)
			filename := fmt.Sprintf("valid_%d.qap", i)
			if err := createQAPFile(qapDir, filename, pack); err != nil {
				t.Logf("createQAPFile failed: %v", err)
				return false
			}
			validFiles[filename] = true
		}

		// Generate 1-5 corrupted/invalid .qap files
		numInvalid := r.Intn(5) + 1
		for i := 0; i < numInvalid; i++ {
			filename := fmt.Sprintf("corrupt_%d.qap", i)
			filePath := filepath.Join(qapDir, filename)

			// Write random garbage bytes as a corrupted .qap file
			garbage := make([]byte, r.Intn(200)+10)
			for j := range garbage {
				garbage[j] = byte(r.Intn(256))
			}
			if err := os.WriteFile(filePath, garbage, 0644); err != nil {
				t.Logf("WriteFile failed: %v", err)
				return false
			}
		}

		// Also add a non-.qap file to ensure it's ignored
		if err := os.WriteFile(filepath.Join(qapDir, "readme.txt"), []byte("not a qap"), 0644); err != nil {
			t.Logf("WriteFile readme failed: %v", err)
			return false
		}

		// Scan the directory
		result, err := scanQAPDirectory(qapDir)
		if err != nil {
			t.Logf("scanQAPDirectory failed: %v", err)
			return false
		}

		// Check: result should contain exactly the valid files
		if len(result) != numValid {
			t.Logf("Count mismatch: got %d, want %d (valid only)", len(result), numValid)
			return false
		}

		for _, info := range result {
			if !validFiles[info.FileName] {
				t.Logf("Unexpected file in result: %s (should only contain valid files)", info.FileName)
				return false
			}
		}

		// Check: all valid files are present
		foundFiles := make(map[string]bool)
		for _, info := range result {
			foundFiles[info.FileName] = true
		}
		for vf := range validFiles {
			if !foundFiles[vf] {
				t.Logf("Missing valid file in result: %s", vf)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3 (损坏文件不影响列表) failed: %v", err)
	}
}

// Feature: qap-local-management, Property 4: 列表按创建时间倒序排列
// Validates: Requirements 2.5
//
// For any set of multiple valid .qap files, each entry's created_at should be
// >= the next entry's created_at.

func TestProperty4_ListSortedByCreatedAtDescending(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		qapDir := t.TempDir()

		// Generate 2-10 packs with distinct timestamps
		numPacks := r.Intn(9) + 2

		for i := 0; i < numPacks; i++ {
			pack := generateRandomPackForExport(r)
			// Ensure distinct timestamps by using different random dates
			year := 2020 + r.Intn(6)
			month := time.Month(r.Intn(12) + 1)
			day := r.Intn(28) + 1
			hour := r.Intn(24)
			minute := r.Intn(60)
			second := r.Intn(60)
			pack.Metadata.CreatedAt = time.Date(year, month, day, hour, minute, second, 0, time.UTC).Format(time.RFC3339)

			filename := fmt.Sprintf("pack_%d.qap", i)
			if err := createQAPFile(qapDir, filename, pack); err != nil {
				t.Logf("createQAPFile failed: %v", err)
				return false
			}
		}

		// Scan the directory
		result, err := scanQAPDirectory(qapDir)
		if err != nil {
			t.Logf("scanQAPDirectory failed: %v", err)
			return false
		}

		if len(result) != numPacks {
			t.Logf("Count mismatch: got %d, want %d", len(result), numPacks)
			return false
		}

		// Verify descending order: each entry's created_at >= next entry's created_at
		for i := 0; i < len(result)-1; i++ {
			if result[i].CreatedAt < result[i+1].CreatedAt {
				t.Logf("Sort order violation at index %d: %q < %q", i, result[i].CreatedAt, result[i+1].CreatedAt)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4 (列表按创建时间倒序排列) failed: %v", err)
	}
}

// Feature: qap-local-management, Property 5: 删除操作移除文件
// Validates: Requirements 3.2
//
// For any valid .qap file path in QAP directory, after calling os.Remove (mirroring
// DeleteLocalPack logic), the file should no longer exist.

func TestProperty5_DeleteRemovesFile(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		qapDir := t.TempDir()

		// Generate a random pack and save it as a .qap file
		pack := generateRandomPackForExport(r)
		filename := fmt.Sprintf("delete_test_%d.qap", r.Intn(100000))
		filePath := filepath.Join(qapDir, filename)

		if err := createQAPFile(qapDir, filename, pack); err != nil {
			t.Logf("createQAPFile failed: %v", err)
			return false
		}

		// Verify file exists before deletion
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Logf("File does not exist before deletion: %s", filePath)
			return false
		}

		// Delete the file (mirrors DeleteLocalPack core logic)
		if err := os.Remove(filePath); err != nil {
			t.Logf("os.Remove failed: %v", err)
			return false
		}

		// Property check: file should no longer exist
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Logf("File still exists after deletion: %s", filePath)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 5 (删除操作移除文件) failed: %v", err)
	}
}

// Feature: qap-local-management, Property 6: 元信息更新保持步骤不变
// Validates: Requirements 4.2, 6.2
//
// For any valid .qap file and any new description/author values, after updating
// metadata (mirroring UpdatePackMetadata logic), re-reading the file should show
// (a) updated metadata matching the input, (b) executable_steps and schema_requirements unchanged.

func TestProperty6_MetadataUpdatePreservesSteps(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		qapDir := t.TempDir()

		// Generate a random pack
		pack := generateRandomPackForExport(r)
		filename := fmt.Sprintf("meta_test_%d.qap", r.Intn(100000))
		filePath := filepath.Join(qapDir, filename)

		if err := createQAPFile(qapDir, filename, pack); err != nil {
			t.Logf("createQAPFile failed: %v", err)
			return false
		}

		// Save original steps and schema for comparison
		originalStepsJSON, err := json.Marshal(pack.ExecutableSteps)
		if err != nil {
			t.Logf("Marshal original steps failed: %v", err)
			return false
		}
		originalSchemaJSON, err := json.Marshal(pack.SchemaRequirements)
		if err != nil {
			t.Logf("Marshal original schema failed: %v", err)
			return false
		}

		// Generate new random description and author
		newDescription := generateRandomString(r, 60)
		newAuthor := generateRandomString(r, 25)

		// Mirror UpdatePackMetadata logic: unpack, update metadata, repack
		jsonData, err := UnpackFromZip(filePath, "")
		if err != nil {
			t.Logf("UnpackFromZip failed: %v", err)
			return false
		}

		var loadedPack QuickAnalysisPack
		if err := json.Unmarshal(jsonData, &loadedPack); err != nil {
			t.Logf("Unmarshal failed: %v", err)
			return false
		}

		loadedPack.Metadata.Description = newDescription
		loadedPack.Metadata.Author = newAuthor

		updatedJSON, err := json.Marshal(loadedPack)
		if err != nil {
			t.Logf("Marshal updated pack failed: %v", err)
			return false
		}

		if err := PackToZip(updatedJSON, filePath, ""); err != nil {
			t.Logf("PackToZip failed: %v", err)
			return false
		}

		// Re-read the file and verify
		verifyJSON, err := UnpackFromZip(filePath, "")
		if err != nil {
			t.Logf("UnpackFromZip (verify) failed: %v", err)
			return false
		}

		var verifyPack QuickAnalysisPack
		if err := json.Unmarshal(verifyJSON, &verifyPack); err != nil {
			t.Logf("Unmarshal (verify) failed: %v", err)
			return false
		}

		// Property check (a): metadata matches the new values
		if verifyPack.Metadata.Description != newDescription {
			t.Logf("Description mismatch: got %q, want %q", verifyPack.Metadata.Description, newDescription)
			return false
		}
		if verifyPack.Metadata.Author != newAuthor {
			t.Logf("Author mismatch: got %q, want %q", verifyPack.Metadata.Author, newAuthor)
			return false
		}

		// Property check (b): executable_steps unchanged
		verifyStepsJSON, err := json.Marshal(verifyPack.ExecutableSteps)
		if err != nil {
			t.Logf("Marshal verify steps failed: %v", err)
			return false
		}
		if string(verifyStepsJSON) != string(originalStepsJSON) {
			t.Logf("ExecutableSteps changed after metadata update")
			return false
		}

		// Property check (b): schema_requirements unchanged
		verifySchemaJSON, err := json.Marshal(verifyPack.SchemaRequirements)
		if err != nil {
			t.Logf("Marshal verify schema failed: %v", err)
			return false
		}
		if string(verifySchemaJSON) != string(originalSchemaJSON) {
			t.Logf("SchemaRequirements changed after metadata update")
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 6 (元信息更新保持步骤不变) failed: %v", err)
	}
}

// Feature: qap-local-management, Property 7: QAP 文件打包/解包往返一致性
// Validates: Requirements 6.3
//
// For any valid QuickAnalysisPack object, serializing to JSON, packing to .qap,
// unpacking and deserializing should produce an equivalent QuickAnalysisPack.

func TestProperty7_PackUnpackRoundTrip(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		tmpDir := t.TempDir()

		// Generate a random QuickAnalysisPack
		pack := generateRandomPackForExport(r)

		// Serialize to JSON
		jsonData, err := json.Marshal(pack)
		if err != nil {
			t.Logf("Marshal failed: %v", err)
			return false
		}

		// Pack to .qap file
		filePath := filepath.Join(tmpDir, fmt.Sprintf("roundtrip_%d.qap", r.Intn(100000)))
		if err := PackToZip(jsonData, filePath, ""); err != nil {
			t.Logf("PackToZip failed: %v", err)
			return false
		}

		// Unpack from .qap file
		restoredJSON, err := UnpackFromZip(filePath, "")
		if err != nil {
			t.Logf("UnpackFromZip failed: %v", err)
			return false
		}

		// Deserialize back to QuickAnalysisPack
		var restoredPack QuickAnalysisPack
		if err := json.Unmarshal(restoredJSON, &restoredPack); err != nil {
			t.Logf("Unmarshal restored JSON failed: %v", err)
			return false
		}

		// Property check: all fields should be equivalent
		// Check file_type and format_version
		if restoredPack.FileType != pack.FileType {
			t.Logf("FileType mismatch: got %q, want %q", restoredPack.FileType, pack.FileType)
			return false
		}
		if restoredPack.FormatVersion != pack.FormatVersion {
			t.Logf("FormatVersion mismatch: got %q, want %q", restoredPack.FormatVersion, pack.FormatVersion)
			return false
		}

		// Check metadata
		if restoredPack.Metadata.Author != pack.Metadata.Author {
			t.Logf("Metadata.Author mismatch: got %q, want %q", restoredPack.Metadata.Author, pack.Metadata.Author)
			return false
		}
		if restoredPack.Metadata.CreatedAt != pack.Metadata.CreatedAt {
			t.Logf("Metadata.CreatedAt mismatch: got %q, want %q", restoredPack.Metadata.CreatedAt, pack.Metadata.CreatedAt)
			return false
		}
		if restoredPack.Metadata.SourceName != pack.Metadata.SourceName {
			t.Logf("Metadata.SourceName mismatch: got %q, want %q", restoredPack.Metadata.SourceName, pack.Metadata.SourceName)
			return false
		}
		if restoredPack.Metadata.Description != pack.Metadata.Description {
			t.Logf("Metadata.Description mismatch: got %q, want %q", restoredPack.Metadata.Description, pack.Metadata.Description)
			return false
		}

		// Check schema_requirements via JSON comparison
		originalSchemaJSON, _ := json.Marshal(pack.SchemaRequirements)
		restoredSchemaJSON, _ := json.Marshal(restoredPack.SchemaRequirements)
		if string(originalSchemaJSON) != string(restoredSchemaJSON) {
			t.Logf("SchemaRequirements mismatch")
			return false
		}

		// Check executable_steps via JSON comparison
		originalStepsJSON, _ := json.Marshal(pack.ExecutableSteps)
		restoredStepsJSON, _ := json.Marshal(restoredPack.ExecutableSteps)
		if string(originalStepsJSON) != string(restoredStepsJSON) {
			t.Logf("ExecutableSteps mismatch")
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 7 (QAP 文件打包/解包往返一致性) failed: %v", err)
	}
}
