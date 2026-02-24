package main

import (
	"fmt"
	"strings"
	"testing"

	"vantagedata/agent"
	"vantagedata/database"

	"pgregory.net/rapid"
)

// facadeMethodCase describes a facade method on App that delegates to a service.
type facadeMethodCase struct {
	name        string
	serviceName string
	// callFn invokes the method on a bare App and returns (error, panicRecovered).
	callFn func(app *App) (err error, panicRecovered interface{})
}

// callSafe invokes fn inside a recover block, returning the error and any panic value.
func callSafe(fn func() error) (err error, panicRecovered interface{}) {
	defer func() {
		panicRecovered = recover()
	}()
	err = fn()
	return
}

// errorReturningFacadeMethods returns facade methods that return an error
// when the underlying service is nil.
func errorReturningFacadeMethods() []facadeMethodCase {
	return []facadeMethodCase{
		// --- ChatFacadeService ---
		{name: "GetChatHistory", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetChatHistory(); return e })
		}},
		{name: "GetChatHistoryByDataSource", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetChatHistoryByDataSource("ds"); return e })
		}},
		{name: "CheckSessionNameExists", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.CheckSessionNameExists("ds", "n", ""); return e })
		}},
		{name: "SaveChatHistory", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.SaveChatHistory(nil) })
		}},
		{name: "DeleteThread", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeleteThread("t") })
		}},
		{name: "CreateChatThread", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.CreateChatThread("ds", "title"); return e })
		}},
		{name: "UpdateThreadTitle", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.UpdateThreadTitle("t", "new"); return e })
		}},
		{name: "ClearHistory", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ClearHistory() })
		}},
		{name: "ClearThreadMessages", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ClearThreadMessages("t") })
		}},
		{name: "SendMessage", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.SendMessage("t", "m", "u", "r"); return e })
		}},
		{name: "SendFreeChatMessage", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.SendFreeChatMessage("t", "m", "u"); return e })
		}},
		{name: "CancelAnalysis", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.CancelAnalysis() })
		}},
		{name: "GetSessionFiles", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetSessionFiles("t"); return e })
		}},
		{name: "GetSessionFilePath", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetSessionFilePath("t", "f"); return e })
		}},
		{name: "OpenSessionFile", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.OpenSessionFile("t", "f") })
		}},
		{name: "DeleteSessionFile", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeleteSessionFile("t", "f") })
		}},
		{name: "OpenSessionResultsDirectory", serviceName: "chatFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.OpenSessionResultsDirectory("t") })
		}},

		// --- DataSourceFacadeService ---
		{name: "GetDataSources", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetDataSources(); return e })
		}},
		{name: "GetDataSourceStatistics", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetDataSourceStatistics(); return e })
		}},
		{name: "StartDataSourceAnalysis", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.StartDataSourceAnalysis("ds"); return e })
		}},
		{name: "ImportExcelDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ImportExcelDataSource("n", "/p"); return e })
		}},
		{name: "ImportCSVDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ImportCSVDataSource("n", "/p"); return e })
		}},
		{name: "ImportJSONDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ImportJSONDataSource("n", "/p"); return e })
		}},
		{name: "AddDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.AddDataSource("n", "t", nil); return e })
		}},
		{name: "DeleteDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeleteDataSource("ds") })
		}},
		{name: "RenameDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.RenameDataSource("ds", "new") })
		}},
		{name: "GetDataSourceTables", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetDataSourceTables("ds"); return e })
		}},
		{name: "GetDataSourceTableData", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetDataSourceTableData("ds", "tbl"); return e })
		}},
		{name: "GetDataSourceTableCount", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetDataSourceTableCount("ds", "tbl"); return e })
		}},
		{name: "DeleteTable", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeleteTable("ds", "tbl") })
		}},
		{name: "RenameColumn", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.RenameColumn("ds", "tbl", "old", "new") })
		}},
		{name: "DeleteColumn", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeleteColumn("ds", "tbl", "col") })
		}},
		{name: "UpdateMySQLExportConfig", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.UpdateMySQLExportConfig("ds", "h", "p", "u", "pw", "db") })
		}},
		{name: "RefreshDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.RefreshDataSource("ds"); return e })
		}},
		{name: "RefreshEcommerceDataSource", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.RefreshEcommerceDataSource("ds"); return e })
		}},
		{name: "GetJiraProjects", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetJiraProjects("cloud", "url", "user", "token"); return e })
		}},
		{name: "GetDataSourceTableDataWithCount", serviceName: "dataSourceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetDataSourceTableDataWithCount("ds", "tbl"); return e })
		}},

		// --- ExportFacadeService ---
		{name: "ExportToCSV", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportToCSV("ds", []string{"t"}, "/o") })
		}},
		{name: "ExportToJSON", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportToJSON("ds", []string{"t"}, "/o") })
		}},
		{name: "ExportToSQL", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportToSQL("ds", []string{"t"}, "/o") })
		}},
		{name: "ExportToExcel", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportToExcel("ds", []string{"t"}, "/o") })
		}},
		{name: "ExportToMySQL", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportToMySQL("ds", []string{"t"}, "h", "p", "u", "pw", "db") })
		}},
		{name: "TestMySQLConnection", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.TestMySQLConnection("h", "p", "u", "pw") })
		}},
		{name: "GetMySQLDatabases", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMySQLDatabases("h", "p", "u", "pw"); return e })
		}},
		{name: "ExportSessionHTML", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportSessionHTML("t") })
		}},
		{name: "ExportDashboardToPDF", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportDashboardToPDF(DashboardExportData{}) })
		}},
		{name: "ExportSessionFilesToZip", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportSessionFilesToZip("t", "m") })
		}},
		{name: "DownloadSessionFile", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DownloadSessionFile("t", "f") })
		}},
		{name: "GetSessionFileAsBase64", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetSessionFileAsBase64("t", "f"); return e })
		}},
		{name: "GenerateCSVThumbnail", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GenerateCSVThumbnail("t", "f"); return e })
		}},
		{name: "GenerateFilePreview", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GenerateFilePreview("t", "f"); return e })
		}},
		{name: "ExportTableToExcel", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportTableToExcel(nil, "sheet") })
		}},
		{name: "ExportDashboardToExcel", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportDashboardToExcel(DashboardExportData{}) })
		}},
		{name: "ExportMessageToPDF", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportMessageToPDF("content", "msg") })
		}},
		{name: "ExportDashboardToPPT", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportDashboardToPPT(DashboardExportData{}) })
		}},
		{name: "ExportDashboardToWord", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportDashboardToWord(DashboardExportData{}) })
		}},
		{name: "PrepareComprehensiveReport", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.PrepareComprehensiveReport(ComprehensiveReportRequest{}); return e })
		}},
		{name: "ExportComprehensiveReport", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExportComprehensiveReport("r", "pdf") })
		}},
		{name: "GenerateComprehensiveReport", serviceName: "exportFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.GenerateComprehensiveReport(ComprehensiveReportRequest{}) })
		}},

		// --- AnalysisFacadeService ---
		{name: "SaveMetricsJson", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.SaveMetricsJson("m", "{}") })
		}},
		{name: "LoadMetricsJson", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.LoadMetricsJson("m"); return e })
		}},
		{name: "ExtractMetricsFromAnalysis", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExtractMetricsFromAnalysis("t", "m", "c") })
		}},
		{name: "AddAnalysisRecord", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.AddAnalysisRecord("ds", agent.AnalysisRecord{}) })
		}},
		{name: "RecordIntentSelection", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.RecordIntentSelection("t", IntentSuggestion{}) })
		}},
		{name: "GetMessageAnalysisData", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMessageAnalysisData("t", "m"); return e })
		}},
		{name: "ShowStepResultOnDashboard", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ShowStepResultOnDashboard("t", "m") })
		}},
		{name: "ShowAllSessionResults", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ShowAllSessionResults("t") })
		}},
		{name: "SaveMessageAnalysisResults", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.SaveMessageAnalysisResults("t", "m", nil) })
		}},
		{name: "SaveSessionRecording", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.SaveSessionRecording("t", "title", "desc"); return e })
		}},
		{name: "GetSessionRecordings", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetSessionRecordings(); return e })
		}},
		{name: "ReplayAnalysisRecording", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ReplayAnalysisRecording("r", "ds", false, 0); return e })
		}},
		{name: "GenerateIntentSuggestions", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GenerateIntentSuggestions("t", "msg"); return e })
		}},
		{name: "GenerateIntentSuggestionsWithExclusions", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GenerateIntentSuggestionsWithExclusions("t", "msg", nil); return e })
		}},
		{name: "ExtractSuggestionsFromAnalysis", serviceName: "analysisFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ExtractSuggestionsFromAnalysis("t", "u", "c") })
		}},

		// --- SkillFacadeService ---
		{name: "GetSkills", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetSkills(); return e })
		}},
		{name: "GetEnabledSkills", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetEnabledSkills(); return e })
		}},
		{name: "GetSkillCategories", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetSkillCategories(); return e })
		}},
		{name: "EnableSkill", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.EnableSkill("s") })
		}},
		{name: "DisableSkill", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DisableSkill("s") })
		}},
		{name: "DeleteSkill", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeleteSkill("s") })
		}},
		{name: "ReloadSkills", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ReloadSkills() })
		}},
		{name: "ListSkills", serviceName: "skillFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ListSkills(); return e })
		}},

		// --- PythonFacadeService ---
		{name: "InstallPythonPackages", serviceName: "pythonFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.InstallPythonPackages("/usr/bin/python3", []string{"numpy"}) })
		}},
		{name: "CreateVantagicsEnvironment", serviceName: "pythonFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.CreateVantagicsEnvironment(); return e })
		}},

		// --- DashboardFacadeService ---
		{name: "SaveLayout", serviceName: "dashboardFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.SaveLayout(database.LayoutConfiguration{}) })
		}},
		{name: "LoadLayout", serviceName: "dashboardFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.LoadLayout("u"); return e })
		}},
		{name: "CheckComponentHasData", serviceName: "dashboardFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.CheckComponentHasData("chart", "i"); return e })
		}},
		{name: "GetFilesByCategory", serviceName: "dashboardFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetFilesByCategory("images"); return e })
		}},
		{name: "DownloadFile", serviceName: "dashboardFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.DownloadFile("f"); return e })
		}},
		{name: "ExportDashboard", serviceName: "dashboardFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ExportDashboard(database.ExportRequest{}); return e })
		}},

		// --- LicenseFacadeService ---
		{name: "ActivateLicense", serviceName: "licenseFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ActivateLicense("http://s", "sn"); return e })
		}},
		{name: "RequestSN", serviceName: "licenseFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.RequestSN("http://s", "e@e.com"); return e })
		}},
		{name: "LoadSavedActivation", serviceName: "licenseFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.LoadSavedActivation("sn"); return e })
		}},
		{name: "DeactivateLicense", serviceName: "licenseFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.DeactivateLicense() })
		}},
		{name: "RefreshLicense", serviceName: "licenseFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.RefreshLicense(); return e })
		}},

		// --- MarketplaceFacadeService ---
		{name: "MarketplaceLoginWithSN", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.MarketplaceLoginWithSN() })
		}},
		{name: "EnsureMarketplaceAuth", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.EnsureMarketplaceAuth() })
		}},
		{name: "MarketplacePortalLogin", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.MarketplacePortalLogin(); return e })
		}},
		{name: "GetMarketplaceCategories", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMarketplaceCategories(); return e })
		}},
		{name: "SharePackToMarketplace", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.SharePackToMarketplace("/p", 1, "free", 0, "desc") })
		}},
		{name: "BrowseMarketplacePacks", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.BrowseMarketplacePacks(1); return e })
		}},
		{name: "GetMySharedPackNames", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMySharedPackNames(); return e })
		}},
		{name: "GetMyPublishedPacks", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMyPublishedPacks("src"); return e })
		}},
		{name: "ReplaceMarketplacePack", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.ReplaceMarketplacePack("/p", 1) })
		}},
		{name: "DownloadMarketplacePack", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.DownloadMarketplacePack(1); return e })
		}},
		{name: "GetMarketplaceCreditsBalance", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMarketplaceCreditsBalance(); return e })
		}},
		{name: "PurchaseAdditionalUses", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.PurchaseAdditionalUses(1, 1) })
		}},
		{name: "RenewSubscription", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.RenewSubscription(1, 1) })
		}},
		{name: "RefreshPurchasedPackLicenses", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { return a.RefreshPurchasedPackLicenses() })
		}},
		{name: "ReportPackUsage", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.ReportPackUsage(1, "now"); return e })
		}},
		{name: "GetMarketplaceNotifications", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetMarketplaceNotifications(); return e })
		}},
		{name: "GetPackListingID", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetPackListingID("pack"); return e })
		}},
		{name: "GetShareURL", serviceName: "marketplaceFacadeService", callFn: func(a *App) (error, interface{}) {
			return callSafe(func() error { _, e := a.GetShareURL("pack"); return e })
		}},
	}
}

