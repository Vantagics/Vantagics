package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"
)

// Feature: pack-result-display, Property 1: Step result auto-collection completeness
// Validates: Requirements 1.1, 1.2

// newTestAppWithLongFlush creates a test App with an EventAggregator that has a very long
// flush delay, preventing timer-triggered Wails EventsEmit calls during property tests.
func newTestAppWithLongFlush() *App {
	app := newTestAppWithAggregator()
	// Set a very long flush delay so the timer never fires during tests.
	// This avoids the Wails context panic from runtime.EventsEmit.
	app.eventAggregator.flushDelay = 24 * time.Hour
	return app
}

// randomFileName generates a random file name with the given extension.
func randomFileName(r *rand.Rand, ext string) string {
	n := r.Intn(10) + 1
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte('a' + r.Intn(26))
	}
	return string(buf) + ext
}

// randomNonImageExt returns a random non-image file extension.
func randomNonImageExt(r *rand.Rand) string {
	exts := []string{".py", ".txt", ".csv", ".json", ".go", ".html", ".md", ".log", ".xml", ".sql"}
	return exts[r.Intn(len(exts))]
}

// randomImageExt returns a random image file extension (.png, .jpg, .jpeg).
func randomImageExt(r *rand.Rand) string {
	exts := []string{".png", ".jpg", ".jpeg"}
	return exts[r.Intn(len(exts))]
}

// generateValidEChartsJSON generates a valid JSON string that looks like an ECharts config.
func generateValidEChartsJSON(r *rand.Rand) string {
	titles := []string{"Sales", "Revenue", "Users", "Growth", "Trend"}
	types := []string{"bar", "line", "pie", "scatter"}
	numData := r.Intn(5) + 1
	data := make([]int, numData)
	for i := range data {
		data[i] = r.Intn(1000)
	}
	dataJSON, _ := json.Marshal(data)
	return fmt.Sprintf(`{"title":{"text":"%s"},"series":[{"type":"%s","data":%s}]}`,
		titles[r.Intn(len(titles))], types[r.Intn(len(types))], string(dataJSON))
}

// generateInvalidJSON generates a string that is NOT valid JSON.
func generateInvalidJSON(r *rand.Rand) string {
	invalids := []string{
		"not-json",
		"{broken",
		"[1,2,",
		"{'single': 'quotes'}",
		"undefined",
		"{key: value}",
	}
	return invalids[r.Intn(len(invalids))]
}


