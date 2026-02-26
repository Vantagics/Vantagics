package i18n

var chineseTranslations = map[string]string{
	// 授权服务�
	"license.invalid_request":        "请求格式无效",
	"license.invalid_sn":             "序列号无�",
	"license.sn_disabled":            "序列号已被禁�",
	"license.sn_expired":             "序列号已过期",
	"license.encrypt_failed":         "加密失败",
	"license.invalid_email":          "请输入有效的邮箱地址",
	"license.email_blacklisted":      "您的邮箱已被限制",
	"license.email_not_whitelisted":  "您的邮箱不在白名单中",
	"license.email_already_used":     "您已经申请过序列�",
	"license.no_available_sn":        "没有可用的序列号，请联系管理�",
	"license.rate_limit_exceeded":    "今日请求次数已达上限，请明天再试",
	"license.email_limit_exceeded":   "该IP今日邮箱申请次数已达上限，请明天再试",
	"license.internal_error":         "服务器内部错�",
	"license.smtp_incomplete":        "SMTP配置不完�",
	"license.email_send_failed":      "发送邮件失败：%s",
	"license.email_sent":             "序列号已发送到您的邮箱",
	"license.sn_deleted":             "成功删除 %d 个未使用的序列号",
	"license.group_has_sn":           "此分组中还有 %d 个序列号，无法删�",
	"license.group_not_found":        "分组不存�",
	"license.sn_not_found":           "序列号不存在",
	"license.sn_updated":             "序列号更新成�",
	"license.sn_created":             "序列号创建成�",
	"license.group_created":          "分组创建成功",
	"license.group_updated":          "分组更新成功",
	"license.group_deleted":          "分组删除成功",
	"license.invalid_group_id":       "无效的分组ID",
	"license.invalid_sn_id":          "无效的序列号ID",
	"license.database_error":         "数据库操作失�",
	"license.activation_success":     "激活成�",
	"license.activation_failed":      "激活失败：%s",
	"license.deactivation_success":   "取消激活成�",
	"license.refresh_success":        "授权刷新成功",
	"license.refresh_failed":         "授权刷新失败�s",

	// 数据源操�
	"datasource.import_success":       "数据源导入成�",
	"datasource.import_failed":        "数据源导入失败：%s",
	"datasource.delete_success":       "数据源删除成�",
	"datasource.delete_failed":        "数据源删除失败：%s",
	"datasource.export_success":       "数据源导出成�",
	"datasource.export_failed":        "数据源导出失败：%s",
	"datasource.not_found":            "数据源不存在",
	"datasource.already_exists":       "数据源已存在",
	"datasource.invalid_name":         "无效的数据源名称",
	"datasource.connection_failed":    "连接失败�s",
	"datasource.test_success":         "连接测试成功",
	"datasource.semantic_opt_success": "语义优化完成",
	"datasource.semantic_opt_failed":  "语义优化失败�s",
	"datasource.semantic_opt_summary": "语义优化后的数据源，包含 %d 个表",

	// 分析操作
	"analysis.execution_failed":     "分析执行失败",
	"analysis.sql_error":            "SQL执行错误�s",
	"analysis.python_error":         "Python脚本执行错误�s",
	"analysis.timeout":              "分析超时",
	"analysis.cancelled":            "分析已被用户取消",
	"analysis.no_results":           "未找到结�",
	"analysis.export_success":       "分析结果导出成功",
	"analysis.export_failed":        "分析结果导出失败�s",
	"analysis.invalid_request":      "无效的分析请�",
	"analysis.queue_full":           "分析队列已满，请稍后再试",
	"analysis.in_progress":          "分析正在进行�",
	"analysis.report_gen_success":   "报告生成成功",
	"analysis.report_gen_failed":    "报告生成失败�s",

	// 文件操作
	"file.not_found":         "文件不存在：%s",
	"file.read_error":        "读取文件失败�s",
	"file.write_error":       "写入文件失败�s",
	"file.delete_error":      "删除文件失败�s",
	"file.invalid_format":    "无效的文件格�",
	"file.too_large":         "文件过大",
	"file.upload_success":    "文件上传成功",
	"file.upload_failed":     "文件上传失败�s",
	"file.download_success":  "文件下载成功",
	"file.download_failed":   "文件下载失败�s",

	// 数据库操�
	"db.connection_failed":   "数据库连接失败：%s",
	"db.query_error":         "数据库查询错误：%s",
	"db.insert_error":        "数据库插入错误：%s",
	"db.update_error":        "数据库更新错误：%s",
	"db.delete_error":        "数据库删除错误：%s",
	"db.transaction_failed":  "数据库事务失败：%s",
	"db.migration_failed":    "数据库迁移失败：%s",
	"db.backup_success":      "数据库备份成�",
	"db.backup_failed":       "数据库备份失败：%s",
	"db.restore_success":     "数据库恢复成�",
	"db.restore_failed":      "数据库恢复失败：%s",

	// Skills管理
	"skills.install_success":   "Skills安装成功�s",
	"skills.install_failed":    "Skills安装失败�s",
	"skills.enable_success":    "Skill启用成功",
	"skills.enable_failed":     "Skill启用失败�s",
	"skills.disable_success":   "Skill禁用成功",
	"skills.disable_failed":    "Skill禁用失败�s",
	"skills.delete_success":    "Skill删除成功",
	"skills.delete_failed":     "Skill删除失败�s",
	"skills.not_found":         "Skill不存�",
	"skills.already_exists":    "Skill已存�",
	"skills.invalid_package":   "无效的Skill�",
	"skills.load_failed":       "加载Skills失败�s",

	// Python环境
	"python.env_create_success":      "Python环境创建成功",
	"python.env_create_failed":       "Python环境创建失败�s",
	"python.package_install_success": "包安装成�",
	"python.package_install_failed":  "包安装失败：%s",
	"python.not_found":               "未找到Python",
	"python.invalid_version":         "无效的Python版本",
	"python.script_error":            "Python脚本执行错误�s",

	// 配置
	"config.load_failed":    "加载配置失败�s",
	"config.save_success":   "配置保存成功",
	"config.save_failed":    "配置保存失败�s",
	"config.invalid_value":  "无效的配置值：%s",
	"config.reset_success":  "配置已重置为默认�",

	// 认证与授�
	"auth.unauthorized":       "未授权访�",
	"auth.forbidden":          "访问被禁�",
	"auth.token_expired":      "认证令牌已过�",
	"auth.token_invalid":      "无效的认证令�",
	"auth.login_required":     "需要登�",
	"auth.permission_denied":  "权限不足",

	// 通用消息
	"general.success":           "操作成功",
	"general.failed":            "操作失败",
	"general.invalid_input":     "无效的输�",
	"general.required_field":    "必填字段�s",
	"general.not_found":         "资源不存�",
	"general.already_exists":    "资源已存�",
	"general.internal_error":    "服务器内部错�",
	"general.network_error":     "网络错误",
	"general.timeout":           "操作超时",
	"general.cancelled":         "操作已取�",
	"general.processing":        "处理�..",
	"general.please_wait":       "请稍�..",

	// 导出操作
	"export.pdf_success":    "PDF导出成功",
	"export.pdf_failed":     "PDF导出失败�s",
	"export.excel_success":  "Excel导出成功",
	"export.excel_failed":   "Excel导出失败�s",
	"export.word_success":   "Word文档导出成功",
	"export.word_failed":    "Word导出失败�s",
	"export.ppt_success":    "PowerPoint导出成功",
	"export.ppt_failed":     "PowerPoint导出失败�s",
	"export.html_success":   "HTML导出成功",
	"export.html_failed":    "HTML导出失败�s",
	"export.no_data":        "没有可导出的数据",
	"export.invalid_format": "无效的导出格�",

	// 导出文档内容
	"export.doc_description":      "�Vantagics 智能分析系统生成",
	"export.datasource_label":     "数据� ",
	"export.analysis_request":     "分析请求: ",
	"export.key_metrics":          "关键指标",
	"export.metric_column":        "指标",
	"export.value_column":         "数�",
	"export.change_column":        "变化",
	"export.data_tables":          "数据表格",
	"export.data_visualization":   "数据可视�",
	"export.chart_number":         "图表 %d / %d",
	"export.table_note":           "注：仅显示前 %d 行数据，�%d �",
	"export.table_extracted":      "[表格数据已提取]",
	"export.generated_by":         "�Vantagics 智能分析系统生成",

	// 报告导出
	"report.font_load_failed":       "无法加载中文字体",
	"report.data_analysis_report":   "数据分析报告",
	"report.data_source_label":      "数据�",
	"report.analysis_request_label": "分析请求",
	"report.generated_time_label":   "生成时间",
	"report.page_number":            "�%d �",

	// Excel 导出
	"excel.default_sheet_name":      "数据�",
	"excel.multi_table_title":       "多表数据分析",
	"excel.report_subject":          "数据分析报表",
	"excel.report_keywords":         "数据分析,报表,Excel",
	"excel.report_category":         "数据分析",

	// PPT 导出
	"ppt.key_metrics":               "关键指标",
	"ppt.data_visualization":        "数据可视�%d",
	"ppt.smart_insights":            "智能洞察",
	"ppt.smart_insights_continued":  "智能洞察（续 %d�",
	"ppt.data_tables":               "数据表格",
	"ppt.data_tables_page":          "数据表格（第 %d 页）",
	"ppt.table_info":                "显示�%d-%d 行，�%d �",
	"ppt.columns_truncated":         "（列数已截断�",
	"ppt.thank_you":                 "感谢查阅",
	"ppt.tagline":                   "数据驱动决策，智能赋能未�",
	"ppt.footer_year":               "Vantagics 智能分析系统 · %s",

	// MCP服务
	"mcp.service_not_found":    "MCP服务不存�",
	"mcp.connection_failed":    "MCP服务连接失败�s",
	"mcp.call_failed":          "MCP服务调用失败�s",
	"mcp.invalid_response":     "无效的MCP服务响应",
	"mcp.timeout":              "MCP服务超时",

	// 搜索API
	"search.api_not_configured": "搜索API未配�",
	"search.api_call_failed":    "搜索API调用失败�s",
	"search.no_results":         "未找到搜索结�",
	"search.invalid_query":      "无效的搜索查�",

	// 会话管理
	"session.create_success":  "会话创建成功",
	"session.create_failed":   "会话创建失败�s",
	"session.delete_success":  "会话删除成功",
	"session.delete_failed":   "会话删除失败�s",
	"session.not_found":       "会话不存�",
	"session.already_exists":  "会话已存�",
	"session.expired":         "会话已过�",

	// 表操�
	"table.delete_success":        "表删除成�",
	"table.delete_failed":         "表删除失败：%s",
	"table.not_found":             "表不存在",
	"table.column_delete_success": "列删除成�",
	"table.column_delete_failed":  "列删除失败：%s",
	"table.column_rename_success": "列重命名成功",
	"table.column_rename_failed":  "列重命名失败�s",

	// 仪表盘操�
	"dashboard.export_success":  "仪表盘导出成�",
	"dashboard.export_failed":   "仪表盘导出失败：%s",
	"dashboard.save_success":    "仪表盘布局保存成功",
	"dashboard.save_failed":     "仪表盘布局保存失败�s",
	"dashboard.no_data":         "没有可用的仪表盘数据",

	// 错误恢复建议
	"error.recovery.check_query":           "请检查您的查询是否清晰明�",
	"error.recovery.simplify_query":        "尝试简化查询条�",
	"error.recovery.refresh_retry":         "如果问题持续，请刷新页面后重�",
	"error.recovery.reduce_data_range":     "请尝试简化查询或减少数据范围",
	"error.recovery.check_network":         "检查网络连接是否稳�",
	"error.recovery.retry_later":           "稍后重试，系统可能正在处理其他任�",
	"error.recovery.resubmit":              "您可以重新发起分析请�",
	"error.recovery.check_data_format":     "请检查数据格式是否正�",
	"error.recovery.try_different_method":  "尝试使用不同的分析方�",
	"error.recovery.contact_support":       "如果问题持续，请联系技术支�",
	"error.recovery.rephrase_query":        "请尝试重新描述您的分析需�",
	"error.recovery.use_simpler_query":     "使用更简单的查询语句",
	"error.recovery.check_libraries":       "所需的分析库可能未安�",
	"error.recovery.check_admin":           "请联系管理员检查系统配�",
	"error.recovery.reduce_batch":          "尝试分批处理数据",
	"error.recovery.check_datasource":      "请检查数据源是否已正确配�",
	"error.recovery.check_table_field":     "确认查询的表或字段名称是否正�",
	"error.recovery.check_deleted":         "检查数据是否已被删除或移动",
	"error.recovery.check_data_type":       "确认数据类型是否正确",
	"error.recovery.clean_reimport":        "尝试清理或重新导入数�",
	"error.recovery.adjust_filters":        "请尝试调整筛选条�",
	"error.recovery.check_data_exists":     "检查数据源是否包含所需数据",
	"error.recovery.add_filters":           "添加更多筛选条�",
	"error.recovery.consider_pagination":   "考虑分页或分批查�",
	"error.recovery.check_service":         "确认服务是否正常运行",
	"error.recovery.check_permissions":     "您可能没有访问此资源的权�",
	"error.recovery.contact_admin":         "请联系管理员获取相应权限",
	"error.recovery.check_account":         "检查您的账户状�",
	"error.recovery.resource_busy":         "资源正在被其他任务使�",
	"error.recovery.check_path":            "请检查资源路径是否正�",
	"error.recovery.confirm_resource":      "联系管理员确认资源状�",

	// 错误消息
	"error.analysis_error":              "分析过程中发生错�",
	"error.analysis_timeout":            "分析超时，请稍后重试",
	"error.analysis_timeout_duration":   "分析超时（已运行 %v�",
	"error.analysis_cancelled":          "分析已取�",
	"error.python_execution":            "代码执行失败",
	"error.python_syntax":               "代码语法错误",
	"error.python_import":               "缺少必要的分析库",
	"error.python_memory":               "内存不足，数据量可能过大",
	"error.data_not_found":              "未找到请求的数据",
	"error.data_invalid":                "数据格式无效",
	"error.data_empty":                  "查询结果为空",
	"error.data_too_large":              "数据量超出限�",
	"error.connection_failed":           "连接失败，请检查网�",
	"error.connection_timeout":          "连接超时",
	"error.permission_denied":           "权限不足",
	"error.resource_busy":               "资源繁忙，请稍后重试",
	"error.resource_not_found":          "资源未找�",
	"error.unknown":                     "发生未知错误",

	// 应用程序对话�
	"app.about_title":                   "关于 万策",
	"app.about_message":                 "万策 (Vantagics)\n\n于万千数据中，定最优之策。\n\n版本�.0.0\n© 2026 Vantagics. All rights reserved.",
	"app.devtools_title":                "打开开发者工�",
	"app.devtools_message":              "请使用以下方法打开开发者工具：\n\n方法1：按 F12 键\n方法2：按 Ctrl+Shift+I\n方法3：按 Ctrl+Shift+J\n方法4：在空白区域右键点击，选择\"检查\"\n\n如果以上方法都不行，请在开发模式下运行：\nwails dev",
	"app.confirm_exit_title":            "确认退�",
	"app.confirm_exit_message":          "当前有正在进行的分析任务，确定要退出吗？\n\n退出将中断分析过程�",
	"app.exit_button":                   "退�",
	"app.cancel_button":                 "取消",
	"app.license_activation_failed":     "授权验证失败: %v\n请检查网络连接或联系管理员�",
	"app.license_refresh_failed":        "授权刷新失败: %v\n您的授权需要重新验证，请检查网络连接或联系管理员�",

	// 报告生成
	"report.save_dialog_title":          "保存分析报告",
	"report.filename_prefix":            "分析报告",
	"report.llm_not_initialized":        "LLM 服务未初始化，请先配�API Key",
	"report.generation_failed":          "报告生成失败: %v",
	"report.data_expired":               "报告数据已过期，请重新生�",
	"report.word_generation_failed":     "Word文档生成失败: %v",
	"report.pdf_generation_failed":      "PDF生成失败: %v",
	"report.no_content":                 "没有可导出的内容",
	"report.write_file_failed":          "写入文件失败: %v",

	// 报告章节
	"report.section.background":         "分析背景与目�",
	"report.section.data_overview":      "数据概况",
	"report.section.key_metrics":        "关键指标分析",
	"report.section.deep_analysis":      "深度数据分析",
	"report.section.findings":           "关键发现与洞�",
	"report.section.conclusions":        "结论与建�",

	// 报告标签（已删除重复�- 在上面已定义�
	"report.chart_label":                "图表 %d / %d",
	"report.total_rows":                 "总行� %d",
	"report.showing_columns":            "显示� %s",
	"report.category_label":             "分类",
	"report.footer_text":                "�%d �",

	// 报告错误（已删除重复�- 在上面已定义�

	// 综合报告
	"comprehensive_report.filename_prefix":    "综合报告",
	"comprehensive_report.save_dialog_title":  "保存综合报告",
	"comprehensive_report.no_valid_analysis":  "没有有效的分析结果可生成报告",
	"comprehensive_report.data_source":        "数据源：",
	"comprehensive_report.session_name":       "会话名称�",
	"comprehensive_report.all_analysis_results": "所有分析结果：",
	"comprehensive_report.analysis_request":   "分析请求",
	"comprehensive_report.analysis_result":    "分析结果",
	"comprehensive_report.insight":            "洞察",
	"comprehensive_report.key_metric":         "关键指标",
	"comprehensive_report.table":              "数据�",
	"comprehensive_report.pack_info_header":   "## 分析包信�",
	"comprehensive_report.pack_author":        "作者：",
	"comprehensive_report.pack_description":   "描述�",
	"comprehensive_report.pack_source_name":   "原始数据源：",

	// 分析技能包 - 预览与导�
	"qap.no_exportable_records":       "没有找到可导出的分析记录",
	"qap.unknown_request":             "(未知请求)",
	"qap.no_exportable_steps":         "所选分析请求没有可导出的操�",
	"qap.no_exportable_operations":    "该会话没有可导出的分析操�",
	"qap.load_pack_dialog_title":      "加载分析技能包",

	// 分析技能包 - 导入
	"qap.invalid_file_format":         "文件格式无效，无法解析分析技能包: %v",
	"qap.wrong_password":              "口令不正�",
	"qap.invalid_pack_file":           "文件格式无效: 不是有效的分析技能包文件",
	"qap.unsupported_version":         "不支持的分析包版� %s，请升级软件后重�",
	"qap.no_executable_steps":         "分析包中没有可执行的步骤",
	"qap.schema_fetch_failed":         "无法获取目标数据源的 schema: %v",
	"qap.missing_required_tables":     "目标数据源缺少必需的表: %s",
	"qap.python_not_configured":       "此分析包包含 Python 脚本，但尚未配置 Python 环境。请在设置中配置 Python 路径后重试�",
	"qap.permission_denied":           "权限不足: %s",
	"qap.not_replay_session":          "该会话不是分析技能包会话",

	// 分析技能包 - 步骤标签
	"qap.step_sql_query":              "SQL 查询 #%d",
	"qap.step_python_script":          "Python 脚本 #%d",
	"qap.step_generic":                "步骤 #%d",
	"qap.step_execution_failed":       "步骤 %d 执行失败: %v",
	"qap.step_python_not_configured":  "步骤 %d 执行失败: Python 环境未配�",

	// 分析技能包 - 导入验证
	"qap.table_not_exists":            "�'%s' 不存�",
	"qap.column_not_exists":           "字段 '%s.%s' 不存�",
	"qap.step_sql_success":            "执行SQL成功 (步骤 %d):\n\n```json:table\n%s\n```",
	"qap.step_sql_failed":             "执行SQL失败�v\n\nSQL:\n```sql\n%s\n```",
	"qap.step_execute_query":          "执行步骤 %d 的分析查�",

	// 分析技能包 - 执行消息
	"qap.step_skipped":                "⏭️ 步骤 %d (%s) 已跳过：依赖的前置步骤执行失�",
	"qap.execution_complete":          "�分析技能包执行完成！共执行�%d 个步骤�",
	"qap.reexecution_complete":        "�分析技能包重新执行完成！共执行�%d 个步骤�",
	"qap.step_sql_error":              "�步骤 %d (%s) 执行失败�v\n\n> 📋 分析请求�s\n\n```sql\n%s\n```",
	"qap.step_sql_success_full":       "�步骤 %d (%s):\n\n> 📋 分析请求�s\n\n```json:table\n%s\n```",
	"qap.step_sql_success_truncated":  "�步骤 %d (%s) (�%d 行，显示�20 �:\n\n> 📋 分析请求�s\n\n```json:table\n%s\n```",
	"qap.step_python_no_env":          "�步骤 %d (%s) 执行失败：Python 环境未配置\n\n> 📋 分析请求�s\n\n```python\n%s\n```",
	"qap.step_python_error":           "�步骤 %d (%s) 执行失败�v\n\n> 📋 分析请求�s\n\n```python\n%s\n```",
	"qap.step_python_success":         "�步骤 %d (%s):\n\n> 📋 分析请求�s\n\n```\n%s\n```",

	// 分析导出
	"analysis_export.description":     "Vantagics 分析过程导出文件 - 包含可执行的 SQL/Python 步骤",
	"analysis_export.dialog_title":    "导出分析过程",

	// 仪表盘导�
	"dashboard.no_exportable_content": "没有可导出的内容",
	"dashboard.message_not_found":     "消息不存� %v",
	"dashboard.step_no_results":       "该步骤没有可显示的结�",
	"dashboard.session_no_results":    "该会话没有可显示的结�",
	"dashboard.mode_switch_blocked":   "当前有正在进行的分析任务，无法切换模�",
	"dashboard.write_pdf_failed":      "写入PDF文件失败: %v",
	"dashboard.write_excel_failed":    "写入Excel文件失败: %v",
	"dashboard.write_ppt_failed":      "写入PPT文件失败: %v",
	"dashboard.write_word_failed":     "写入Word文件失败: %v",

	// 市场
	"marketplace.insufficient_credits": "积分余额不足，需�%d 积分，当前余�%.0f 积分",

	// 数据源导�
	"datasource.unsupported_format":   "不支持的文件格式: %s。请使用 .xlsx �.xls 格式�Excel 文件",
	"datasource.excel_format_error":   "无法打开 Excel 文件：文件格式不受支持。请确保文件是有效的 .xlsx 格式（Excel 2007 或更高版本）",
	"datasource.excel_open_failed":    "无法打开 Excel 文件: %v",
	"datasource.no_sheets":            "Excel 文件中没有找到工作表",
	"datasource.no_valid_data":        "Excel 文件中没有找到有效数�",

	// 意图生成
	"intent.generation_failed":        "意图生成失败: %v",
	"intent.parse_failed":             "响应解析失败: %v",
	"intent.no_suggestions":           "未能生成意图建议",

	// 授权客户�
	"license_client.build_request_failed":  "构建请求失败: %v",
	"license_client.connect_failed":        "连接服务器失� %v",
	"license_client.read_response_failed":  "读取响应失败: %v",
	"license_client.parse_response_failed": "解析响应失败: %v",
	"license_client.decrypt_failed":        "解密失败: %v",
	"license_client.parse_config_failed":   "解析配置失败: %v",
	"license_client.credits_insufficient":  "Credits 不足，剩�%.1f credits，每次分析需�%.1f credits",
	"license_client.daily_limit_reached":   "今日分析次数已达上限�d次），请明天再试",
	"license_client.first_use":             "首次使用，需要验证授�",
	"license_client.trial_label":           "试用�",
	"license_client.official_label":        "正式�",
	"license_client.refresh_needed":        "%s授权需要刷新（已超�d天）",

	// 使用授权
	"usage.expired":                   "使用权限已过期，请续�",
	"usage.uses_exhausted":            "使用次数已用完，请重新购�",

	// PDF 字体
	"pdf.font_load_failed":            "无法加载中文字体",

	// 工具结果
	"tool.no_valid_info":              "抱歉，未能获取到有效信息�",

	// SQL 验证
	"sql.readonly_violation":          "检测到非只读SQL操作: %s (只允许SELECT查询)",

	// 仪表盘导出对话框
	"dashboard.export_pdf_title":        "导出仪表盘为PDF",
	"dashboard.export_excel_title":      "导出仪表盘数据为Excel",
	"dashboard.export_ppt_title":        "导出仪表盘为PPT",
	"dashboard.export_word_title":       "导出仪表盘为Word",
	"dashboard.export_table_title":      "导出表格为Excel",
	"dashboard.export_message_pdf_title": "导出分析结果为PDF",
	"dashboard.filter_pdf":              "PDF文件",
	"dashboard.filter_excel":            "Excel文件",
	"dashboard.filter_ppt":              "PowerPoint文件",
	"dashboard.filter_word":             "Word文件",
	"dashboard.sheet_fallback":          "表格%d",
	"dashboard.sheet_default":           "数据分析",
	"dashboard.export_result_label":     "分析结果导出",
	"dashboard.generate_excel_failed":   "Excel生成失败: %v",
	"dashboard.generate_pdf_failed":     "PDF生成失败: %v",
	"dashboard.generate_ppt_failed":     "PPT生成失败: %v",
	"dashboard.generate_word_failed":    "Word生成失败: %v",
	"dashboard.refresh_failed":          "刷新失败: %v",

	// 分析上下�
	"context.message_number":            "%s %s (消息 #%d):\n%s",
	"context.tables_involved":           "📊 涉及数据� %s",
	"context.analysis_topic":            "🎯 分析主题: %s",
	"context.key_data":                  "📈 关键数据: %s",

	// 分析错误
	"analysis.error_format":             "�**错误** [%s]\n\n%s",
	"analysis.timeout_detail":           "分析超时（已运行 %d�d秒）。请尝试简化查询或稍后重试�",
	"analysis.timeout_request":          "分析请求超时。请尝试简化查询或稍后重试�",
	"analysis.network_error_msg":        "网络连接错误。请检查网络连接后重试�",
	"analysis.database_error_msg":       "数据库查询错误。请检查数据源配置或查询条件�",
	"analysis.python_error_msg":         "Python 执行错误。请检查分析代码或数据格式�",
	"analysis.llm_error_msg":            "AI 模型调用错误。请检�API 配置或稍后重试�",
	"analysis.error_detail":             "分析过程中发生错� %s",
	"analysis.cancelled_msg":            "⚠️ 分析已取消�",
	"analysis.error_with_detail":        "�**分析出错** [%s]\n\n%s\n\n<details><summary>详细错误信息</summary>\n\n```\n%s\n```\n</details>",
	"analysis.timing":                   "\n\n---\n⏱️ 分析耗时: %d�d�",
	"analysis.timing_check":             "⏱️ 分析耗时:",
	"analysis.queue_wait":               "等待分析队列�..（当�%d/%d 个任务进行中�",
	"analysis.queue_timeout":            "等待分析队列超时（已等待 %v）。当前有 %d 个分析任务进行中。请稍后重试�",
	"analysis.queue_wait_elapsed":       "等待分析队列�..（已等待 %v，当�%d/%d 个任务进行中�",
	"analysis.max_concurrent":           "当前已有 %d 个分析会话进行中（最大并发数�d）。请等待部分分析完成后再开始新的分析，或在设置中增加最大并发分析任务数�",

	// 会话
	"session.analysis_prefix":           "分析: %s",
	"session.analysis_prompt":           "请分析数据源 '%s' (%s)，提供数据概览、关键指标和洞察�",

	// 位置
	"location.label":                    "📍 位置: %s",

	// 授权刷新
	"license_refresh.not_activated":     "未激活，无法刷新",
	"license_refresh.no_sn":             "未找到序列号",
	"license_refresh.no_server":         "未找到授权服务器地址",
	"license_refresh.failed":            "刷新失败: %v",
	"license_refresh.invalid_sn":        "序列号无效，已切换到开源软件模式。请使用您自己的 LLM API 配置�",
	"license_refresh.sn_expired":        "序列号已过期，已切换到开源软件模式。请使用您自己的 LLM API 配置�",
	"license_refresh.sn_disabled":       "序列号已被禁用，已切换到开源软件模式。请使用您自己的 LLM API 配置�",
	"license_refresh.default_invalid":   "授权已失效，已切换到开源软件模式。请使用您自己的 LLM API 配置�",

	// 数据源导出元数据
	"datasource.export_description":     "数据�%s",
	"datasource.export_subject":         "数据源导�",

	// 位置工具
	"location.current_city":             "当前位置: %s, %s (精度: %.0f�",
	"location.current_address":          "当前位置: %s (精度: %.0f�",
	"location.current_coords":           "当前位置: 纬度 %.6f, 经度 %.6f (精度: %.0f�",
	"location.config":                   "用户设置位置: %s, %s",
	"location.ip_based":                 "IP定位: %s, %s (精度: �.0f�",
	"location.unavailable":              "无法获取位置信息: %s。请直接询问用户所在城市，或使用默认城市（如北京）进行查询�",

	// 导出工具
	"export.file_generated":             "�%s文件已生� %s (%.2f KB)\n\n文件已保存到会话目录，可以在界面中下载�",

	// 内存提取�
	"memory.table_columns":              "�%s 包含字段: %s",
	"memory.field_values":               "字段 %s 的可能� %s",

	// 排除管理�
	"exclusion.header":                  "已排�d个分析方向，分为%d类：\n",
	"exclusion.footer":                  "请从其他角度理解用户意图�",
	"exclusion.count_format":            "- %s�d项）\n",

	// 上下文记�
	"context.no_compressed_history":     "暂无压缩历史（对话足够短，全部保留在短期记忆中）",
	"context.ai_summary_header":         "📚 AI 自动生成的对话摘�",
}
