/**
 * Data Normalizer
 * 
 * 数据规范化器，负责将不同格式的数据转换为统一格式
 * 
 * 增强功能 (Task 5.1):
 * - ECharts 配置验证
 * - 表格数据验证
 * - 指标数据验证
 * - 调试日志
 */

import {
  AnalysisResultType,
  NormalizedResult,
  ValidationResult,
  NormalizedTableData,
  NormalizedMetricData,
  NormalizedInsightData,
  NormalizedFileData,
} from '../types/AnalysisResult';
import { createLogger } from './systemLog';

const logger = createLogger('DataNormalizer');

// ============================================================================
// 类型检测方法 (Task 5.1)
// ============================================================================

/**
 * 验证是否为有效的 ECharts 配置
 * 检查 series, xAxis, yAxis, tooltip, legend 等特征字段
 * 
 * @param data - 待验证的数据
 * @returns 是否为有效的 ECharts 配置
 * 
 * 验证: 需求 1.1 - ECharts 图表配置能正确解析并渲染
 */
export function isValidEChartsConfig(data: any): boolean {
  logger.debug('[TypeDetection] Checking ECharts config validity');
  
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    logger.debug('[TypeDetection] ECharts validation failed: not an object');
    return false;
  }
  
  // ECharts 配置的特征字段
  const echartsFields = ['series', 'xAxis', 'yAxis', 'tooltip', 'legend', 'grid', 'title', 'dataset', 'radar', 'polar', 'geo', 'visualMap', 'dataZoom'];
  
  // 检查是否包含至少一个 ECharts 特征字段
  const hasEChartsField = echartsFields.some(field => field in data);
  
  if (!hasEChartsField) {
    logger.debug('[TypeDetection] ECharts validation failed: no characteristic fields found');
    return false;
  }
  
  // 如果有 series 字段，验证其结构
  if ('series' in data) {
    const series = data.series;
    
    // series 可以是数组或单个对象
    if (Array.isArray(series)) {
      // 验证数组中的每个 series 项
      const validSeries = series.every((s: any) => {
        if (!s || typeof s !== 'object') return false;
        // series 项通常有 type 或 data 字段
        return 'type' in s || 'data' in s || 'name' in s;
      });
      
      if (!validSeries && series.length > 0) {
        logger.debug('[TypeDetection] ECharts validation warning: series array contains invalid items');
      }
    } else if (series !== null && typeof series === 'object') {
      // 单个 series 对象
      if (!('type' in series) && !('data' in series)) {
        logger.debug('[TypeDetection] ECharts validation warning: series object missing type or data');
      }
    }
  }
  
  // 验证 xAxis/yAxis 结构（如果存在）
  if ('xAxis' in data && data.xAxis !== null) {
    const xAxis = Array.isArray(data.xAxis) ? data.xAxis[0] : data.xAxis;
    if (xAxis && typeof xAxis === 'object') {
      logger.debug(`[TypeDetection] ECharts xAxis type: ${xAxis.type || 'default'}`);
    }
  }
  
  if ('yAxis' in data && data.yAxis !== null) {
    const yAxis = Array.isArray(data.yAxis) ? data.yAxis[0] : data.yAxis;
    if (yAxis && typeof yAxis === 'object') {
      logger.debug(`[TypeDetection] ECharts yAxis type: ${yAxis.type || 'default'}`);
    }
  }
  
  logger.debug('[TypeDetection] ECharts validation passed');
  return true;
}

/**
 * 验证是否为有效的表格数据
 * 支持多种格式：
 * - { columns: [], rows: [] } 标准格式
 * - { headers: [], data: [] } 替代格式
 * - 对象数组 [{ key: value }, ...]
 * - 二维数组 [[header1, header2], [val1, val2], ...]
 * 
 * @param data - 待验证的数据
 * @returns 是否为有效的表格数据
 * 
 * 验证: 需求 2.1 - 表格数据能正确解析和显示
 */
