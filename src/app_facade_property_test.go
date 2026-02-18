package main

import (
	"fmt"
	"strings"
	"testing"

	"pgregory.net/rapid"
)

// facadeMethodCase describes a facade method on App that delegates to a service.
// callFn invokes the method and returns (error, panicRecovered).
type facadeMethodCase struct {
	name        string
	serviceName string
	callFn      func(app *App) (err error, panicRecovered interface{})
}

// callSafe invokes fn inside a recover block, returning the error and any panic value.
func callSafe(fn func() error) (err error, panicRecovered interface{}) {
	defer func() {
		panicRecovered = recover()
	}()
	err = fn()
	return
}

// errorReturningFacadeMethods returns facade methods that return an error when the service is nil.
// These are the methods where we can directly verify error != nil.
func errorReturningFacadeMethods() []facadeMethodCase {
	return []facadeMethodCase{
		// ChatFacadeService methods
		{name: "GetChatHistory", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetChatHistory(); return err })
		}},
		{name: "GetChatHistoryByDataSource", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetChatHistoryByDataSource("ds-1"); return err })
		}},
		{name: "CheckSessionNameExists", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.CheckSessionNameExists("ds-1", "name", ""); return err })
		}},
		{name: "SaveChatHistory", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.SaveChatHistory(nil) })
		}},
		{name: "DeleteThread", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DeleteThread("t-1") })
		}},
		{name: "CreateChatThread", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.CreateChatThread("ds-1", "title"); return err })
		}},
		{name: "UpdateThreadTitle", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.UpdateThreadTitle("t-1", "new"); return err })
		}},
		{name: "ClearHistory", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ClearHistory() })
		}},
		{name: "ClearThreadMessages", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ClearThreadMessages("t-1") })
		}},
		{name: "SendMessage", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.SendMessage("t-1", "msg", "uid", "rid"); return err })
		}},
		{name: "SendFreeChatMessage", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.SendFreeChatMessage("t-1", "msg", "uid"); return err })
		}},
		{name: "CancelAnalysis", serviceName: "chatFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.CancelAnalysis() })
		}},

		// DataSourceFacadeService methods
		{name: "GetDataSources", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetDataSources(); return err })
		}},
		{name: "GetDataSourceStatistics", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetDataSourceStatistics(); return err })
		}},
		{name: "StartDataSourceAnalysis", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.StartDataSourceAnalysis("ds-1"); return err })
		}},
		{name: "ImportExcelDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ImportExcelDataSource("name", "/path"); return err })
		}},
		{name: "ImportCSVDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ImportCSVDataSource("name", "/path"); return err })
		}},
		{name: "ImportJSONDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ImportJSONDataSource("name", "/path"); return err })
		}},
		{name: "AddDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.AddDataSource("name", "type", nil); return err })
		}},
		{name: "DeleteDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DeleteDataSource("ds-1") })
		}},
		{name: "RenameDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.RenameDataSource("ds-1", "new") })
		}},
		{name: "GetDataSourceTables", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetDataSourceTables("ds-1"); return err })
		}},
		{name: "GetDataSourceTableData", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetDataSourceTableData("ds-1", "tbl"); return err })
		}},
		{name: "GetDataSourceTableCount", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetDataSourceTableCount("ds-1", "tbl"); return err })
		}},
		{name: "DeleteTable", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DeleteTable("ds-1", "tbl") })
		}},
		{name: "RenameColumn", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.RenameColumn("ds-1", "tbl", "old", "new") })
		}},
		{name: "DeleteColumn", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DeleteColumn("ds-1", "tbl", "col") })
		}},
		{name: "UpdateMySQLExportConfig", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.UpdateMySQLExportConfig("ds-1", "h", "p", "u", "pw", "db") })
		}},
		{name: "RefreshDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.RefreshDataSource("ds-1"); return err })
		}},
		{name: "RefreshEcommerceDataSource", serviceName: "dataSourceFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.RefreshEcommerceDataSource("ds-1"); return err })
		}},

		// ExportFacadeService methods
		{name: "ExportToCSV", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ExportToCSV("ds-1", []string{"tbl"}, "/out") })
		}},
		{name: "ExportToJSON", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ExportToJSON("ds-1", []string{"tbl"}, "/out") })
		}},
		{name: "ExportToSQL", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ExportToSQL("ds-1", []string{"tbl"}, "/out") })
		}},
		{name: "ExportToExcel", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ExportToExcel("ds-1", []string{"tbl"}, "/out") })
		}},
		{name: "ExportToMySQL", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ExportToMySQL("ds-1", []string{"tbl"}, "h", "p", "u", "pw", "db") })
		}},
		{name: "TestMySQLConnection", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.TestMySQLConnection("h", "p", "u", "pw") })
		}},
		{name: "GetMySQLDatabases", serviceName: "exportFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetMySQLDatabases("h", "p", "u", "pw"); return err })
		}},

		// AnalysisFacadeService methods
		{name: "SaveMetricsJson", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.SaveMetricsJson("msg-1", "{}") })
		}},
		{name: "LoadMetricsJson", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.LoadMetricsJson("msg-1"); return err })
		}},
		{name: "ExtractMetricsFromAnalysis", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ExtractMetricsFromAnalysis("t-1", "msg-1", "content") })
		}},
		{name: "AddAnalysisRecord", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.AddAnalysisRecord("ds-1", agent.AnalysisRecord{}) })
		}},
		{name: "RecordIntentSelection", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.RecordIntentSelection("t-1", IntentSuggestion{}) })
		}},
		{name: "GetMessageAnalysisData", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetMessageAnalysisData("t-1", "msg-1"); return err })
		}},
		{name: "ShowStepResultOnDashboard", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ShowStepResultOnDashboard("t-1", "msg-1") })
		}},
		{name: "ShowAllSessionResults", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ShowAllSessionResults("t-1") })
		}},
		{name: "SaveMessageAnalysisResults", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.SaveMessageAnalysisResults("t-1", "msg-1", nil) })
		}},
		{name: "SaveSessionRecording", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.SaveSessionRecording("t-1", "title", "desc"); return err })
		}},
		{name: "GetSessionRecordings", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetSessionRecordings(); return err })
		}},
		{name: "ReplayAnalysisRecording", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ReplayAnalysisRecording("rec-1", "ds-1", false, 0); return err })
		}},
		{name: "GenerateIntentSuggestions", serviceName: "analysisFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GenerateIntentSuggestions("t-1", "msg"); return err })
		}},

		// SkillFacadeService methods
		{name: "GetSkills", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetSkills(); return err })
		}},
		{name: "GetEnabledSkills", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetEnabledSkills(); return err })
		}},
		{name: "GetSkillCategories", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetSkillCategories(); return err })
		}},
		{name: "EnableSkill", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.EnableSkill("skill-1") })
		}},
		{name: "DisableSkill", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DisableSkill("skill-1") })
		}},
		{name: "DeleteSkill", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DeleteSkill("skill-1") })
		}},
		{name: "ReloadSkills", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.ReloadSkills() })
		}},
		{name: "ListSkills", serviceName: "skillFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ListSkills(); return err })
		}},

		// PythonFacadeService methods
		{name: "InstallPythonPackages", serviceName: "pythonFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.InstallPythonPackages("/usr/bin/python3", []string{"numpy"}) })
		}},
		{name: "CreateVantageDataEnvironment", serviceName: "pythonFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.CreateVantageDataEnvironment(); return err })
		}},

		// DashboardFacadeService methods
		{name: "SaveLayout", serviceName: "dashboardFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.SaveLayout(database.LayoutConfiguration{}) })
		}},
		{name: "LoadLayout", serviceName: "dashboardFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.LoadLayout("user-1"); return err })
		}},
		{name: "CheckComponentHasData", serviceName: "dashboardFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.CheckComponentHasData("chart", "inst-1"); return err })
		}},
		{name: "GetFilesByCategory", serviceName: "dashboardFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.GetFilesByCategory("images"); return err })
		}},
		{name: "DownloadFile", serviceName: "dashboardFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.DownloadFile("file-1"); return err })
		}},
		{name: "ExportDashboard", serviceName: "dashboardFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ExportDashboard(database.ExportRequest{}); return err })
		}},

		// LicenseFacadeService methods
		{name: "ActivateLicense", serviceName: "licenseFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.ActivateLicense("http://server", "sn-1"); return err })
		}},
		{name: "RequestSN", serviceName: "licenseFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.RequestSN("http://server", "test@example.com"); return err })
		}},
		{name: "LoadSavedActivation", serviceName: "licenseFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.LoadSavedActivation("sn-1"); return err })
		}},
		{name: "DeactivateLicense", serviceName: "licenseFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { return app.DeactivateLicense() })
		}},
		{name: "RefreshLicense", serviceName: "licenseFacadeService", callFn: func(app *App) (error, interface{}) {
			return callSafe(func() error { _, err := app.RefreshLicense(); return err })
		}},
	}
}