// TestProperty1a_ChartFileDetectionCompleteness verifies that detectAndSendPythonChartFiles
// returns exactly one AnalysisResultItem of type "image" per image file (.png, .jpg, .jpeg)
// and zero for non-image files.
// **Validates: Requirements 1.1, 1.2**
func TestProperty1a_ChartFileDetectionCompleteness(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		app := newTestAppWithLongFlush()
		workDir := t.TempDir()

		numImageFiles := r.Intn(6)
		numNonImageFiles := r.Intn(6)

		imageFileNames := make(map[string]bool)
		for i := 0; i < numImageFiles; i++ {
			name := randomFileName(r, randomImageExt(r))
			for imageFileNames[name] {
				name = randomFileName(r, randomImageExt(r))
			}
			imageFileNames[name] = true
			os.WriteFile(filepath.Join(workDir, name), []byte("fake-image-data"), 0644)
		}

		for i := 0; i < numNonImageFiles; i++ {
			name := randomFileName(r, randomNonImageExt(r))
			for imageFileNames[name] {
				name = randomFileName(r, randomNonImageExt(r))
			}
			os.WriteFile(filepath.Join(workDir, name), []byte("not-an-image"), 0644)
		}

		results := app.detectAndSendPythonChartFiles("thread1", "msg1", workDir, "")

		if len(results) != numImageFiles {
			t.Logf("seed=%d: expected %d image results, got %d (images=%d, non-images=%d)",
				seed, numImageFiles, len(results), numImageFiles, numNonImageFiles)
			return false
		}

		for _, r := range results {
			if r.Type != "image" {
				t.Logf("seed=%d: expected type 'image', got '%s'", seed, r.Type)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1a (chart file detection completeness) failed: %v", err)
	}
}

// TestProperty1b_EChartsDetectionCompleteness verifies that detectAndSendPythonECharts
// returns exactly N AnalysisResultItem of type "echarts" for output containing N valid
// json:echarts blocks.
// **Validates: Requirements 1.1, 1.2**
func TestProperty1b_EChartsDetectionCompleteness(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		app := newTestAppWithLongFlush()

		numBlocks := r.Intn(6)
		var parts []string
		parts = append(parts, "Python output start")

		for i := 0; i < numBlocks; i++ {
			chartJSON := generateValidEChartsJSON(r)
			parts = append(parts, fmt.Sprintf("```json:echarts\n%s\n```", chartJSON))
			parts = append(parts, fmt.Sprintf("Some text between blocks %d", i))
		}

		parts = append(parts, "Python output end")
		output := strings.Join(parts, "\n")

		results := app.detectAndSendPythonECharts("thread1", "msg1", output, "")

		if len(results) != numBlocks {
			t.Logf("seed=%d: expected %d echarts results, got %d", seed, numBlocks, len(results))
			return false
		}

		for _, r := range results {
			if r.Type != "echarts" {
				t.Logf("seed=%d: expected type 'echarts', got '%s'", seed, r.Type)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1b (ECharts detection completeness) failed: %v", err)
	}
}

// TestProperty1c_InvalidEChartsSkipped verifies that invalid JSON in echarts blocks
// produces zero results.
// **Validates: Requirements 1.1, 1.2**
func TestProperty1c_InvalidEChartsSkipped(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		app := newTestAppWithLongFlush()

		numInvalid := r.Intn(5) + 1
		var parts []string
		parts = append(parts, "Python output start")

		for i := 0; i < numInvalid; i++ {
			invalidJSON := generateInvalidJSON(r)
			parts = append(parts, fmt.Sprintf("```json:echarts\n%s\n```", invalidJSON))
		}

		parts = append(parts, "Python output end")
		output := strings.Join(parts, "\n")

		results := app.detectAndSendPythonECharts("thread1", "msg1", output, "")

		if len(results) != 0 {
			t.Logf("seed=%d: expected 0 results for invalid JSON, got %d", seed, len(results))
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 1c (invalid ECharts skipped) failed: %v", err)
	}
}

// Feature: pack-result-display, Property 3: Step result re-push data consistency
// Validates: Requirements 4.1, 4.2, 4.4

// newTestAppWithChatService creates a test App with both EventAggregator and ChatService
// using a temporary directory for chat storage.
func newTestAppWithChatService(tmpDir string) *App {
	app := newTestAppWithLongFlush()
	app.chatService = NewChatService(filepath.Join(tmpDir, "sessions"))
	return app
}

// randomAnalysisResultType returns a random analysis result type from table, echarts, image.
func randomAnalysisResultType(r *rand.Rand) string {
	types := []string{"table", "echarts", "image"}
	return types[r.Intn(len(types))]
}

// randomTableData generates random table data as a map.
func randomTableData(r *rand.Rand) interface{} {
	numCols := r.Intn(5) + 1
	numRows := r.Intn(10) + 1
	columns := make([]string, numCols)
	for i := range columns {
		columns[i] = fmt.Sprintf("col_%d", i)
	}
	rows := make([]map[string]interface{}, numRows)
	for i := range rows {
		row := make(map[string]interface{})
		for _, col := range columns {
			row[col] = r.Intn(10000)
		}
		rows[i] = row
	}
	return map[string]interface{}{
		"columns": columns,
		"rows":    rows,
	}
}

// randomEChartsData generates random ECharts configuration data as a map.
func randomEChartsData(r *rand.Rand) interface{} {
	titles := []string{"Sales", "Revenue", "Users", "Growth", "Trend"}
	types := []string{"bar", "line", "pie", "scatter"}
	numData := r.Intn(8) + 1
	data := make([]int, numData)
	for i := range data {
		data[i] = r.Intn(1000)
	}
	return map[string]interface{}{
		"title":  map[string]interface{}{"text": titles[r.Intn(len(titles))]},
		"series": []interface{}{map[string]interface{}{"type": types[r.Intn(len(types))], "data": data}},
	}
}

// randomImageData generates a random base64-like image data string.
func randomImageData(r *rand.Rand) interface{} {
	n := r.Intn(50) + 10
	buf := make([]byte, n)
	for i := range buf {
		buf[i] = byte('A' + r.Intn(26))
	}
	return "data:image/png;base64," + string(buf)
}

// randomMetadata generates random metadata for an analysis result item.
func randomMetadata(r *rand.Rand) map[string]interface{} {
	meta := make(map[string]interface{})
	if r.Intn(2) == 0 {
		meta["title"] = fmt.Sprintf("Result_%d", r.Intn(100))
	}
	if r.Intn(2) == 0 {
		meta["step"] = r.Intn(10)
	}
	return meta
}

// generateRandomAnalysisResults generates a random slice of AnalysisResultItems.
func generateRandomAnalysisResults(r *rand.Rand) []AnalysisResultItem {
	n := r.Intn(8) + 1 // 1-8 items
	items := make([]AnalysisResultItem, n)
	for i := range items {
		itemType := randomAnalysisResultType(r)
		var data interface{}
		switch itemType {
		case "table":
			data = randomTableData(r)
		case "echarts":
			data = randomEChartsData(r)
		case "image":
			data = randomImageData(r)
		}
		items[i] = AnalysisResultItem{
			ID:       fmt.Sprintf("item_%d_%d", i, r.Intn(10000)),
			Type:     itemType,
			Data:     data,
			Metadata: randomMetadata(r),
		}
	}
	return items
}

// TestProperty3a_SaveRetrieveDataConsistency verifies that for any set of
// AnalysisResultItems saved to a message via ChatService.SaveAnalysisResults,
// the data retrieved via ChatService.GetMessageAnalysisData preserves the exact
// count and types of all items. This is the core data path that
// ShowStepResultOnDashboard relies on for re-pushing results.
// **Validates: Requirements 4.1, 4.2, 4.4**
func TestProperty3a_SaveRetrieveDataConsistency(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		tmpDir := t.TempDir()
		app := newTestAppWithChatService(tmpDir)

		// Create a thread and add a message
		thread, err := app.chatService.CreateThread("ds1", "test-thread")
		if err != nil {
			t.Logf("seed=%d: failed to create thread: %v", seed, err)
			return false
		}

		msgID := fmt.Sprintf("msg_%d", r.Intn(100000))
		msg := ChatMessage{
			ID:        msgID,
			Role:      "assistant",
			Content:   "✅ Step result content",
			Timestamp: time.Now().Unix(),
		}
		if err := app.chatService.AddMessage(thread.ID, msg); err != nil {
			t.Logf("seed=%d: failed to add message: %v", seed, err)
			return false
		}

		// Generate and save random analysis results
		savedResults := generateRandomAnalysisResults(r)
		if err := app.chatService.SaveAnalysisResults(thread.ID, msgID, savedResults); err != nil {
			t.Logf("seed=%d: failed to save analysis results: %v", seed, err)
			return false
		}

		// Retrieve via GetMessageAnalysisData (same path ShowStepResultOnDashboard uses)
		analysisData, err := app.chatService.GetMessageAnalysisData(thread.ID, msgID)
		if err != nil {
			t.Logf("seed=%d: GetMessageAnalysisData returned error: %v", seed, err)
			return false
		}

		// Extract analysisResults from the returned map
		rawItems, ok := analysisData["analysisResults"]
		if !ok || rawItems == nil {
			t.Logf("seed=%d: analysisResults not found in returned data", seed)
			return false
		}

		retrievedItems, ok := rawItems.([]AnalysisResultItem)
		if !ok {
			t.Logf("seed=%d: analysisResults is not []AnalysisResultItem", seed)
			return false
		}

		// Property: count must match
		if len(retrievedItems) != len(savedResults) {
			t.Logf("seed=%d: count mismatch: saved=%d, retrieved=%d",
				seed, len(savedResults), len(retrievedItems))
			return false
		}

		// Property: type distribution must match
		savedTypeCounts := make(map[string]int)
		for _, item := range savedResults {
			savedTypeCounts[item.Type]++
		}
		retrievedTypeCounts := make(map[string]int)
		for _, item := range retrievedItems {
			retrievedTypeCounts[item.Type]++
		}
		if !reflect.DeepEqual(savedTypeCounts, retrievedTypeCounts) {
			t.Logf("seed=%d: type distribution mismatch: saved=%v, retrieved=%v",
				seed, savedTypeCounts, retrievedTypeCounts)
			return false
		}

		// Property: each item's ID and Type must be preserved in order
		for i, saved := range savedResults {
			if retrievedItems[i].ID != saved.ID {
				t.Logf("seed=%d: item %d ID mismatch: saved=%s, retrieved=%s",
					seed, i, saved.ID, retrievedItems[i].ID)
				return false
			}
			if retrievedItems[i].Type != saved.Type {
				t.Logf("seed=%d: item %d Type mismatch: saved=%s, retrieved=%s",
					seed, i, saved.Type, retrievedItems[i].Type)
				return false
			}
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3a (save-retrieve data consistency) failed: %v", err)
	}
}

// TestProperty3b_RePushFailsForNonexistentMessage verifies that ShowStepResultOnDashboard
// returns an error when the message does not exist in the thread.
// This tests the error path which returns before reaching EventAggregator.FlushNow,
// so it is safe to call directly without Wails context.
// **Validates: Requirements 4.1, 4.2, 4.4**
func TestProperty3b_RePushFailsForNonexistentMessage(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		tmpDir := t.TempDir()
		app := newTestAppWithChatService(tmpDir)

		// Create a thread but do NOT add any message
		thread, err := app.chatService.CreateThread("ds1", "test-thread")
		if err != nil {
			t.Logf("seed=%d: failed to create thread: %v", seed, err)
			return false
		}

		// Generate a random non-existent message ID
		fakeMsgID := fmt.Sprintf("nonexistent_%d", r.Intn(100000))

		// Call ShowStepResultOnDashboard - should return error before reaching FlushNow
		err = app.ShowStepResultOnDashboard(thread.ID, fakeMsgID)

		// Property: must return an error for non-existent message
		if err == nil {
			t.Logf("seed=%d: expected error for non-existent message, got nil", seed)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3b (re-push fails for non-existent message) failed: %v", err)
	}
}

// TestProperty3c_RePushFailsForMessageWithNoResults verifies that ShowStepResultOnDashboard
// returns an error when the message exists but has no analysis results.
// This tests the error path which returns before reaching EventAggregator.FlushNow.
// **Validates: Requirements 4.1, 4.2, 4.4**
func TestProperty3c_RePushFailsForMessageWithNoResults(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		tmpDir := t.TempDir()
		app := newTestAppWithChatService(tmpDir)

		// Create a thread and add a message with NO analysis results
		thread, err := app.chatService.CreateThread("ds1", "test-thread")
		if err != nil {
			t.Logf("seed=%d: failed to create thread: %v", seed, err)
			return false
		}

		msgID := fmt.Sprintf("msg_%d", r.Intn(100000))
		msg := ChatMessage{
			ID:        msgID,
			Role:      "assistant",
			Content:   fmt.Sprintf("Plain message content %d", r.Intn(1000)),
			Timestamp: time.Now().Unix(),
		}
		if err := app.chatService.AddMessage(thread.ID, msg); err != nil {
			t.Logf("seed=%d: failed to add message: %v", seed, err)
			return false
		}

		// Do NOT save any analysis results

		// Call ShowStepResultOnDashboard - should return error before reaching FlushNow
		err = app.ShowStepResultOnDashboard(thread.ID, msgID)

		// Property: must return an error when message has no results
		if err == nil {
			t.Logf("seed=%d: expected error for message with no results, got nil", seed)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3c (re-push fails for message with no results) failed: %v", err)
	}
}

// TestProperty3d_RePushDataTypeConsistency verifies that for any set of analysis results
// with mixed types (table, echarts, image), the data retrieved from storage preserves
// the exact type distribution. This validates the data consistency property that
// ShowStepResultOnDashboard depends on: the data it reads and pushes to EventAggregator
// must match what was originally saved.
// **Validates: Requirements 4.1, 4.2, 4.4**
func TestProperty3d_RePushDataTypeConsistency(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))
		tmpDir := t.TempDir()
		app := newTestAppWithChatService(tmpDir)

		// Create a thread and add a message
		thread, err := app.chatService.CreateThread("ds1", "test-thread")
		if err != nil {
			t.Logf("seed=%d: failed to create thread: %v", seed, err)
			return false
		}

		msgID := fmt.Sprintf("msg_%d", r.Intn(100000))
		msg := ChatMessage{
			ID:        msgID,
			Role:      "assistant",
			Content:   "✅ Step with mixed results",
			Timestamp: time.Now().Unix(),
		}
		if err := app.chatService.AddMessage(thread.ID, msg); err != nil {
			t.Logf("seed=%d: failed to add message: %v", seed, err)
			return false
		}

		// Generate results with specific type distribution
		numTable := r.Intn(4)
		numECharts := r.Intn(4)
		numImage := r.Intn(4)
		// Ensure at least one item
		if numTable+numECharts+numImage == 0 {
			numTable = 1
		}

		var results []AnalysisResultItem
		for i := 0; i < numTable; i++ {
			results = append(results, AnalysisResultItem{
				ID:       fmt.Sprintf("table_%d", i),
				Type:     "table",
				Data:     randomTableData(r),
				Metadata: randomMetadata(r),
			})
		}
		for i := 0; i < numECharts; i++ {
			results = append(results, AnalysisResultItem{
				ID:       fmt.Sprintf("echarts_%d", i),
				Type:     "echarts",
				Data:     randomEChartsData(r),
				Metadata: randomMetadata(r),
			})
		}
		for i := 0; i < numImage; i++ {
			results = append(results, AnalysisResultItem{
				ID:       fmt.Sprintf("image_%d", i),
				Type:     "image",
				Data:     randomImageData(r),
				Metadata: randomMetadata(r),
			})
		}

		if err := app.chatService.SaveAnalysisResults(thread.ID, msgID, results); err != nil {
			t.Logf("seed=%d: failed to save analysis results: %v", seed, err)
			return false
		}

		// Retrieve via GetMessageAnalysisData (same path ShowStepResultOnDashboard uses)
		analysisData, err := app.chatService.GetMessageAnalysisData(thread.ID, msgID)
		if err != nil {
			t.Logf("seed=%d: GetMessageAnalysisData returned error: %v", seed, err)
			return false
		}

		rawItems, ok := analysisData["analysisResults"]
		if !ok || rawItems == nil {
			t.Logf("seed=%d: analysisResults not found", seed)
			return false
		}

		retrievedItems, ok := rawItems.([]AnalysisResultItem)
		if !ok {
			t.Logf("seed=%d: analysisResults is not []AnalysisResultItem", seed)
			return false
		}

		// Count types in retrieved items
		retrievedTableCount := 0
		retrievedEChartsCount := 0
		retrievedImageCount := 0
		for _, item := range retrievedItems {
			switch item.Type {
			case "table":
				retrievedTableCount++
			case "echarts":
				retrievedEChartsCount++
			case "image":
				retrievedImageCount++
			}
		}

		// Property: type distribution must be preserved exactly
		if retrievedTableCount != numTable {
			t.Logf("seed=%d: table count mismatch: saved=%d, retrieved=%d",
				seed, numTable, retrievedTableCount)
			return false
		}
		if retrievedEChartsCount != numECharts {
			t.Logf("seed=%d: echarts count mismatch: saved=%d, retrieved=%d",
				seed, numECharts, retrievedEChartsCount)
			return false
		}
		if retrievedImageCount != numImage {
			t.Logf("seed=%d: image count mismatch: saved=%d, retrieved=%d",
				seed, numImage, retrievedImageCount)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 3d (re-push data type consistency) failed: %v", err)
	}
}