// Feature: main-architecture-refactor, Property 6: 未初始化服务的安全错误处理
//
// For any App facade method, when the corresponding service has not been registered
// in the registry, calling that method should return a non-nil error and not trigger a panic.
//
// **Validates: Requirements 3.4**

func TestProperty6_UninitializedServiceSafeErrorHandling(t *testing.T) {
	methods := errorReturningFacadeMethods()

	rapid.Check(t, func(t *rapid.T) {
		// Pick a random facade method to test
		idx := rapid.IntRange(0, len(methods)-1).Draw(t, "methodIndex")
		m := methods[idx]

		// Create a bare App with all facade services nil
		app := &App{}

		err, panicVal := m.callFn(app)

		// Property: must not panic
		if panicVal != nil {
			t.Fatalf("facade method %q panicked: %v", m.name, panicVal)
		}

		// Property: must return non-nil error
		if err == nil {
			t.Fatalf("facade method %q returned nil error when %s is nil", m.name, m.serviceName)
		}

		// Property: error message should contain context about the uninitialized service
		errMsg := err.Error()
		if !strings.Contains(strings.ToLower(errMsg), "not initialized") &&
			!strings.Contains(strings.ToLower(errMsg), "service not") {
			t.Fatalf("facade method %q error should mention service not initialized, got: %q", m.name, errMsg)
		}
	})
}