export function isValidTableData(data: any): boolean {
  logger.debug('[TypeDetection] Checking table data validity');
  
  if (!data) {
    logger.debug('[TypeDetection] Table validation failed: data is null/undefined');
    return false;
  }
  
  // 格式1: 标准格式 { columns: [], rows: [] }
  if (typeof data === 'object' && !Array.isArray(data)) {
    if ('columns' in data && 'rows' in data) {
      const validColumns = Array.isArray(data.columns);
      const validRows = Array.isArray(data.rows);
      
      if (validColumns && validRows) {
        logger.debug(`[TypeDetection] Table validation passed: standard format with ${data.columns.length} columns, ${data.rows.length} rows`);
        return true;
      }
    }
    
    // 格式2: 替代格式 { headers: [], data: [] }
    if ('headers' in data && 'data' in data) {
      const validHeaders = Array.isArray(data.headers);
      const validData = Array.isArray(data.data);
      
      if (validHeaders && validData) {
        logger.debug(`[TypeDetection] Table validation passed: headers/data format with ${data.headers.length} headers, ${data.data.length} rows`);
        return true;
      }
    }
  }
  
  // 格式3: 对象数组 [{ key: value }, ...]
  if (Array.isArray(data) && data.length > 0) {
    const firstItem = data[0];
    
    // 检查是否为对象数组
    if (firstItem && typeof firstItem === 'object' && !Array.isArray(firstItem)) {
      const keys = Object.keys(firstItem);
      if (keys.length > 0) {
        logger.debug(`[TypeDetection] Table validation passed: object array with ${keys.length} columns, ${data.length} rows`);
        return true;
      }
    }
    
    // 格式4: 二维数组 [[header1, header2], [val1, val2], ...]
    if (Array.isArray(firstItem)) {
      // 检查是否所有行都是数组
      const allArrays = data.every((row: any) => Array.isArray(row));
      if (allArrays) {
        logger.debug(`[TypeDetection] Table validation passed: 2D array with ${firstItem.length} columns, ${data.length} rows`);
        return true;
      }
    }
  }
  
  logger.debug('[TypeDetection] Table validation failed: unrecognized format');
  return false;
}

/**
 * 验证是否为有效的指标数据
 * 检查 title/name 和 value 字段
 * 
 * @param data - 待验证的数据
 * @returns 是否为有效的指标数据
 * 
 * 验证: 需求 3.1 - 指标卡片正确显示标题、数值、变化
 */
export function isValidMetricData(data: any): boolean {
  logger.debug('[TypeDetection] Checking metric data validity');
  
  if (!data || typeof data !== 'object' || Array.isArray(data)) {
    logger.debug('[TypeDetection] Metric validation failed: not an object');
    return false;
  }
  
  // 检查必要字段：title 或 name
  const hasTitle = 'title' in data || 'name' in data;
  if (!hasTitle) {
    logger.debug('[TypeDetection] Metric validation failed: missing title/name field');
    return false;
  }
  
  // 检查必要字段：value
  const hasValue = 'value' in data;
  if (!hasValue) {
    logger.debug('[TypeDetection] Metric validation failed: missing value field');
    return false;
  }
  
  // 验证 value 不为 undefined
  if (data.value === undefined) {
    logger.debug('[TypeDetection] Metric validation failed: value is undefined');
    return false;
  }
  
  // 可选字段检查（用于调试日志）
  const hasChange = 'change' in data;
  const hasUnit = 'unit' in data;
  
  logger.debug(`[TypeDetection] Metric validation passed: title="${data.title || data.name}", value="${data.value}", hasChange=${hasChange}, hasUnit=${hasUnit}`);
  return true;
}

/**
 * 自动检测数据类型
 * 根据数据结构特征推断最可能的类型
 * 
 * @param data - 待检测的数据
 * @returns 推断的数据类型，如果无法确定则返回 null
 */