// Feature: pack-result-display, Property 4: Replay session comprehensive report content extraction
// Validates: Requirements 5.2

// simulateComprehensiveReportExtraction 模拟 PrepareComprehensiveReport 中的消息遍历和内容提取逻辑。
// 它遍历线程消息，对每个非建议请求的 user 消息，提取用户请求内容，
// 并检查下一条 assistant 消息的内容（成功步骤包含 ✅，失败步骤包含 ❌）。
// 返回提取的分析内容条目数和成功步骤的 assistant 响应数。
func simulateComprehensiveReportExtraction(messages []ChatMessage) (totalContents int, successfulAssistantResponses int) {
	suggestionPatterns := []string{
		"请给出一些本数据源的分析建议",
		"Give me some analysis suggestions for this data source",
	}

	for i, msg := range messages {
		if msg.Role != "user" {
			continue
		}

		// 跳过建议请求消息
		isSuggestionRequest := false
		trimmedContent := strings.TrimSpace(msg.Content)
		for _, pattern := range suggestionPatterns {
			if strings.Contains(trimmedContent, pattern) {
				isSuggestionRequest = true
				break
			}
		}
		if isSuggestionRequest {
			continue
		}

		// 用户分析请求内容计入 analysisContents
		totalContents++

		// 检查下一条 assistant 消息
		if i+1 < len(messages) && messages[i+1].Role == "assistant" {
			assistantMsg := messages[i+1]
			if assistantMsg.Content != "" {
				// assistant 响应内容计入 analysisContents
				totalContents++
				// 统计成功步骤（包含 ✅ 且不包含 ❌）
				if strings.Contains(assistantMsg.Content, "✅") && !strings.Contains(assistantMsg.Content, "❌") {
					successfulAssistantResponses++
				}
			}
		}
	}
	return
}

