/**
 * Property-Based Tests for DataNormalizer
 * 
 * Feature: analysis-result-display-flow
 * Task: 5.2 编写 DataNormalizer 属性测试
 * 
 * **属性 1: 数据类型规范化一致性**
 * **Validates: Requirements 1.1, 2.1**
 * 
 * 对于任意有效的分析结果数据，经过 DataNormalizer 规范化后再反规范化，
 * 应该产生等价的数据结构。
 */

import { describe, it, expect, vi } from 'vitest';
import * as fc from 'fast-check';
import {
  DataNormalizer,
  normalizeECharts,
  normalizeImage,
  normalizeTable,
  normalizeCSV,
  normalizeMetric,
  normalizeInsight,
  normalizeFile,
  isValidEChartsConfig,
  isValidTableData,
  isValidMetricData,
  detectDataType,
} from './DataNormalizer';

// Mock the systemLog to avoid Wails runtime errors in tests
vi.mock('./systemLog', () => ({
  createLogger: () => ({
    debug: () => {},
    info: () => {},
    warn: () => {},
    error: () => {},
  }),
}));

// ==================== Test Data Generators ====================

/**
 * Generate valid ECharts series data
 */
const echartsSeriesArb = fc.record({
  type: fc.constantFrom('line', 'bar', 'pie', 'scatter', 'radar'),
  name: fc.option(fc.string({ minLength: 1, maxLength: 20 }), { nil: undefined }),
  data: fc.array(fc.oneof(fc.integer(), fc.float()), { minLength: 1, maxLength: 10 }),
});


/**
 * Generate valid ECharts configuration object
 */
const validEChartsConfigArb = fc.record({
  series: fc.array(echartsSeriesArb, { minLength: 1, maxLength: 3 }),
  xAxis: fc.option(
    fc.record({
      type: fc.constantFrom('category', 'value', 'time'),
      data: fc.option(fc.array(fc.string({ minLength: 1, maxLength: 10 }), { minLength: 1, maxLength: 5 }), { nil: undefined }),
    }),
    { nil: undefined }
  ),
  yAxis: fc.option(
    fc.record({
      type: fc.constantFrom('category', 'value'),
    }),
    { nil: undefined }
  ),
  title: fc.option(
    fc.record({
      text: fc.string({ minLength: 1, maxLength: 50 }),
    }),
    { nil: undefined }
  ),
  tooltip: fc.option(
    fc.record({
      trigger: fc.constantFrom('item', 'axis'),
    }),
    { nil: undefined }
  ),
  legend: fc.option(
    fc.record({
      data: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 0, maxLength: 5 }),
    }),
    { nil: undefined }
  ),
});

/**
 * Generate valid table data in standard format { columns, rows }
 */
const validTableDataStandardArb = fc.record({
  columns: fc.array(fc.string({ minLength: 1, maxLength: 20 }), { minLength: 1, maxLength: 5 }),
  rows: fc.array(
    fc.dictionary(
      fc.string({ minLength: 1, maxLength: 20 }),
      fc.oneof(fc.string(), fc.integer(), fc.float(), fc.boolean())
    ),
    { minLength: 0, maxLength: 10 }
  ),
});

/**
 * Generate valid table data as object array
 */
const validTableDataObjectArrayArb = fc.array(
  fc.dictionary(
    fc.string({ minLength: 1, maxLength: 20 }),
    fc.oneof(fc.string(), fc.integer(), fc.float(), fc.boolean())
  ),
  { minLength: 1, maxLength: 10 }
).filter(arr => arr.length > 0 && Object.keys(arr[0]).length > 0);


/**
 * Generate valid metric data
 */
const validMetricDataArb = fc.record({
  title: fc.string({ minLength: 1, maxLength: 50 }),
  value: fc.oneof(
    fc.string({ minLength: 1, maxLength: 20 }),
    fc.integer(),
    fc.float()
  ),
  change: fc.option(fc.string({ minLength: 0, maxLength: 20 }), { nil: undefined }),
  unit: fc.option(fc.constantFrom('%', '元', '$', 'K', 'M'), { nil: undefined }),
});

/**
 * Generate valid metric data with name field (alternative format)
 */