export function detectDataType(data: any): AnalysisResultType | null {
  logger.debug('[TypeDetection] Auto-detecting data type');
  
  if (!data) {
    logger.debug('[TypeDetection] Detection failed: data is null/undefined');
    return null;
  }
  
  // 字符串类型检测
  if (typeof data === 'string') {
    // 检查是否为图片 data URL
    if (data.startsWith('data:image/')) {
      logger.debug('[TypeDetection] Detected type: image (data URL)');
      return 'image';
    }
    
    // 检查是否为 base64 图片
    if (data.startsWith('iVBORw0KGgo') || data.startsWith('/9j/') || 
        data.startsWith('R0lGOD') || data.startsWith('UklGR')) {
      logger.debug('[TypeDetection] Detected type: image (base64)');
      return 'image';
    }
    
    // 检查是否为 CSV
    if (data.includes(',') && data.includes('\n')) {
      const lines = data.split('\n').filter((l: string) => l.trim());
      if (lines.length > 1) {
        const firstLineCommas = (lines[0].match(/,/g) || []).length;
        const secondLineCommas = (lines[1].match(/,/g) || []).length;
        if (firstLineCommas > 0 && firstLineCommas === secondLineCommas) {
          logger.debug('[TypeDetection] Detected type: csv');
          return 'csv';
        }
      }
    }
    
    // 尝试解析为 JSON
    try {
      const parsed = JSON.parse(data);
      return detectDataType(parsed);
    } catch {
      // 不是有效的 JSON
      logger.debug('[TypeDetection] String is not valid JSON, treating as insight');
      return 'insight';
    }
  }
  
  // 对象类型检测
  if (typeof data === 'object' && !Array.isArray(data)) {
    // 优先检测 ECharts（因为 ECharts 配置也是对象）
    if (isValidEChartsConfig(data)) {
      logger.debug('[TypeDetection] Detected type: echarts');
      return 'echarts';
    }
    
    // 检测指标数据
    if (isValidMetricData(data)) {
      logger.debug('[TypeDetection] Detected type: metric');
      return 'metric';
    }
    
    // 检测表格数据（标准格式）
    if (isValidTableData(data)) {
      logger.debug('[TypeDetection] Detected type: table');
      return 'table';
    }
    
    // 检测文件数据
    if ('fileName' in data || 'filePath' in data || ('name' in data && 'path' in data)) {
      logger.debug('[TypeDetection] Detected type: file');
      return 'file';
    }
    
    // 检测洞察数据
    if ('text' in data) {
      logger.debug('[TypeDetection] Detected type: insight');
      return 'insight';
    }
  }
  
  // 数组类型检测
  if (Array.isArray(data)) {
    if (isValidTableData(data)) {
      logger.debug('[TypeDetection] Detected type: table (array format)');
      return 'table';
    }
  }
  
  logger.debug('[TypeDetection] Could not determine data type');
  return null;
}

// ============================================================================
// 规范化方法
// ============================================================================

/**
 * 规范化 ECharts 数据
 * 输入: JSON字符串或对象
 * 输出: 解析后的ECharts配置对象
 * 
 * 增强: 使用 isValidEChartsConfig 进行验证
 */
export function normalizeECharts(data: string | object): NormalizedResult<object> {
  logger.debug('[Normalize] Processing ECharts data');
  
  try {
    let parsed: object;
    
    if (typeof data === 'string') {
      logger.debug('[Normalize] ECharts input is string, parsing JSON');
      // 尝试解析JSON字符串
      parsed = JSON.parse(data);
    } else if (typeof data === 'object' && data !== null) {
      parsed = data;
    } else {
      logger.warn('[Normalize] Invalid ECharts data: expected string or object');
      return { success: false, error: 'Invalid ECharts data: expected string or object' };
    }
    
    // 基本验证：检查是否有必要的ECharts属性
    if (!parsed || typeof parsed !== 'object') {
      logger.warn('[Normalize] Invalid ECharts data: parsed result is not an object');
      return { success: false, error: 'Invalid ECharts data: parsed result is not an object' };
    }
    
    // 使用增强的验证方法
    if (!isValidEChartsConfig(parsed)) {
      logger.warn('[Normalize] Data does not appear to be a valid ECharts config, but proceeding anyway');
      // 仍然返回成功，因为可能是简化的配置
    } else {
      logger.debug('[Normalize] ECharts config validation passed');
    }
    
    return { success: true, data: parsed };
  } catch (error) {
    logger.error(`Failed to normalize ECharts data: ${error}`);
    return { success: false, error: `Failed to parse ECharts data: ${error}` };
  }
}

/**
 * 规范化图片数据
 * 输入: base64字符串（可能带或不带data URL前缀）
 * 输出: 完整的data URL格式
 */