// generateReplaySessionMessages 生成模拟 Replay_Session 的消息列表。
// 每个步骤由一对 user/assistant 消息组成。
// numSuccess 个步骤的 assistant 消息包含 ✅ 标识，
// numFailed 个步骤的 assistant 消息包含 ❌ 标识。
func generateReplaySessionMessages(r *rand.Rand, numSuccess, numFailed int) []ChatMessage {
	var messages []ChatMessage
	stepID := 1

	// 生成成功和失败步骤的索引，然后随机打乱
	type stepInfo struct {
		isSuccess bool
	}
	steps := make([]stepInfo, 0, numSuccess+numFailed)
	for i := 0; i < numSuccess; i++ {
		steps = append(steps, stepInfo{isSuccess: true})
	}
	for i := 0; i < numFailed; i++ {
		steps = append(steps, stepInfo{isSuccess: false})
	}
	// 随机打乱步骤顺序
	r.Shuffle(len(steps), func(i, j int) {
		steps[i], steps[j] = steps[j], steps[i]
	})

	for _, step := range steps {
		// 用户请求消息
		userMsg := ChatMessage{
			ID:        fmt.Sprintf("user_%d_%d", stepID, r.Intn(100000)),
			Role:      "user",
			Content:   fmt.Sprintf("执行步骤 %d 的分析查询", stepID),
			Timestamp: time.Now().Unix() + int64(stepID),
		}
		messages = append(messages, userMsg)

		// Assistant 响应消息
		var assistantContent string
		if step.isSuccess {
			// 成功步骤：包含 ✅ 标识和分析结果
			assistantContent = fmt.Sprintf("✅ 步骤 %d (分析查询 %d):\n\n```json:table\n[{\"col1\": %d}]\n```",
				stepID, stepID, r.Intn(1000))
		} else {
			// 失败步骤：包含 ❌ 标识和错误信息
			assistantContent = fmt.Sprintf("❌ 步骤 %d (分析查询 %d) 执行失败：模拟错误 %d",
				stepID, stepID, r.Intn(1000))
		}

		assistantMsg := ChatMessage{
			ID:        fmt.Sprintf("assistant_%d_%d", stepID, r.Intn(100000)),
			Role:      "assistant",
			Content:   assistantContent,
			Timestamp: time.Now().Unix() + int64(stepID) + 1,
		}
		messages = append(messages, assistantMsg)
		stepID++
	}

	return messages
}

