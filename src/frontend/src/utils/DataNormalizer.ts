/**
 * Data Normalizer
 * 
 * 数据规范化器，负责将不同格式的数据转换为统一格式
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

/**
 * 规范化 ECharts 数据
 * 输入: JSON字符串或对象
 * 输出: 解析后的ECharts配置对象
 */
export function normalizeECharts(data: string | object): NormalizedResult<object> {
  try {
    let parsed: object;
    
    if (typeof data === 'string') {
      // 尝试解析JSON字符串
      parsed = JSON.parse(data);
    } else if (typeof data === 'object' && data !== null) {
      parsed = data;
    } else {
      return { success: false, error: 'Invalid ECharts data: expected string or object' };
    }
    
    // 基本验证：检查是否有必要的ECharts属性
    if (!parsed || typeof parsed !== 'object') {
      return { success: false, error: 'Invalid ECharts data: parsed result is not an object' };
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
  try {
    if (typeof data !== 'string' || !data) {
      return { success: false, error: 'Invalid image data: expected non-empty string' };
    }
    
    // 如果已经是完整的data URL格式
    if (data.startsWith('data:image/')) {
      return { success: true, data };
    }
    
    // 检测图片类型并添加前缀
    // 尝试从base64数据检测图片类型
    let mimeType = 'image/png'; // 默认PNG
    
    // PNG: 以 iVBORw0KGgo 开头
    if (data.startsWith('iVBORw0KGgo')) {
      mimeType = 'image/png';
    }
    // JPEG: 以 /9j/ 开头
    else if (data.startsWith('/9j/')) {
      mimeType = 'image/jpeg';
    }
    // GIF: 以 R0lGOD 开头
    else if (data.startsWith('R0lGOD')) {
      mimeType = 'image/gif';
    }
    // WebP: 以 UklGR 开头
    else if (data.startsWith('UklGR')) {
      mimeType = 'image/webp';
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
 */
export function normalizeTable(data: any): NormalizedResult<NormalizedTableData> {
  try {
    let parsed: any;
    
    // 解析字符串
    if (typeof data === 'string') {
      // 清理可能的formatter函数（ECharts残留）
      let cleanedData = data
        .replace(/,?\s*"?formatter"?\s*:\s*function\s*\([^)]*\)\s*\{[^}]*\}/g, '')
        .replace(/,(\s*[}\]])/g, '$1');
      parsed = JSON.parse(cleanedData);
    } else {
      parsed = data;
    }
    
    // 处理数组格式
    if (Array.isArray(parsed) && parsed.length > 0) {
      const firstRow = parsed[0];
      
      // 如果第一行是对象，提取列名
      if (typeof firstRow === 'object' && firstRow !== null && !Array.isArray(firstRow)) {
        return {
          success: true,
          data: {
            columns: Object.keys(firstRow),
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
        return { success: true, data: { columns, rows } };
      }
    }
    
    // 处理已经是 { columns, rows } 格式的数据
    if (parsed && typeof parsed === 'object' && 'columns' in parsed && 'rows' in parsed) {
      return { success: true, data: parsed as NormalizedTableData };
    }
    
    // 空数据
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
  try {
    let csvContent = data;
    
    // 如果是data URL格式，提取内容
    if (data.startsWith('data:')) {
      const base64Match = data.match(/base64,(.+)/);
      if (base64Match) {
        csvContent = atob(base64Match[1]);
      }
    }
    
    // 解析CSV
    const lines = csvContent.split(/\r?\n/).filter(line => line.trim());
    if (lines.length === 0) {
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
 */
export function normalizeMetric(data: any): NormalizedResult<NormalizedMetricData> {
  try {
    if (!data || typeof data !== 'object') {
      return { success: false, error: 'Invalid metric data: expected object' };
    }
    
    const title = data.name || data.title || '';
    let value = data.value;
    
    // 处理带单位的值
    if (data.unit && typeof value !== 'string') {
      value = `${value}${data.unit}`;
    } else {
      value = String(value ?? '');
    }
    
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
  try {
    if (typeof data === 'string') {
      return {
        success: true,
        data: {
          text: data,
          icon: 'lightbulb'
        }
      };
    }
    
    if (data && typeof data === 'object') {
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
  try {
    if (!data || typeof data !== 'object') {
      return { success: false, error: 'Invalid file data: expected object' };
    }
    
    return {
      success: true,
      data: {
        fileName: data.name || data.fileName || '',
        filePath: data.path || data.filePath || '',
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
 */
export function validate(type: AnalysisResultType, data: any): ValidationResult {
  const errors: string[] = [];
  
  switch (type) {
    case 'echarts':
      if (typeof data !== 'object' || data === null) {
        errors.push('ECharts data must be an object');
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
  
  return {
    valid: errors.length === 0,
    errors
  };
}

// 导出默认对象
export const DataNormalizer = {
  normalize,
  validate,
  normalizeECharts,
  normalizeImage,
  normalizeTable,
  normalizeCSV,
  normalizeMetric,
  normalizeInsight,
  normalizeFile,
};

export default DataNormalizer;