export function normalizeImage(data: string): NormalizedResult<string> {
  logger.debug('[Normalize] Processing image data');
  
  try {
    if (typeof data !== 'string' || !data) {
      logger.warn('[Normalize] Invalid image data: expected non-empty string');
      return { success: false, error: 'Invalid image data: expected non-empty string' };
    }
    
    // 如果已经是完整的data URL格式
    if (data.startsWith('data:image/')) {
      logger.debug('[Normalize] Image already in data URL format');
      return { success: true, data };
    }
    
    // 检测图片类型并添加前缀
    // 尝试从base64数据检测图片类型
    let mimeType = 'image/png'; // 默认PNG
    
    // PNG: 以 iVBORw0KGgo 开头
    if (data.startsWith('iVBORw0KGgo')) {
      mimeType = 'image/png';
      logger.debug('[Normalize] Detected image type: PNG');
    }
    // JPEG: 以 /9j/ 开头
    else if (data.startsWith('/9j/')) {
      mimeType = 'image/jpeg';
      logger.debug('[Normalize] Detected image type: JPEG');
    }
    // GIF: 以 R0lGOD 开头
    else if (data.startsWith('R0lGOD')) {
      mimeType = 'image/gif';
      logger.debug('[Normalize] Detected image type: GIF');
    }
    // WebP: 以 UklGR 开头
    else if (data.startsWith('UklGR')) {
      mimeType = 'image/webp';
      logger.debug('[Normalize] Detected image type: WebP');
    } else {
      logger.debug('[Normalize] Could not detect image type, defaulting to PNG');
    }
    
    return { success: true, data: `data:${mimeType};base64,${data}` };
  } catch (error) {
    logger.error(`Failed to normalize image data: ${error}`);
    return { success: false, error: `Failed to normalize image data: ${error}` };
  }
}

/**
 * 规范化表格数据
 * 输入: JSON字符串、数组或对象
 * 输出: { columns: string[], rows: object[] }
 * 
 * 增强: 使用 isValidTableData 进行验证
 */
export function normalizeTable(data: any): NormalizedResult<NormalizedTableData> {
  logger.debug('[Normalize] Processing table data');
  
  try {
    let parsed: any;
    
    // 解析字符串
    if (typeof data === 'string') {
      logger.debug('[Normalize] Table input is string, parsing JSON');
      // 清理可能的formatter函数（ECharts残留）
      let cleanedData = data
        .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
        .replace(/,(\s*[}\]])/g, '$1');
      parsed = JSON.parse(cleanedData);
    } else {
      parsed = data;
    }
    
    // 使用增强的验证方法进行预检查
    if (!isValidTableData(parsed)) {
      logger.debug('[Normalize] Data does not match known table formats, attempting conversion');
    }
    
    // 处理数组格式
    if (Array.isArray(parsed) && parsed.length > 0) {
      const firstRow = parsed[0];
      
      // 如果第一行是对象，提取列名
      if (typeof firstRow === 'object' && firstRow !== null && !Array.isArray(firstRow)) {
        const columns = Object.keys(firstRow);
        logger.debug(`[Normalize] Table converted from object array: ${columns.length} columns, ${parsed.length} rows`);
        return {
          success: true,
          data: {
            columns,
            rows: parsed
          }
        };
      }
      
      // 如果是二维数组，第一行作为列名
      if (Array.isArray(firstRow)) {
        const columns = firstRow.map(String);
        const rows = parsed.slice(1).map((row: any[]) => {
          const obj: Record<string, any> = {};
          columns.forEach((col, idx) => {
            obj[col] = row[idx];
          });
          return obj;
        });
        logger.debug(`[Normalize] Table converted from 2D array: ${columns.length} columns, ${rows.length} rows`);
        return { success: true, data: { columns, rows } };
      }
    }
    
    // 处理已经是 { columns, rows } 格式的数据
    if (parsed && typeof parsed === 'object' && 'columns' in parsed && 'rows' in parsed) {
      logger.debug(`[Normalize] Table already in standard format: ${parsed.columns.length} columns, ${parsed.rows.length} rows`);
      return { success: true, data: parsed as NormalizedTableData };
    }
    
    // 处理 { headers, data } 格式
    if (parsed && typeof parsed === 'object' && 'headers' in parsed && 'data' in parsed) {
      logger.debug(`[Normalize] Converting headers/data format to standard format`);
      return {
        success: true,
        data: {
          columns: parsed.headers,
          rows: parsed.data
        }
      };
    }
    
    // 空数据
    logger.debug('[Normalize] Table data is empty or unrecognized, returning empty table');
    return { success: true, data: { columns: [], rows: [] } };
  } catch (error) {
    logger.error(`Failed to normalize table data: ${error}`);
    return { success: false, error: `Failed to parse table data: ${error}` };
  }
}

/**
 * 规范化CSV数据
 * 输入: CSV字符串或base64 data URL
 * 输出: 与table相同的格式
 */