// TestProperty4a_ReplaySessionContentExtractionCount 验证对于包含多个步骤消息的 Replay_Session，
// 内容提取逻辑从所有 user 消息中提取分析内容，且成功步骤的 assistant 响应数量
// 等于会话中成功步骤的数量（不包含失败步骤的错误消息）。
// **Validates: Requirements 5.2**
func TestProperty4a_ReplaySessionContentExtractionCount(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		numSuccess := r.Intn(6) + 1 // 1-6 个成功步骤
		numFailed := r.Intn(4)       // 0-3 个失败步骤

		messages := generateReplaySessionMessages(r, numSuccess, numFailed)

		totalContents, successfulResponses := simulateComprehensiveReportExtraction(messages)

		// 属性：成功步骤的 assistant 响应数量应等于 numSuccess
		if successfulResponses != numSuccess {
			t.Logf("seed=%d: 成功步骤响应数量不匹配: expected=%d, got=%d (numSuccess=%d, numFailed=%d)",
				seed, numSuccess, successfulResponses, numSuccess, numFailed)
			return false
		}

		// 属性：总提取内容数应等于 (numSuccess + numFailed) * 2
		// 每个步骤贡献 2 条内容：1 条用户请求 + 1 条 assistant 响应
		expectedTotal := (numSuccess + numFailed) * 2
		if totalContents != expectedTotal {
			t.Logf("seed=%d: 总内容数量不匹配: expected=%d, got=%d",
				seed, expectedTotal, totalContents)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4a (replay session content extraction count) failed: %v", err)
	}
}