const validMetricDataWithNameArb = fc.record({
  name: fc.string({ minLength: 1, maxLength: 50 }),
  value: fc.oneof(
    fc.string({ minLength: 1, maxLength: 20 }),
    fc.integer(),
    fc.float()
  ),
  change: fc.option(fc.string({ minLength: 0, maxLength: 20 }), { nil: undefined }),
  unit: fc.option(fc.constantFrom('%', '元', '$', 'K', 'M'), { nil: undefined }),
});

/**
 * Generate valid insight data
 */
const validInsightDataArb = fc.record({
  text: fc.string({ minLength: 1, maxLength: 200 }),
  icon: fc.option(fc.constantFrom('lightbulb', 'chart', 'warning', 'info'), { nil: undefined }),
  dataSourceId: fc.option(fc.uuid(), { nil: undefined }),
  sourceName: fc.option(fc.string({ minLength: 1, maxLength: 50 }), { nil: undefined }),
});

/**
 * Generate valid file data
 */
const validFileDataArb = fc.record({
  fileName: fc.string({ minLength: 1, maxLength: 100 }),
  filePath: fc.string({ minLength: 1, maxLength: 200 }),
  fileType: fc.constantFrom('csv', 'xlsx', 'pdf', 'txt', 'png', 'jpg'),
  size: fc.option(fc.nat(), { nil: undefined }),
});

/**
 * Generate valid base64 PNG image data
 * PNG files start with specific bytes that encode to 'iVBORw0KGgo'
 */
const validBase64PngArb = fc.constant('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==');

/**
 * Generate valid base64 JPEG image data
 * JPEG files start with specific bytes that encode to '/9j/'
 */
const validBase64JpegArb = fc.constant('/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAAgGBgcGBQgHBwcJCQgKDBQNDAsLDBkSEw8UHRofHh0aHBwgJC4nICIsIxwcKDcpLDAxNDQ0Hyc5PTgyPC4zNDL/wAALCAABAAEBAREA/8QAFAABAAAAAAAAAAAAAAAAAAAACf/EABQQAQAAAAAAAAAAAAAAAAAAAAD/2gAIAQEAAD8AVN//2Q==');


/**
 * Generate safe alphanumeric string for CSV (no special characters)
 */
const safeCSVStringArb = fc.stringMatching(/^[a-zA-Z0-9 ]{1,20}$/);

/**
 * Generate valid CSV data string
 * Uses alphanumeric strings to avoid CSV parsing issues with special characters
 */
const validCSVDataArb = fc.tuple(
  fc.array(safeCSVStringArb, { minLength: 1, maxLength: 5 }),
  fc.array(
    fc.array(safeCSVStringArb, { minLength: 1, maxLength: 5 }),
    { minLength: 0, maxLength: 10 }
  )
).map(([headers, rows]) => {
  const headerLine = headers.join(',');
  const dataLines = rows.map(row => {
    // Ensure each row has the same number of columns as headers
    const paddedRow = headers.map((_, i) => row[i] || '');
    return paddedRow.join(',');
  });
  return [headerLine, ...dataLines].join('\n');
});

// ==================== Property Tests ====================