export function normalizeCSV(data: string): NormalizedResult<NormalizedTableData> {
  logger.debug('[Normalize] Processing CSV data');
  
  try {
    let csvContent = data;
    
    // 如果是data URL格式，提取内容
    if (data.startsWith('data:')) {
      logger.debug('[Normalize] CSV is in data URL format, extracting content');
      const base64Match = data.match(/base64,(.+)/);
      if (base64Match) {
        csvContent = atob(base64Match[1]);
      }
    }
    
    // 解析CSV
    const lines = csvContent.split(/\r?\n/).filter(line => line.trim());
    if (lines.length === 0) {
      logger.debug('[Normalize] CSV is empty');
      return { success: true, data: { columns: [], rows: [] } };
    }
    
    // 第一行作为列名
    const columns = parseCSVLine(lines[0]);
    
    // 其余行作为数据
    const rows = lines.slice(1).map(line => {
      const values = parseCSVLine(line);
      const obj: Record<string, any> = {};
      columns.forEach((col, idx) => {
        obj[col] = values[idx] ?? '';
      });
      return obj;
    });
    
    logger.debug(`[Normalize] CSV parsed: ${columns.length} columns, ${rows.length} rows`);
    return { success: true, data: { columns, rows } };
  } catch (error) {
    logger.error(`Failed to normalize CSV data: ${error}`);
    return { success: false, error: `Failed to parse CSV data: ${error}` };
  }
}

/**
 * 解析CSV行（处理引号和逗号）
 */
function parseCSVLine(line: string): string[] {
  const result: string[] = [];
  let current = '';
  let inQuotes = false;
  
  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    
    if (char === '"') {
      if (inQuotes && line[i + 1] === '"') {
        current += '"';
        i++;
      } else {
        inQuotes = !inQuotes;
      }
    } else if (char === ',' && !inQuotes) {
      result.push(current.trim());
      current = '';
    } else {
      current += char;
    }
  }
  
  result.push(current.trim());
  return result;
}

/**
 * 规范化指标数据
 * 输入: { name, value, unit?, change? } 或类似格式
 * 输出: { title, value, change }
 * 
 * 增强: 使用 isValidMetricData 进行验证
 */
export function normalizeMetric(data: any): NormalizedResult<NormalizedMetricData> {
  logger.debug('[Normalize] Processing metric data');
  
  try {
    if (!data || typeof data !== 'object') {
      logger.warn('[Normalize] Invalid metric data: expected object');
      return { success: false, error: 'Invalid metric data: expected object' };
    }
    
    // 使用增强的验证方法
    if (!isValidMetricData(data)) {
      logger.warn('[Normalize] Metric data validation failed');
      return { success: false, error: 'Invalid metric data: missing required fields (title/name and value)' };
    }
    
    const title = data.name || data.title || '';
    let value = data.value;
    
    // 处理带单位的值
    if (data.unit && typeof value !== 'string') {
      value = `${value}${data.unit}`;
    } else {
      value = String(value ?? '');
    }
    
    logger.debug(`[Normalize] Metric normalized: title="${title}", value="${value}"`);
    
    return {
      success: true,
      data: {
        title,
        value,
        change: data.change || '',
        unit: data.unit
      }
    };
  } catch (error) {
    logger.error(`Failed to normalize metric data: ${error}`);
    return { success: false, error: `Failed to normalize metric data: ${error}` };
  }
}

/**
 * 规范化洞察数据
 * 输入: string 或 { text, icon?, data_source_id? }
 * 输出: { text, icon, dataSourceId? }
 */
export function normalizeInsight(data: any): NormalizedResult<NormalizedInsightData> {
  logger.debug('[Normalize] Processing insight data');
  
  try {
    if (typeof data === 'string') {
      logger.debug('[Normalize] Insight is plain string');
      return {
        success: true,
        data: {
          text: data,
          icon: 'lightbulb'
        }
      };
    }
    
    if (data && typeof data === 'object') {
      logger.debug(`[Normalize] Insight normalized: text="${(data.text || '').substring(0, 50)}..."`);
      return {
        success: true,
        data: {
          text: data.text || '',
          icon: data.icon || 'lightbulb',
          dataSourceId: data.data_source_id || data.dataSourceId,
          sourceName: data.source_name || data.sourceName
        }
      };
    }
    
    logger.warn('[Normalize] Invalid insight data: expected string or object');
    return { success: false, error: 'Invalid insight data: expected string or object' };
  } catch (error) {
    logger.error(`Failed to normalize insight data: ${error}`);
    return { success: false, error: `Failed to normalize insight data: ${error}` };
  }
}

/**
 * 规范化文件数据
 */