// TestProperty4b_ReplaySessionSkipsSuggestionMessages 验证内容提取逻辑正确跳过
// 建议请求消息（自动生成的第一条消息），不将其计入分析内容。
// **Validates: Requirements 5.2**
func TestProperty4b_ReplaySessionSkipsSuggestionMessages(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		numSuccess := r.Intn(5) + 1
		numFailed := r.Intn(3)

		messages := generateReplaySessionMessages(r, numSuccess, numFailed)

		// 在消息列表开头插入建议请求消息对
		suggestionPatterns := []string{
			"请给出一些本数据源的分析建议",
			"Give me some analysis suggestions for this data source",
		}
		suggestionContent := suggestionPatterns[r.Intn(len(suggestionPatterns))]

		suggestionUser := ChatMessage{
			ID:        fmt.Sprintf("suggestion_user_%d", r.Intn(100000)),
			Role:      "user",
			Content:   suggestionContent,
			Timestamp: time.Now().Unix() - 100,
		}
		suggestionAssistant := ChatMessage{
			ID:        fmt.Sprintf("suggestion_assistant_%d", r.Intn(100000)),
			Role:      "assistant",
			Content:   "以下是一些分析建议...",
			Timestamp: time.Now().Unix() - 99,
		}

		// 将建议消息插入到开头
		allMessages := append([]ChatMessage{suggestionUser, suggestionAssistant}, messages...)

		totalContents, successfulResponses := simulateComprehensiveReportExtraction(allMessages)

		// 属性：建议消息不应被计入，成功步骤数量不变
		if successfulResponses != numSuccess {
			t.Logf("seed=%d: 跳过建议消息后成功步骤数量不匹配: expected=%d, got=%d",
				seed, numSuccess, successfulResponses)
			return false
		}

		// 属性：总内容数不包含建议消息的贡献
		expectedTotal := (numSuccess + numFailed) * 2
		if totalContents != expectedTotal {
			t.Logf("seed=%d: 跳过建议消息后总内容数量不匹配: expected=%d, got=%d",
				seed, expectedTotal, totalContents)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4b (skip suggestion messages) failed: %v", err)
	}
}

// TestProperty4c_ReplaySessionFailedStepsExcluded 验证对于仅包含失败步骤的 Replay_Session，
// 成功步骤的 assistant 响应数量为 0，但所有用户请求和 assistant 响应仍被提取。
// **Validates: Requirements 5.2**
func TestProperty4c_ReplaySessionFailedStepsExcluded(t *testing.T) {
	config := &quick.Config{
		MaxCount: 100,
		Rand:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	f := func(seed int64) bool {
		r := rand.New(rand.NewSource(seed))

		numFailed := r.Intn(5) + 1 // 1-5 个失败步骤，0 个成功步骤

		messages := generateReplaySessionMessages(r, 0, numFailed)

		_, successfulResponses := simulateComprehensiveReportExtraction(messages)

		// 属性：没有成功步骤时，成功响应数量应为 0
		if successfulResponses != 0 {
			t.Logf("seed=%d: 全部失败步骤时成功响应数量应为 0, got=%d",
				seed, successfulResponses)
			return false
		}

		return true
	}

	if err := quick.Check(f, config); err != nil {
		t.Errorf("Property 4c (failed steps excluded from successful count) failed: %v", err)
	}
}