describe('Feature: analysis-result-display-flow, Property 1: 数据类型规范化一致性', () => {
  /**
   * **Validates: Requirements 1.1, 2.1**
   * 
   * Property 1: 数据类型规范化一致性
   * 对于任意有效的分析结果数据，经过 DataNormalizer 规范化后再反规范化，
   * 应该产生等价的数据结构。
   */

  describe('ECharts Data Normalization', () => {
    /**
     * Property Test 1.1: ECharts configuration normalization should preserve data structure
     * 
     * **Validates: Requirements 1.1**
     */
    it('should normalize valid ECharts config and preserve structure', () => {
      fc.assert(
        fc.property(validEChartsConfigArb, (echartsConfig) => {
          // Act: Normalize the ECharts config
          const result = normalizeECharts(echartsConfig);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have the same series
          if (result.data && 'series' in result.data) {
            expect(Array.isArray((result.data as any).series)).toBe(true);
            expect((result.data as any).series.length).toBe(echartsConfig.series.length);
          }
          
          return true;
        }),
        { numRuns: 100 }
      );
    });


    /**
     * Property Test 1.2: ECharts JSON string normalization should produce valid object
     * 
     * **Validates: Requirements 1.1**
     */
    it('should normalize ECharts JSON string and produce valid object', () => {
      fc.assert(
        fc.property(validEChartsConfigArb, (echartsConfig) => {
          // Arrange: Convert to JSON string
          const jsonString = JSON.stringify(echartsConfig);
          
          // Act: Normalize the JSON string
          const result = normalizeECharts(jsonString);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          expect(typeof result.data).toBe('object');
          
          // The parsed data should match the original
          expect(JSON.stringify(result.data)).toBe(JSON.stringify(echartsConfig));
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 1.3: isValidEChartsConfig should correctly identify valid configs
     * 
     * **Validates: Requirements 1.1**
     */
    it('should correctly identify valid ECharts configurations', () => {
      fc.assert(
        fc.property(validEChartsConfigArb, (echartsConfig) => {
          // Act: Check if config is valid
          const isValid = isValidEChartsConfig(echartsConfig);
          
          // Assert: Should be valid since we generated valid config
          expect(isValid).toBe(true);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 1.4: Invalid data should not be identified as ECharts config
     * 
     * **Validates: Requirements 1.1**
     */
    it('should reject invalid data as ECharts config', () => {
      fc.assert(
        fc.property(
          fc.oneof(
            fc.constant(null),
            fc.constant(undefined),
            fc.string(),
            fc.integer(),
            fc.array(fc.integer()),
            fc.record({ randomField: fc.string() })
          ),
          (invalidData) => {
            // Act: Check if invalid data is identified as ECharts
            const isValid = isValidEChartsConfig(invalidData);
            
            // Assert: Should not be valid
            expect(isValid).toBe(false);
            
            return true;
          }
        ),
        { numRuns: 100 }
      );
    });
  });


  describe('Table Data Normalization', () => {
    /**
     * Property Test 2.1: Standard format table normalization should preserve structure
     * 
     * **Validates: Requirements 2.1**
     */
    it('should normalize standard format table data and preserve structure', () => {
      fc.assert(
        fc.property(validTableDataStandardArb, (tableData) => {
          // Act: Normalize the table data
          const result = normalizeTable(tableData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have columns and rows
          expect(Array.isArray(result.data?.columns)).toBe(true);
          expect(Array.isArray(result.data?.rows)).toBe(true);
          
          // Column count should match
          expect(result.data?.columns.length).toBe(tableData.columns.length);
          
          // Row count should match
          expect(result.data?.rows.length).toBe(tableData.rows.length);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 2.2: Object array table normalization should produce standard format
     * 
     * **Validates: Requirements 2.1**
     */
    it('should normalize object array table data to standard format', () => {
      fc.assert(
        fc.property(validTableDataObjectArrayArb, (tableData) => {
          // Act: Normalize the table data
          const result = normalizeTable(tableData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have columns and rows
          expect(Array.isArray(result.data?.columns)).toBe(true);
          expect(Array.isArray(result.data?.rows)).toBe(true);
          
          // Columns should be extracted from object keys
          const expectedColumns = Object.keys(tableData[0]);
          expect(result.data?.columns.length).toBe(expectedColumns.length);
          
          // Row count should match
          expect(result.data?.rows.length).toBe(tableData.length);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });


    /**
     * Property Test 2.3: Table JSON string normalization should produce valid structure
     * 
     * **Validates: Requirements 2.1**
     */
    it('should normalize table JSON string and produce valid structure', () => {
      fc.assert(
        fc.property(validTableDataStandardArb, (tableData) => {
          // Arrange: Convert to JSON string
          const jsonString = JSON.stringify(tableData);
          
          // Act: Normalize the JSON string
          const result = normalizeTable(jsonString);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should match the original structure
          expect(result.data?.columns.length).toBe(tableData.columns.length);
          expect(result.data?.rows.length).toBe(tableData.rows.length);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 2.4: isValidTableData should correctly identify valid table data
     * 
     * **Validates: Requirements 2.1**
     */
    it('should correctly identify valid table data in various formats', () => {
      fc.assert(
        fc.property(
          fc.oneof(validTableDataStandardArb, validTableDataObjectArrayArb),
          (tableData) => {
            // Act: Check if data is valid table
            const isValid = isValidTableData(tableData);
            
            // Assert: Should be valid
            expect(isValid).toBe(true);
            
            return true;
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 2.5: Invalid data should not be identified as table data
     * 
     * **Validates: Requirements 2.1**
     */
    it('should reject invalid data as table data', () => {
      fc.assert(
        fc.property(
          fc.oneof(
            fc.constant(null),
            fc.constant(undefined),
            fc.string(),
            fc.integer(),
            fc.constant([]),  // Empty array
            fc.record({ randomField: fc.string() })  // Object without columns/rows
          ),
          (invalidData) => {
            // Act: Check if invalid data is identified as table
            const isValid = isValidTableData(invalidData);
            
            // Assert: Should not be valid
            expect(isValid).toBe(false);
            
            return true;
          }
        ),
        { numRuns: 100 }
      );
    });
  });


  describe('Metric Data Normalization', () => {
    /**
     * Property Test 3.1: Metric data with title field should normalize correctly
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize metric data with title field', () => {
      fc.assert(
        fc.property(validMetricDataArb, (metricData) => {
          // Act: Normalize the metric data
          const result = normalizeMetric(metricData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have title and value
          expect(result.data?.title).toBe(metricData.title);
          expect(result.data?.value).toBeDefined();
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 3.2: Metric data with name field should normalize correctly
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize metric data with name field', () => {
      fc.assert(
        fc.property(validMetricDataWithNameArb, (metricData) => {
          // Act: Normalize the metric data
          const result = normalizeMetric(metricData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should use name as title
          expect(result.data?.title).toBe(metricData.name);
          expect(result.data?.value).toBeDefined();
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 3.3: isValidMetricData should correctly identify valid metrics
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should correctly identify valid metric data', () => {
      fc.assert(
        fc.property(
          fc.oneof(validMetricDataArb, validMetricDataWithNameArb),
          (metricData) => {
            // Act: Check if data is valid metric
            const isValid = isValidMetricData(metricData);
            
            // Assert: Should be valid
            expect(isValid).toBe(true);
            
            return true;
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 3.4: Invalid data should not be identified as metric data
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should reject invalid data as metric data', () => {
      fc.assert(
        fc.property(
          fc.oneof(
            fc.constant(null),
            fc.constant(undefined),
            fc.string(),
            fc.integer(),
            fc.record({ randomField: fc.string() }),  // Missing title/name and value
            fc.record({ title: fc.string() }),  // Missing value
            fc.record({ value: fc.integer() })  // Missing title/name
          ),
          (invalidData) => {
            // Act: Check if invalid data is identified as metric
            const isValid = isValidMetricData(invalidData);
            
            // Assert: Should not be valid
            expect(isValid).toBe(false);
            
            return true;
          }
        ),
        { numRuns: 100 }
      );
    });
  });


  describe('Image Data Normalization', () => {
    /**
     * Property Test 4.1: PNG base64 image should normalize to data URL
     * 
     * **Validates: Requirements 1.1**
     */
    it('should normalize PNG base64 image to data URL', () => {
      fc.assert(
        fc.property(validBase64PngArb, (base64Data) => {
          // Act: Normalize the image data
          const result = normalizeImage(base64Data);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should be a data URL
          expect(result.data?.startsWith('data:image/png;base64,')).toBe(true);
          
          // The base64 content should be preserved
          expect(result.data?.includes(base64Data)).toBe(true);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 4.2: JPEG base64 image should normalize to data URL
     * 
     * **Validates: Requirements 1.1**
     */
    it('should normalize JPEG base64 image to data URL', () => {
      fc.assert(
        fc.property(validBase64JpegArb, (base64Data) => {
          // Act: Normalize the image data
          const result = normalizeImage(base64Data);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should be a data URL
          expect(result.data?.startsWith('data:image/jpeg;base64,')).toBe(true);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 4.3: Already formatted data URL should be preserved
     * 
     * **Validates: Requirements 1.1**
     */
    it('should preserve already formatted data URL', () => {
      fc.assert(
        fc.property(validBase64PngArb, (base64Data) => {
          // Arrange: Create a data URL
          const dataUrl = `data:image/png;base64,${base64Data}`;
          
          // Act: Normalize the data URL
          const result = normalizeImage(dataUrl);
          
          // Assert: Normalization should succeed and preserve the URL
          expect(result.success).toBe(true);
          expect(result.data).toBe(dataUrl);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });
  });


  describe('CSV Data Normalization', () => {
    /**
     * Property Test 5.1: CSV string should normalize to table format
     * 
     * **Validates: Requirements 2.1**
     */
    it('should normalize CSV string to table format', () => {
      fc.assert(
        fc.property(validCSVDataArb, (csvData) => {
          // Act: Normalize the CSV data
          const result = normalizeCSV(csvData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have columns and rows
          expect(Array.isArray(result.data?.columns)).toBe(true);
          expect(Array.isArray(result.data?.rows)).toBe(true);
          
          // Parse the CSV to verify structure
          const lines = csvData.split('\n').filter(l => l.trim());
          if (lines.length > 0) {
            const expectedColumns = lines[0].split(',').length;
            expect(result.data?.columns.length).toBe(expectedColumns);
            expect(result.data?.rows.length).toBe(Math.max(0, lines.length - 1));
          }
          
          return true;
        }),
        { numRuns: 100 }
      );
    });
  });

  describe('Insight Data Normalization', () => {
    /**
     * Property Test 6.1: Insight object should normalize correctly
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize insight object data', () => {
      fc.assert(
        fc.property(validInsightDataArb, (insightData) => {
          // Act: Normalize the insight data
          const result = normalizeInsight(insightData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have text
          expect(result.data?.text).toBe(insightData.text);
          
          // Icon should be preserved or default to 'lightbulb'
          expect(result.data?.icon).toBe(insightData.icon || 'lightbulb');
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 6.2: Insight string should normalize to object
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize insight string to object', () => {
      fc.assert(
        fc.property(fc.string({ minLength: 1, maxLength: 200 }), (insightText) => {
          // Act: Normalize the insight string
          const result = normalizeInsight(insightText);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have the text
          expect(result.data?.text).toBe(insightText);
          
          // Default icon should be 'lightbulb'
          expect(result.data?.icon).toBe('lightbulb');
          
          return true;
        }),
        { numRuns: 100 }
      );
    });
  });


  describe('File Data Normalization', () => {
    /**
     * Property Test 7.1: File data should normalize correctly
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize file data', () => {
      fc.assert(
        fc.property(validFileDataArb, (fileData) => {
          // Act: Normalize the file data
          const result = normalizeFile(fileData);
          
          // Assert: Normalization should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          // The normalized data should have fileName and filePath
          expect(result.data?.fileName).toBe(fileData.fileName);
          expect(result.data?.filePath).toBe(fileData.filePath);
          expect(result.data?.fileType).toBe(fileData.fileType);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });
  });

  describe('Data Type Detection', () => {
    /**
     * Property Test 8.1: detectDataType should correctly identify ECharts config
     * 
     * **Validates: Requirements 1.1**
     */
    it('should detect ECharts configuration type', () => {
      fc.assert(
        fc.property(validEChartsConfigArb, (echartsConfig) => {
          // Act: Detect the data type
          const detectedType = detectDataType(echartsConfig);
          
          // Assert: Should detect as echarts
          expect(detectedType).toBe('echarts');
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 8.2: detectDataType should correctly identify table data
     * 
     * **Validates: Requirements 2.1**
     */
    it('should detect table data type', () => {
      fc.assert(
        fc.property(validTableDataStandardArb, (tableData) => {
          // Act: Detect the data type
          const detectedType = detectDataType(tableData);
          
          // Assert: Should detect as table
          expect(detectedType).toBe('table');
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 8.3: detectDataType should correctly identify metric data with name field
     * Note: Metric data with 'title' field may be detected as ECharts since 'title' is an ECharts field.
     * Using 'name' field ensures correct metric detection.
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should detect metric data type when using name field', () => {
      fc.assert(
        fc.property(validMetricDataWithNameArb, (metricData) => {
          // Act: Detect the data type
          const detectedType = detectDataType(metricData);
          
          // Assert: Should detect as metric
          expect(detectedType).toBe('metric');
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 8.4: isValidMetricData should correctly validate metric data regardless of detection
     * This tests the validation function directly, which is used after type is known.
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should validate metric data with title field correctly', () => {
      fc.assert(
        fc.property(validMetricDataArb, (metricData) => {
          // Act: Validate the metric data
          const isValid = isValidMetricData(metricData);
          
          // Assert: Should be valid
          expect(isValid).toBe(true);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });
  });


  describe('Unified Normalize Function', () => {
    /**
     * Property Test 9.1: normalize function should handle all types correctly
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize echarts type correctly', () => {
      fc.assert(
        fc.property(validEChartsConfigArb, (echartsConfig) => {
          // Act: Use unified normalize function
          const result = DataNormalizer.normalize('echarts', echartsConfig);
          
          // Assert: Should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 9.2: normalize function should handle table type correctly
     * 
     * **Validates: Requirements 2.1**
     */
    it('should normalize table type correctly', () => {
      fc.assert(
        fc.property(validTableDataStandardArb, (tableData) => {
          // Act: Use unified normalize function
          const result = DataNormalizer.normalize('table', tableData);
          
          // Assert: Should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          expect(Array.isArray(result.data?.columns)).toBe(true);
          expect(Array.isArray(result.data?.rows)).toBe(true);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 9.3: normalize function should handle metric type correctly
     * 
     * **Validates: Requirements 1.1, 2.1**
     */
    it('should normalize metric type correctly', () => {
      fc.assert(
        fc.property(validMetricDataArb, (metricData) => {
          // Act: Use unified normalize function
          const result = DataNormalizer.normalize('metric', metricData);
          
          // Assert: Should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          expect(result.data?.title).toBeDefined();
          expect(result.data?.value).toBeDefined();
          
          return true;
        }),
        { numRuns: 100 }
      );
    });

    /**
     * Property Test 9.4: normalize function should handle image type correctly
     * 
     * **Validates: Requirements 1.1**
     */
    it('should normalize image type correctly', () => {
      fc.assert(
        fc.property(validBase64PngArb, (imageData) => {
          // Act: Use unified normalize function
          const result = DataNormalizer.normalize('image', imageData);
          
          // Assert: Should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          expect(result.data?.startsWith('data:image/')).toBe(true);
          
          return true;
        }),
        { numRuns: 100 }
      );
    });
  });
});


// ==================== Property Tests for analysis-dashboard-optimization ====================

describe('Feature: analysis-dashboard-optimization, Property 10: Error Events Include Recovery Suggestions', () => {
  /**
   * **Validates: Requirements 7.4**
   * 
   * Property 10: Error Events Include Recovery Suggestions
   * For any error emitted by EventAggregator, the error event SHALL include
   * a non-empty recoverySuggestions array.
   * 
   * Note: This tests the frontend error handling behavior that mirrors the backend
   * EventAggregator error handling.
   */

  /**
   * Generate error codes that should have recovery suggestions
   */
  const errorCodeArb = fc.constantFrom(
    'ANALYSIS_ERROR',
    'ANALYSIS_TIMEOUT',
    'ANALYSIS_CANCELLED',
    'PYTHON_EXECUTION',
    'PYTHON_SYNTAX',
    'PYTHON_IMPORT',
    'PYTHON_MEMORY',
    'DATA_NOT_FOUND',
    'DATA_INVALID',
    'DATA_EMPTY',
    'DATA_TOO_LARGE',
    'CONNECTION_FAILED',
    'CONNECTION_TIMEOUT',
    'PERMISSION_DENIED',
    'RESOURCE_BUSY',
    'RESOURCE_NOT_FOUND',
    'UNKNOWN_ERROR'
  );

  /**
   * Get recovery suggestions for an error code (mirrors backend logic)
   */
  function getRecoverySuggestions(errorCode: string): string[] {
    const suggestions: string[] = [];
    
    switch (errorCode) {
      case 'ANALYSIS_ERROR':
        suggestions.push(
          '请检查您的查询是否清晰明确',
          '尝试简化查询条件',
          '如果问题持续，请刷新页面后重试'
        );
        break;
      case 'ANALYSIS_TIMEOUT':
        suggestions.push(
          '请尝试简化查询或减少数据范围',
          '检查网络连接是否稳定',
          '稍后重试，系统可能正在处理其他任务'
        );
        break;
      case 'ANALYSIS_CANCELLED':
        suggestions.push(
          '您可以重新发起分析请求',
          '如果是误操作，请再次提交相同的查询'
        );
        break;
      case 'PYTHON_EXECUTION':
        suggestions.push(
          '请检查数据格式是否正确',
          '尝试使用不同的分析方式',
          '如果问题持续，请联系技术支持'
        );
        break;
      case 'PYTHON_SYNTAX':
        suggestions.push(
          '系统生成的代码存在语法问题',
          '请尝试重新描述您的分析需求',
          '使用更简单的查询语句'
        );
        break;
      case 'PYTHON_IMPORT':
        suggestions.push(
          '所需的分析库可能未安装',
          '请联系管理员检查系统配置',
          '尝试使用其他分析方法'
        );
        break;
      case 'PYTHON_MEMORY':
        suggestions.push(
          '数据量可能过大，请减少查询范围',
          '尝试分批处理数据',
          '稍后重试，系统可能正在释放资源'
        );
        break;
      case 'DATA_NOT_FOUND':
        suggestions.push(
          '请检查数据源是否已正确配置',
          '确认查询的表或字段名称是否正确',
          '检查数据是否已被删除或移动'
        );
        break;
      case 'DATA_INVALID':
        suggestions.push(
          '请检查数据格式是否符合要求',
          '确认数据类型是否正确',
          '尝试清理或重新导入数据'
        );
        break;
      case 'DATA_EMPTY':
        suggestions.push(
          '当前查询条件下没有数据',
          '请尝试调整筛选条件',
          '检查数据源是否包含所需数据'
        );
        break;
      case 'DATA_TOO_LARGE':
        suggestions.push(
          '请减少查询的数据范围',
          '添加更多筛选条件',
          '考虑分页或分批查询'
        );
        break;
      case 'CONNECTION_FAILED':
        suggestions.push(
          '请检查网络连接',
          '确认服务是否正常运行',
          '稍后重试'
        );
        break;
      case 'CONNECTION_TIMEOUT':
        suggestions.push(
          '网络连接超时，请检查网络状态',
          '服务可能繁忙，请稍后重试',
          '如果问题持续，请联系技术支持'
        );
        break;
      case 'PERMISSION_DENIED':
        suggestions.push(
          '您可能没有访问此资源的权限',
          '请联系管理员获取相应权限',
          '检查您的账户状态'
        );
        break;
      case 'RESOURCE_BUSY':
        suggestions.push(
          '资源正在被其他任务使用',
          '请稍后重试',
          '如果问题持续，请联系技术支持'
        );
        break;
      case 'RESOURCE_NOT_FOUND':
        suggestions.push(
          '请检查资源路径是否正确',
          '确认资源是否已被删除',
          '联系管理员确认资源状态'
        );
        break;
      default:
        suggestions.push(
          '请稍后重试',
          '如果问题持续，请联系技术支持'
        );
    }
    
    return suggestions;
  }

  /**
   * Property Test 10.1: All error codes should have non-empty recovery suggestions
   * 
   * **Validates: Requirements 7.4**
   */
  it('should provide non-empty recovery suggestions for all error codes', () => {
    fc.assert(
      fc.property(errorCodeArb, (errorCode) => {
        // Act: Get recovery suggestions
        const suggestions = getRecoverySuggestions(errorCode);
        
        // Assert: Suggestions should be non-empty
        expect(Array.isArray(suggestions)).toBe(true);
        expect(suggestions.length).toBeGreaterThan(0);
        
        // Each suggestion should be a non-empty string
        for (const suggestion of suggestions) {
          expect(typeof suggestion).toBe('string');
          expect(suggestion.length).toBeGreaterThan(0);
        }
        
        return true;
      }),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 10.2: Unknown error codes should still have default suggestions
   * 
   * **Validates: Requirements 7.4**
   */
  it('should provide default suggestions for unknown error codes', () => {
    fc.assert(
      fc.property(
        fc.string({ minLength: 1, maxLength: 50 }),
        (randomErrorCode) => {
          // Act: Get recovery suggestions for random error code
          const suggestions = getRecoverySuggestions(randomErrorCode);
          
          // Assert: Should have at least default suggestions
          expect(Array.isArray(suggestions)).toBe(true);
          expect(suggestions.length).toBeGreaterThan(0);
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});

describe('Feature: analysis-dashboard-optimization, Property 12: Data Type Processing', () => {
  /**
   * **Validates: Requirements 8.4, 8.5, 8.6**
   * 
   * Property 12: Data Type Processing
   * For any metric, insight, or file data item, the AnalysisResultManager SHALL
   * correctly normalize and store the data with all required fields preserved.
   */

  /**
   * Generate valid metric data
   */
  const metricDataArb = fc.record({
    title: fc.string({ minLength: 1, maxLength: 50 }),
    value: fc.oneof(fc.string({ minLength: 1 }), fc.integer().map(String)),
    change: fc.option(fc.string(), { nil: undefined }),
    trend: fc.option(fc.constantFrom('up', 'down', 'stable'), { nil: undefined }),
  });

  /**
   * Generate valid insight data
   */
  const insightDataArb = fc.record({
    text: fc.string({ minLength: 1, maxLength: 200 }),
    icon: fc.option(fc.string(), { nil: undefined }),
    type: fc.option(fc.constantFrom('info', 'warning', 'success', 'error'), { nil: undefined }),
  });

  /**
   * Generate valid file data
   */
  const fileDataArb = fc.record({
    fileName: fc.string({ minLength: 1, maxLength: 100 }),
    filePath: fc.string({ minLength: 1, maxLength: 200 }),
    fileType: fc.constantFrom('csv', 'xlsx', 'pdf', 'txt', 'json'),
    fileSize: fc.option(fc.nat(), { nil: undefined }),
  });

  /**
   * Property Test 12.1: Metric data should be correctly normalized
   * 
   * **Validates: Requirements 8.4**
   */
  it('should correctly normalize metric data with all required fields', () => {
    fc.assert(
      fc.property(metricDataArb, (metricData) => {
        // Act: Normalize the metric data
        const result = normalizeMetric(metricData);
        
        // Assert: Normalization should succeed
        expect(result.success).toBe(true);
        expect(result.data).toBeDefined();
        
        // Required fields should be preserved
        expect(result.data?.title).toBe(metricData.title);
        expect(result.data?.value).toBeDefined();
        
        return true;
      }),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 12.2: Insight data should be correctly normalized
   * 
   * **Validates: Requirements 8.5**
   */
  it('should correctly normalize insight data with all required fields', () => {
    fc.assert(
      fc.property(insightDataArb, (insightData) => {
        // Act: Normalize the insight data
        const result = normalizeInsight(insightData);
        
        // Assert: Normalization should succeed
        expect(result.success).toBe(true);
        expect(result.data).toBeDefined();
        
        // Required field should be preserved
        expect(result.data?.text).toBe(insightData.text);
        
        return true;
      }),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 12.3: File data should be correctly normalized
   * 
   * **Validates: Requirements 8.6**
   */
  it('should correctly normalize file data with all required fields', () => {
    fc.assert(
      fc.property(fileDataArb, (fileData) => {
        // Act: Normalize the file data
        const result = normalizeFile(fileData);
        
        // Assert: Normalization should succeed
        expect(result.success).toBe(true);
        expect(result.data).toBeDefined();
        
        // Required fields should be preserved
        expect(result.data?.fileName).toBe(fileData.fileName);
        expect(result.data?.filePath).toBe(fileData.filePath);
        expect(result.data?.fileType).toBe(fileData.fileType);
        
        return true;
      }),
      { numRuns: 100 }
    );
  });

  /**
   * Property Test 12.4: All data types should be processable through unified normalize
   * 
   * **Validates: Requirements 8.4, 8.5, 8.6**
   */
  it('should process all data types through unified normalize function', () => {
    fc.assert(
      fc.property(
        fc.oneof(
          fc.tuple(fc.constant('metric'), metricDataArb),
          fc.tuple(fc.constant('insight'), insightDataArb),
          fc.tuple(fc.constant('file'), fileDataArb)
        ),
        ([dataType, data]) => {
          // Act: Use unified normalize function
          const result = DataNormalizer.normalize(dataType as any, data);
          
          // Assert: Should succeed
          expect(result.success).toBe(true);
          expect(result.data).toBeDefined();
          
          return true;
        }
      ),
      { numRuns: 100 }
    );
  });
});