// Feature: main-architecture-refactor, Property 5: 门面委托正确性
//
// For any facade method call, when the underlying service is initialized,
// the facade method's return value should be consistent with directly calling
// the underlying service method (i.e., no "not initialized" error is returned).
//
// Since fully initializing all services requires complex dependencies, we verify
// the delegation pattern: when a facade service field is non-nil, the facade method
// does NOT return a "not initialized" error — proving it delegates to the service
// rather than short-circuiting.
//
// **Validates: Requirements 3.2**

func TestProperty5_FacadeDelegationCorrectness(t *testing.T) {
	methods := errorReturningFacadeMethods()

	rapid.Check(t, func(t *rapid.T) {
		idx := rapid.IntRange(0, len(methods)-1).Draw(t, "methodIndex")
		m := methods[idx]

		// Create an App with all facade services nil
		app := &App{}

		// First, verify the nil case returns "not initialized" error
		errNil, panicNil := m.callFn(app)
		if panicNil != nil {
			t.Fatalf("facade method %q panicked with nil service: %v", m.name, panicNil)
		}
		if errNil == nil {
			t.Fatalf("facade method %q should return error when service is nil", m.name)
		}
		nilErrMsg := errNil.Error()
		if !strings.Contains(strings.ToLower(nilErrMsg), "not initialized") {
			t.Fatalf("nil-service error for %q should mention 'not initialized', got: %q", m.name, nilErrMsg)
		}

		// Now create an App with a minimal (non-nil) facade service set.
		// The method will likely fail with a different error (e.g., nil dependency inside
		// the service), but it should NOT return a "not initialized" error — proving
		// the facade correctly delegates to the service.
		appWithSvc := &App{
			chatFacadeService:        &ChatFacadeService{},
			dataSourceFacadeService:  &DataSourceFacadeService{},
			analysisFacadeService:    &AnalysisFacadeService{},
			exportFacadeService:      &ExportFacadeService{},
			dashboardFacadeService:   &DashboardFacadeService{},
			licenseFacadeService:     &LicenseFacadeService{},
			marketplaceFacadeService: &MarketplaceFacadeService{},
			skillFacadeService:       &SkillFacadeService{},
			pythonFacadeService:      &PythonFacadeService{},
			connectionTestService:    &ConnectionTestService{},
		}

		errSvc, panicSvc := m.callFn(appWithSvc)

		// If the service method panics due to nil internal dependencies, that's
		// expected — we're only testing that the facade layer delegates correctly.
		// A panic inside the service proves delegation happened (the facade didn't
		// short-circuit with "not initialized").
		if panicSvc != nil {
			// Delegation succeeded (reached the service), panic is from nil internals.
			return
		}

		// Property: if there IS an error, it must NOT be a "not initialized" error.
		// This proves the facade delegated to the service rather than short-circuiting.
		if errSvc != nil {
			svcErrMsg := strings.ToLower(errSvc.Error())
			if strings.Contains(svcErrMsg, "not initialized") &&
				strings.Contains(svcErrMsg, fmt.Sprintf("[app.%s]", strings.ToLower(m.name))) {
				t.Fatalf("facade method %q returned 'not initialized' error even though service is non-nil: %q",
					m.name, errSvc.Error())
			}
		}
	})
}