export function normalizeFile(data: any): NormalizedResult<NormalizedFileData> {
  logger.debug('[Normalize] Processing file data');
  
  try {
    if (!data || typeof data !== 'object') {
      logger.warn('[Normalize] Invalid file data: expected object');
      return { success: false, error: 'Invalid file data: expected object' };
    }
    
    const fileName = data.name || data.fileName || '';
    const filePath = data.path || data.filePath || '';
    
    logger.debug(`[Normalize] File normalized: name="${fileName}", path="${filePath}"`);
    
    return {
      success: true,
      data: {
        fileName,
        filePath,
        fileType: data.type || data.fileType || '',
        size: data.size,
        preview: data.preview
      }
    };
  } catch (error) {
    logger.error(`Failed to normalize file data: ${error}`);
    return { success: false, error: `Failed to normalize file data: ${error}` };
  }
}

/**
 * 统一入口方法：根据类型规范化数据
 */
export function normalize(type: AnalysisResultType, rawData: any): NormalizedResult {
  logger.debug(`[Normalize] Processing data of type: ${type}`);
  
  switch (type) {
    case 'echarts':
      return normalizeECharts(rawData);
    case 'image':
      return normalizeImage(rawData);
    case 'table':
      return normalizeTable(rawData);
    case 'csv':
      return normalizeCSV(rawData);
    case 'metric':
      return normalizeMetric(rawData);
    case 'insight':
      return normalizeInsight(rawData);
    case 'file':
      return normalizeFile(rawData);
    default:
      logger.warn(`Unknown data type: ${type}`);
      return { success: false, error: `Unknown data type: ${type}` };
  }
}

/**
 * 验证数据是否符合类型要求
 * 
 * 增强: 使用新的验证方法进行更严格的检查
 */
export function validate(type: AnalysisResultType, data: any): ValidationResult {
  logger.debug(`[Validate] Validating data of type: ${type}`);
  
  const errors: string[] = [];
  
  switch (type) {
    case 'echarts':
      if (typeof data !== 'object' || data === null) {
        errors.push('ECharts data must be an object');
      } else if (!isValidEChartsConfig(data)) {
        // 警告但不作为错误，因为可能是简化配置
        logger.debug('[Validate] ECharts config does not have standard fields, but may still be valid');
      }
      break;
      
    case 'image':
      if (typeof data !== 'string') {
        errors.push('Image data must be a string');
      } else if (!data.startsWith('data:image/')) {
        errors.push('Image data must be a valid data URL');
      }
      break;
      
    case 'table':
      if (!data || typeof data !== 'object') {
        errors.push('Table data must be an object');
      } else if (!Array.isArray(data.columns) || !Array.isArray(data.rows)) {
        errors.push('Table data must have columns and rows arrays');
      }
      break;
      
    case 'csv':
      // CSV规范化后与table格式相同
      if (!data || typeof data !== 'object') {
        errors.push('CSV data must be an object');
      } else if (!Array.isArray(data.columns) || !Array.isArray(data.rows)) {
        errors.push('CSV data must have columns and rows arrays');
      }
      break;
      
    case 'metric':
      if (!isValidMetricData(data)) {
        if (!data || typeof data !== 'object') {
          errors.push('Metric data must be an object');
        } else {
          if (!data.title && !data.name) {
            errors.push('Metric must have a title or name');
          }
          if (data.value === undefined) {
            errors.push('Metric must have a value');
          }
        }
      }
      break;
      
    case 'insight':
      if (!data || typeof data !== 'object') {
        errors.push('Insight data must be an object');
      } else if (!data.text) {
        errors.push('Insight must have text');
      }
      break;
      
    case 'file':
      if (!data || typeof data !== 'object') {
        errors.push('File data must be an object');
      } else if (!data.fileName && !data.name) {
        errors.push('File must have a fileName');
      }
      break;
      
    default:
      errors.push(`Unknown data type: ${type}`);
  }
  
  if (errors.length > 0) {
    logger.debug(`[Validate] Validation failed with ${errors.length} error(s): ${errors.join(', ')}`);
  } else {
    logger.debug('[Validate] Validation passed');
  }
  
  return {
    valid: errors.length === 0,
    errors
  };
}

// 导出默认对象
export const DataNormalizer = {
  // 规范化方法
  normalize,
  validate,
  normalizeECharts,
  normalizeImage,
  normalizeTable,
  normalizeCSV,
  normalizeMetric,
  normalizeInsight,
  normalizeFile,
  // 类型检测方法 (Task 5.1)
  isValidEChartsConfig,
  isValidTableData,
  isValidMetricData,
  detectDataType,
};

export default DataNormalizer;
