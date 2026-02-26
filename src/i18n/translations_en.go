package i18n

var englishTranslations = map[string]string{
	// License Server
	"license.invalid_request":        "Invalid request format",
	"license.invalid_sn":             "Invalid serial number",
	"license.sn_disabled":            "Serial number has been disabled",
	"license.sn_expired":             "Serial number has expired",
	"license.encrypt_failed":         "Encryption failed",
	"license.invalid_email":          "Please enter a valid email address",
	"license.email_blacklisted":      "Your email has been restricted",
	"license.email_not_whitelisted":  "Your email is not in the whitelist",
	"license.email_already_used":     "You have already requested a serial number",
	"license.no_available_sn":        "No available serial numbers, please contact administrator",
	"license.rate_limit_exceeded":    "Daily request limit reached, please try again tomorrow",
	"license.email_limit_exceeded":   "Daily email limit reached for this IP, please try again tomorrow",
	"license.internal_error":         "Server internal error",
	"license.smtp_incomplete":        "SMTP configuration incomplete",
	"license.email_send_failed":      "Failed to send email: %s",
	"license.email_sent":             "Serial number has been sent to your email",
	"license.sn_deleted":             "Successfully deleted %d unused serial numbers",
	"license.group_has_sn":           "This group still has %d serial numbers, cannot delete",
	"license.group_not_found":        "Group not found",
	"license.sn_not_found":           "Serial number not found",
	"license.sn_updated":             "Serial number updated successfully",
	"license.sn_created":             "Serial number created successfully",
	"license.group_created":          "Group created successfully",
	"license.group_updated":          "Group updated successfully",
	"license.group_deleted":          "Group deleted successfully",
	"license.invalid_group_id":       "Invalid group ID",
	"license.invalid_sn_id":          "Invalid serial number ID",
	"license.database_error":         "Database operation failed",
	"license.activation_success":     "Activation successful",
	"license.activation_failed":      "Activation failed: %s",
	"license.deactivation_success":   "Deactivation successful",
	"license.refresh_success":        "License refreshed successfully",
	"license.refresh_failed":         "License refresh failed: %s",

	// Data Source Operations
	"datasource.import_success":       "Data source imported successfully",
	"datasource.import_failed":        "Data source import failed: %s",
	"datasource.delete_success":       "Data source deleted successfully",
	"datasource.delete_failed":        "Data source deletion failed: %s",
	"datasource.export_success":       "Data source exported successfully",
	"datasource.export_failed":        "Data source export failed: %s",
	"datasource.not_found":            "Data source not found",
	"datasource.already_exists":       "Data source already exists",
	"datasource.invalid_name":         "Invalid data source name",
	"datasource.connection_failed":    "Connection failed: %s",
	"datasource.test_success":         "Connection test successful",
	"datasource.semantic_opt_success": "Semantic optimization completed",
	"datasource.semantic_opt_failed":  "Semantic optimization failed: %s",
	"datasource.semantic_opt_summary": "Semantically optimized data source with %d tables",

	// Analysis Operations
	"analysis.execution_failed":     "Analysis execution failed",
	"analysis.sql_error":            "SQL execution error: %s",
	"analysis.python_error":         "Python script execution error: %s",
	"analysis.timeout":              "Analysis timeout",
	"analysis.cancelled":            "Analysis cancelled by user",
	"analysis.no_results":           "No results found",
	"analysis.export_success":       "Analysis results exported successfully",
	"analysis.export_failed":        "Analysis results export failed: %s",
	"analysis.invalid_request":      "Invalid analysis request",
	"analysis.queue_full":           "Analysis queue is full, please try again later",
	"analysis.in_progress":          "Analysis is already in progress",
	"analysis.report_gen_success":   "Report generated successfully",
	"analysis.report_gen_failed":    "Report generation failed: %s",

	// File Operations
	"file.not_found":         "File not found: %s",
	"file.read_error":        "Failed to read file: %s",
	"file.write_error":       "Failed to write file: %s",
	"file.delete_error":      "Failed to delete file: %s",
	"file.invalid_format":    "Invalid file format",
	"file.too_large":         "File is too large",
	"file.upload_success":    "File uploaded successfully",
	"file.upload_failed":     "File upload failed: %s",
	"file.download_success":  "File downloaded successfully",
	"file.download_failed":   "File download failed: %s",

	// Database Operations
	"db.connection_failed":   "Database connection failed: %s",
	"db.query_error":         "Database query error: %s",
	"db.insert_error":        "Database insert error: %s",
	"db.update_error":        "Database update error: %s",
	"db.delete_error":        "Database delete error: %s",
	"db.transaction_failed":  "Database transaction failed: %s",
	"db.migration_failed":    "Database migration failed: %s",
	"db.backup_success":      "Database backup successful",
	"db.backup_failed":       "Database backup failed: %s",
	"db.restore_success":     "Database restore successful",
	"db.restore_failed":      "Database restore failed: %s",

	// Skills Management
	"skills.install_success":   "Skills installed successfully: %s",
	"skills.install_failed":    "Skills installation failed: %s",
	"skills.enable_success":    "Skill enabled successfully",
	"skills.enable_failed":     "Skill enable failed: %s",
	"skills.disable_success":   "Skill disabled successfully",
	"skills.disable_failed":    "Skill disable failed: %s",
	"skills.delete_success":    "Skill deleted successfully",
	"skills.delete_failed":     "Skill deletion failed: %s",
	"skills.not_found":         "Skill not found",
	"skills.already_exists":    "Skill already exists",
	"skills.invalid_package":   "Invalid skill package",
	"skills.load_failed":       "Failed to load skills: %s",

	// Python Environment
	"python.env_create_success":    "Python environment created successfully",
	"python.env_create_failed":     "Python environment creation failed: %s",
	"python.package_install_success": "Packages installed successfully",
	"python.package_install_failed":  "Package installation failed: %s",
	"python.not_found":             "Python not found",
	"python.invalid_version":       "Invalid Python version",
	"python.script_error":          "Python script execution error: %s",

	// Configuration
	"config.load_failed":    "Failed to load configuration: %s",
	"config.save_success":   "Configuration saved successfully",
	"config.save_failed":    "Failed to save configuration: %s",
	"config.invalid_value":  "Invalid configuration value: %s",
	"config.reset_success":  "Configuration reset to defaults",

	// Authentication & Authorization
	"auth.unauthorized":       "Unauthorized access",
	"auth.forbidden":          "Access forbidden",
	"auth.token_expired":      "Authentication token expired",
	"auth.token_invalid":      "Invalid authentication token",
	"auth.login_required":     "Login required",
	"auth.permission_denied":  "Permission denied",

	// General Messages
	"general.success":           "Operation successful",
	"general.failed":            "Operation failed",
	"general.invalid_input":     "Invalid input",
	"general.required_field":    "Required field: %s",
	"general.not_found":         "Resource not found",
	"general.already_exists":    "Resource already exists",
	"general.internal_error":    "Internal server error",
	"general.network_error":     "Network error",
	"general.timeout":           "Operation timeout",
	"general.cancelled":         "Operation cancelled",
	"general.processing":        "Processing...",
	"general.please_wait":       "Please wait...",

	// Export Operations
	"export.pdf_success":    "PDF exported successfully",
	"export.pdf_failed":     "PDF export failed: %s",
	"export.excel_success":  "Excel exported successfully",
	"export.excel_failed":   "Excel export failed: %s",
	"export.word_success":   "Word document exported successfully",
	"export.word_failed":    "Word export failed: %s",
	"export.ppt_success":    "PowerPoint exported successfully",
	"export.ppt_failed":     "PowerPoint export failed: %s",
	"export.html_success":   "HTML exported successfully",
	"export.html_failed":    "HTML export failed: %s",
	"export.no_data":        "No data to export",
	"export.invalid_format": "Invalid export format",

	// Export Document Content
	"export.doc_description":      "Generated by Vantagics intelligent analysis system",
	"export.datasource_label":     "Data Source: ",
	"export.analysis_request":     "Analysis Request: ",
	"export.key_metrics":          "Key Metrics",
	"export.metric_column":        "Metric",
	"export.value_column":         "Value",
	"export.change_column":        "Change",
	"export.data_tables":          "Data Tables",
	"export.data_visualization":   "Data Visualization",
	"export.chart_number":         "Chart %d / %d",
	"export.table_note":           "Note: Showing first %d rows of %d total rows",
	"export.table_extracted":      "[Table data extracted]",
	"export.generated_by":         "Generated by Vantagics intelligent analysis system",

	// Report Export
	"report.font_load_failed":       "Failed to load Chinese font",
	"report.data_analysis_report":   "Data Analysis Report",
	"report.data_source_label":      "Data Source",
	"report.analysis_request_label": "Analysis Request",
	"report.generated_time_label":   "Generated Time",
	"report.page_number":            "Page %d",

	// Excel Export
	"excel.default_sheet_name":      "Data Table",
	"excel.multi_table_title":       "Multi-Table Data Analysis",
	"excel.report_subject":          "Data Analysis Report",
	"excel.report_keywords":         "Data Analysis, Report, Excel",
	"excel.report_category":         "Data Analysis",

	// PPT Export
	"ppt.key_metrics":               "Key Metrics",
	"ppt.data_visualization":        "Data Visualization %d",
	"ppt.smart_insights":            "Smart Insights",
	"ppt.smart_insights_continued":  "Smart Insights (Continued %d)",
	"ppt.data_tables":               "Data Tables",
	"ppt.data_tables_page":          "Data Tables (Page %d)",
	"ppt.table_info":                "Showing rows %d-%d of %d total",
	"ppt.columns_truncated":         "(Columns truncated)",
	"ppt.thank_you":                 "Thank You",
	"ppt.tagline":                   "Data-Driven Decisions, AI-Powered Future",
	"ppt.footer_year":               "Vantagics Intelligent Analysis System · %s",

	// MCP Services
	"mcp.service_not_found":    "MCP service not found",
	"mcp.connection_failed":    "MCP service connection failed: %s",
	"mcp.call_failed":          "MCP service call failed: %s",
	"mcp.invalid_response":     "Invalid MCP service response",
	"mcp.timeout":              "MCP service timeout",

	// Search API
	"search.api_not_configured": "Search API not configured",
	"search.api_call_failed":    "Search API call failed: %s",
	"search.no_results":         "No search results found",
	"search.invalid_query":      "Invalid search query",

	// Session Management
	"session.create_success":  "Session created successfully",
	"session.create_failed":   "Session creation failed: %s",
	"session.delete_success":  "Session deleted successfully",
	"session.delete_failed":   "Session deletion failed: %s",
	"session.not_found":       "Session not found",
	"session.already_exists":  "Session already exists",
	"session.expired":         "Session expired",

	// Table Operations
	"table.delete_success":     "Table deleted successfully",
	"table.delete_failed":      "Table deletion failed: %s",
	"table.not_found":          "Table not found",
	"table.column_delete_success": "Column deleted successfully",
	"table.column_delete_failed":  "Column deletion failed: %s",
	"table.column_rename_success": "Column renamed successfully",
	"table.column_rename_failed":  "Column rename failed: %s",

	// Dashboard Operations
	"dashboard.export_success":  "Dashboard exported successfully",
	"dashboard.export_failed":   "Dashboard export failed: %s",
	"dashboard.save_success":    "Dashboard layout saved successfully",
	"dashboard.save_failed":     "Dashboard layout save failed: %s",
	"dashboard.no_data":         "No dashboard data available",

	// Error Recovery Suggestions
	"error.recovery.check_query":           "Please check if your query is clear and specific",
	"error.recovery.simplify_query":        "Try simplifying the query conditions",
	"error.recovery.refresh_retry":         "If the problem persists, please refresh the page and try again",
	"error.recovery.reduce_data_range":     "Please try reducing the data range",
	"error.recovery.check_network":         "Check if the network connection is stable",
	"error.recovery.retry_later":           "Please try again later, the system may be processing other tasks",
	"error.recovery.resubmit":              "You can resubmit the analysis request",
	"error.recovery.check_data_format":     "Please check if the data format is correct",
	"error.recovery.try_different_method":  "Try using a different analysis method",
	"error.recovery.contact_support":       "If the problem persists, please contact technical support",
	"error.recovery.rephrase_query":        "Please try rephrasing your analysis request",
	"error.recovery.use_simpler_query":     "Use simpler query statements",
	"error.recovery.check_libraries":       "Required analysis libraries may not be installed",
	"error.recovery.check_admin":           "Please contact administrator to check system configuration",
	"error.recovery.reduce_batch":          "Try processing data in batches",
	"error.recovery.check_datasource":      "Please check if the data source is configured correctly",
	"error.recovery.check_table_field":     "Confirm if the table or field name is correct",
	"error.recovery.check_deleted":         "Check if the data has been deleted or moved",
	"error.recovery.check_data_type":       "Confirm if the data type is correct",
	"error.recovery.clean_reimport":        "Try cleaning or reimporting the data",
	"error.recovery.adjust_filters":        "Please try adjusting the filter conditions",
	"error.recovery.check_data_exists":     "Check if the data source contains the required data",
	"error.recovery.add_filters":           "Add more filter conditions",
	"error.recovery.consider_pagination":   "Consider pagination or batch queries",
	"error.recovery.check_service":         "Confirm if the service is running normally",
	"error.recovery.check_permissions":     "You may not have permission to access this resource",
	"error.recovery.contact_admin":         "Please contact administrator for appropriate permissions",
	"error.recovery.check_account":         "Check your account status",
	"error.recovery.resource_busy":         "Resource is being used by other tasks",
	"error.recovery.check_path":            "Please check if the resource path is correct",
	"error.recovery.confirm_resource":      "Contact administrator to confirm resource status",

	// Error Messages
	"error.analysis_error":              "An error occurred during analysis",
	"error.analysis_timeout":            "Analysis timeout, please try again later",
	"error.analysis_timeout_duration":   "Analysis timeout (ran for %v)",
	"error.analysis_cancelled":          "Analysis cancelled",
	"error.python_execution":            "Code execution failed",
	"error.python_syntax":               "Code syntax error",
	"error.python_import":               "Missing required analysis libraries",
	"error.python_memory":               "Insufficient memory, data may be too large",
	"error.data_not_found":              "Requested data not found",
	"error.data_invalid":                "Invalid data format",
	"error.data_empty":                  "Query result is empty",
	"error.data_too_large":              "Data exceeds size limit",
	"error.connection_failed":           "Connection failed, please check network",
	"error.connection_timeout":          "Connection timeout",
	"error.permission_denied":           "Insufficient permissions",
	"error.resource_busy":               "Resource busy, please try again later",
	"error.resource_not_found":          "Resource not found",
	"error.unknown":                     "Unknown error occurred",

	// Application Dialogs
	"app.about_title":                   "About Vantagics",
	"app.about_message":                 "Vantagics\n\nSee Beyond. Decide Better.\n\nVersion: 1.0.0\n© 2026 Vantagics. All rights reserved.",
	"app.devtools_title":                "Open Developer Tools",
	"app.devtools_message":              "Please use one of the following methods to open DevTools:\n\nMethod 1: Press F12\nMethod 2: Press Ctrl+Shift+I\nMethod 3: Press Ctrl+Shift+J\nMethod 4: Right-click in blank area and select \"Inspect\"\n\nIf none of the above works, run in development mode:\nwails dev",
	"app.confirm_exit_title":            "Confirm Exit",
	"app.confirm_exit_message":          "There is an analysis task in progress. Are you sure you want to exit?\n\nExiting will interrupt the analysis process.",
	"app.exit_button":                   "Exit",
	"app.cancel_button":                 "Cancel",
	"app.license_activation_failed":     "License verification failed: %v\nPlease check your network connection or contact administrator.",
	"app.license_refresh_failed":        "License refresh failed: %v\nYour license needs to be re-verified, please check your network connection or contact administrator.",

	// Report Generation
	"report.save_dialog_title":          "Save Analysis Report",
	"report.filename_prefix":            "Analysis_Report",
	"report.llm_not_initialized":        "LLM service not initialized, please configure API Key first",
	"report.generation_failed":          "Report generation failed: %v",
	"report.data_expired":               "Report data has expired, please regenerate",
	"report.word_generation_failed":     "Word document generation failed: %v",
	"report.pdf_generation_failed":      "PDF generation failed: %v",
	"report.no_content":                 "No content to export",
	"report.write_file_failed":          "Failed to write file: %v",

	// Report Sections
	"report.section.background":         "Background and Purpose",
	"report.section.data_overview":      "Data Overview",
	"report.section.key_metrics":        "Key Metrics Analysis",
	"report.section.deep_analysis":      "In-Depth Data Analysis",
	"report.section.findings":           "Key Findings and Insights",
	"report.section.conclusions":        "Conclusions and Recommendations",

	// Report Labels (duplicates removed - already defined above)
	"report.chart_label":                "Chart %d / %d",
	"report.total_rows":                 "Total Rows: %d",
	"report.showing_columns":            "Showing Columns: %s",
	"report.category_label":             "Category",
	"report.footer_text":                "Page %d",

	// Report Errors (duplicate removed - already defined above)

	// Comprehensive Report
	"comprehensive_report.filename_prefix":    "Comprehensive_Report",
	"comprehensive_report.save_dialog_title":  "Save Comprehensive Report",
	"comprehensive_report.no_valid_analysis":  "No valid analysis results to generate report",
	"comprehensive_report.data_source":        "Data Source: ",
	"comprehensive_report.session_name":       "Session Name: ",
	"comprehensive_report.all_analysis_results": "All Analysis Results: ",
	"comprehensive_report.analysis_request":   "Analysis Request",
	"comprehensive_report.analysis_result":    "Analysis Result",
	"comprehensive_report.insight":            "Insight",
	"comprehensive_report.key_metric":         "Key Metric",
	"comprehensive_report.table":              "Data Table",
	"comprehensive_report.pack_info_header":   "## Analysis Pack Information",
	"comprehensive_report.pack_author":        "Author: ",
	"comprehensive_report.pack_description":   "Description: ",
	"comprehensive_report.pack_source_name":   "Original Data Source: ",

	// Quick Analysis Pack - Preview & Export
	"qap.no_exportable_records":       "No exportable analysis records found",
	"qap.unknown_request":             "(Unknown request)",
	"qap.no_exportable_steps":         "Selected analysis requests have no exportable operations",
	"qap.no_exportable_operations":    "This session has no exportable analysis operations",
	"qap.load_pack_dialog_title":      "Load Quick Analysis Pack",

	// Quick Analysis Pack - Import
	"qap.invalid_file_format":         "Invalid file format, cannot parse quick analysis pack: %v",
	"qap.wrong_password":              "Incorrect password",
	"qap.invalid_pack_file":           "Invalid file format: not a valid quick analysis pack file",
	"qap.unsupported_version":         "Unsupported pack version: %s, please upgrade the software and try again",
	"qap.no_executable_steps":         "The analysis pack has no executable steps",
	"qap.schema_fetch_failed":         "Unable to get target data source schema: %v",
	"qap.missing_required_tables":     "Target data source is missing required tables: %s",
	"qap.python_not_configured":       "This analysis pack contains Python scripts, but Python environment is not configured. Please configure the Python path in Settings and try again.",
	"qap.permission_denied":           "Permission denied: %s",
	"qap.not_replay_session":          "This session is not a quick analysis session",

	// Quick Analysis Pack - Step Labels
	"qap.step_sql_query":              "SQL Query #%d",
	"qap.step_python_script":          "Python Script #%d",
	"qap.step_generic":                "Step #%d",
	"qap.step_execution_failed":       "Step %d execution failed: %v",
	"qap.step_python_not_configured":  "Step %d execution failed: Python environment not configured",

	// Quick Analysis Pack - Import Validation
	"qap.table_not_exists":            "Table '%s' does not exist",
	"qap.column_not_exists":           "Column '%s.%s' does not exist",
	"qap.step_sql_success":            "SQL executed successfully (Step %d):\n\n```json:table\n%s\n```",
	"qap.step_sql_failed":             "SQL execution failed: %v\n\nSQL:\n```sql\n%s\n```",
	"qap.step_execute_query":          "Execute step %d analysis query",

	// Quick Analysis Pack - Execution Messages
	"qap.step_skipped":                "⏭️ Step %d (%s) skipped: dependent step failed",
	"qap.execution_complete":          "�Quick analysis pack execution complete! Executed %d steps in total.",
	"qap.reexecution_complete":        "�Quick analysis pack re-execution complete! Executed %d steps in total.",
	"qap.step_sql_error":              "�Step %d (%s) execution failed: %v\n\n> 📋 Analysis request: %s\n\n```sql\n%s\n```",
	"qap.step_sql_success_full":       "�Step %d (%s):\n\n> 📋 Analysis request: %s\n\n```json:table\n%s\n```",
	"qap.step_sql_success_truncated":  "�Step %d (%s) (%d rows, showing first 20):\n\n> 📋 Analysis request: %s\n\n```json:table\n%s\n```",
	"qap.step_python_no_env":          "�Step %d (%s) execution failed: Python environment not configured\n\n> 📋 Analysis request: %s\n\n```python\n%s\n```",
	"qap.step_python_error":           "�Step %d (%s) execution failed: %v\n\n> 📋 Analysis request: %s\n\n```python\n%s\n```",
	"qap.step_python_success":         "�Step %d (%s):\n\n> 📋 Analysis request: %s\n\n```\n%s\n```",

	// Analysis Export
	"analysis_export.description":     "Vantagics analysis export file - contains executable SQL/Python steps",
	"analysis_export.dialog_title":    "Export Analysis",

	// Dashboard Export
	"dashboard.no_exportable_content": "No content available to export",
	"dashboard.message_not_found":     "Message not found: %v",
	"dashboard.step_no_results":       "This step has no displayable results",
	"dashboard.session_no_results":    "This session has no displayable results",
	"dashboard.mode_switch_blocked":   "Cannot switch mode while analysis is in progress",
	"dashboard.write_pdf_failed":      "Failed to write PDF file: %v",
	"dashboard.write_excel_failed":    "Failed to write Excel file: %v",
	"dashboard.write_ppt_failed":      "Failed to write PPT file: %v",
	"dashboard.write_word_failed":     "Failed to write Word file: %v",

	// Marketplace
	"marketplace.insufficient_credits": "Insufficient credits, need %d credits, current balance %.0f credits",

	// Data Source Import
	"datasource.unsupported_format":   "Unsupported file format: %s. Please use .xlsx or .xls format Excel files",
	"datasource.excel_format_error":   "Unable to open Excel file: file format not supported. Please ensure the file is a valid .xlsx format (Excel 2007 or later)",
	"datasource.excel_open_failed":    "Unable to open Excel file: %v",
	"datasource.no_sheets":            "No worksheets found in the Excel file",
	"datasource.no_valid_data":        "No valid data found in the Excel file",

	// Intent Generator
	"intent.generation_failed":        "Intent generation failed: %v",
	"intent.parse_failed":             "Response parsing failed: %v",
	"intent.no_suggestions":           "Unable to generate intent suggestions",

	// License Client
	"license_client.build_request_failed":  "Failed to build request: %v",
	"license_client.connect_failed":        "Failed to connect to server: %v",
	"license_client.read_response_failed":  "Failed to read response: %v",
	"license_client.parse_response_failed": "Failed to parse response: %v",
	"license_client.decrypt_failed":        "Decryption failed: %v",
	"license_client.parse_config_failed":   "Failed to parse configuration: %v",
	"license_client.credits_insufficient":  "Insufficient credits, remaining %.1f credits, each analysis requires %.1f credits",
	"license_client.daily_limit_reached":   "Daily analysis limit reached (%d times), please try again tomorrow",
	"license_client.first_use":             "First use, license verification required",
	"license_client.trial_label":           "Trial",
	"license_client.official_label":        "Official",
	"license_client.refresh_needed":        "%s license needs refresh (exceeded %d days)",

	// Usage License
	"usage.expired":                   "Usage permission has expired, please renew",
	"usage.uses_exhausted":            "Usage count exhausted, please repurchase",

	// PDF Font
	"pdf.font_load_failed":            "Unable to load Chinese font",

	// Tool Results
	"tool.no_valid_info":              "Sorry, unable to retrieve valid information.",

	// SQL Validation
	"sql.readonly_violation":          "Non-read-only SQL operation detected: %s (only SELECT queries are allowed)",

	// Dashboard Export Dialogs
	"dashboard.export_pdf_title":        "Export Dashboard to PDF",
	"dashboard.export_excel_title":      "Export Dashboard Data to Excel",
	"dashboard.export_ppt_title":        "Export Dashboard to PPT",
	"dashboard.export_word_title":       "Export Dashboard to Word",
	"dashboard.export_table_title":      "Export Table to Excel",
	"dashboard.export_message_pdf_title": "Export Analysis Results to PDF",
	"dashboard.filter_pdf":              "PDF Files",
	"dashboard.filter_excel":            "Excel Files",
	"dashboard.filter_ppt":              "PowerPoint Files",
	"dashboard.filter_word":             "Word Files",
	"dashboard.sheet_fallback":          "Table%d",
	"dashboard.sheet_default":           "Data Analysis",
	"dashboard.export_result_label":     "Analysis Results Export",
	"dashboard.generate_excel_failed":   "Excel generation failed: %v",
	"dashboard.generate_pdf_failed":     "PDF generation failed: %v",
	"dashboard.generate_ppt_failed":     "PPT generation failed: %v",
	"dashboard.generate_word_failed":    "Word generation failed: %v",
	"dashboard.refresh_failed":          "Refresh failed: %v",

	// Analysis Context
	"context.message_number":            "%s %s (Message #%d):\n%s",
	"context.tables_involved":           "📊 Tables involved: %s",
	"context.analysis_topic":            "🎯 Analysis topic: %s",
	"context.key_data":                  "📈 Key data: %s",

	// Analysis Errors
	"analysis.error_format":             "�**Error** [%s]\n\n%s",
	"analysis.timeout_detail":           "Analysis timed out (ran for %dm%ds). Please try simplifying the query or try again later.",
	"analysis.timeout_request":          "Analysis request timed out. Please try simplifying the query or try again later.",
	"analysis.network_error_msg":        "Network connection error. Please check your network connection and try again.",
	"analysis.database_error_msg":       "Database query error. Please check the data source configuration or query conditions.",
	"analysis.python_error_msg":         "Python execution error. Please check the analysis code or data format.",
	"analysis.llm_error_msg":            "AI model call error. Please check the API configuration or try again later.",
	"analysis.error_detail":             "An error occurred during analysis: %s",
	"analysis.cancelled_msg":            "⚠️ Analysis cancelled.",
	"analysis.error_with_detail":        "�**Analysis Error** [%s]\n\n%s\n\n<details><summary>Error Details</summary>\n\n```\n%s\n```\n</details>",
	"analysis.timing":                   "\n\n---\n⏱️ Analysis time: %dm%ds",
	"analysis.timing_check":             "⏱️ Analysis time:",
	"analysis.queue_wait":               "Waiting in analysis queue... (%d/%d tasks in progress)",
	"analysis.queue_timeout":            "Timeout waiting for analysis queue (waited %v). There are currently %d analysis tasks in progress. Please try again later.",
	"analysis.queue_wait_elapsed":       "Waiting in analysis queue... (waited %v, %d/%d tasks in progress)",
	"analysis.max_concurrent":           "There are currently %d analysis sessions in progress (max concurrent: %d). Please wait for some analyses to complete before starting a new analysis, or increase the max concurrent analysis limit in settings.",

	// Session
	"session.analysis_prefix":           "Analysis: %s",
	"session.analysis_prompt":           "Please analyze data source '%s' (%s), providing data overview, key metrics and insights.",

	// Location
	"location.label":                    "📍 Location: %s",

	// License Refresh
	"license_refresh.not_activated":     "Not activated, cannot refresh",
	"license_refresh.no_sn":             "Serial number not found",
	"license_refresh.no_server":         "License server address not found",
	"license_refresh.failed":            "Refresh failed: %v",
	"license_refresh.invalid_sn":        "Serial number is invalid, switched to open source software mode. Please use your own LLM API configuration.",
	"license_refresh.sn_expired":        "Serial number has expired, switched to open source software mode. Please use your own LLM API configuration.",
	"license_refresh.sn_disabled":       "Serial number has been disabled, switched to open source software mode. Please use your own LLM API configuration.",
	"license_refresh.default_invalid":   "License is no longer valid, switched to open source software mode. Please use your own LLM API configuration.",

	// Data Source Export Metadata
	"datasource.export_description":     "Data Source %s",
	"datasource.export_subject":         "Data Source Export",

	// Location Tool
	"location.current_city":             "Current location: %s, %s (accuracy: %.0fm)",
	"location.current_address":          "Current location: %s (accuracy: %.0fm)",
	"location.current_coords":           "Current location: lat %.6f, lon %.6f (accuracy: %.0fm)",
	"location.config":                   "User configured location: %s, %s",
	"location.ip_based":                 "IP-based location: %s, %s (accuracy: ~%.0fm)",
	"location.unavailable":              "Unable to get location: %s. Please ask the user for their city, or use a default city for the query.",

	// Export Tool
	"export.file_generated":             "�%s file generated: %s (%.2f KB)\n\nFile saved to session directory, available for download in the interface.",

	// Memory Extractor
	"memory.table_columns":              "Table %s contains columns: %s",
	"memory.field_values":               "Possible values for field %s: %s",

	// Exclusion Manager
	"exclusion.header":                  "Excluded %d analysis directions in %d categories:\n",
	"exclusion.footer":                  "Please understand user intent from other perspectives.",
	"exclusion.count_format":            "- %s (%d items)\n",

	// Context Memory
	"context.no_compressed_history":     "No compressed history (conversation short enough, all kept in short-term memory)",
	"context.ai_summary_header":         "📚 AI-generated conversation summary:",
}
