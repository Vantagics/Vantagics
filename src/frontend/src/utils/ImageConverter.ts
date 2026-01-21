/**
 * Image Converter - Image Format Detection and Conversion
 * 
 * Handles detection and conversion of various image formats:
 * - Base64 strings (with or without data URL prefix)
 * - HTTP/HTTPS URLs
 * - File paths (file://, files/, relative paths)
 * - Unknown formats with graceful error handling
 */

/**
 * Enum for detected image format types
 */
export enum ImageFormatType {
  BASE64_DATA_URL = 'base64_data_url',      // data:image/png;base64,iVBORw0KGgo...
  BASE64_STRING = 'base64_string',          // iVBORw0KGgo... (without prefix)
  HTTP_URL = 'http_url',                    // http://example.com/image.png
  HTTPS_URL = 'https_url',                  // https://example.com/image.png
  FILE_PATH = 'file_path',                  // file:// or files/ or relative paths
  UNKNOWN = 'unknown',                      // Unknown format
}

/**
 * Result of image format detection
 */
export interface ImageFormatDetectionResult {
  format: ImageFormatType;
  isValid: boolean;
  error?: string;
  mimeType?: string;
  extractedData?: string;
}

/**
 * Regex patterns for image format detection
 */
const IMAGE_FORMAT_PATTERNS = {
  // Data URL format: data:image/[type];base64,[data]
  dataUrl: /^data:image\/([a-z]+);base64,(.+)$/i,
  
  // Base64 string pattern (RFC 4648)
  // Matches strings that are valid base64 (alphanumeric, +, /, =)
  base64: /^[A-Za-z0-9+/]*={0,2}$/,
  
  // HTTP/HTTPS URLs
  httpUrl: /^https?:\/\/.+/i,
  
  // File paths - matches file://, files/, ./, ../, or filenames with extensions
  fileUrl: /^file:\/\/.+/i,
  filesPrefix: /^files\/.+/i,
  relativePath: /^(\.\/|\.\.\/)/,
  filename: /^[a-zA-Z0-9_\-./\\]+\.(?:png|jpg|jpeg|gif|webp|svg|bmp)$/i,
};

/**
 * Detect image format from a string
 * 
 * Validates Requirements 4.1, 4.2, 4.3, 4.4
 * 
 * @param data - The image data string to detect
 * @returns ImageFormatDetectionResult with detected format and validation status
 */
export function detectImageFormat(data: string): ImageFormatDetectionResult {
  // Validate input
  if (!data || typeof data !== 'string') {
    return {
      format: ImageFormatType.UNKNOWN,
      isValid: false,
      error: 'Image data must be a non-empty string',
    };
  }

  const trimmedData = data.trim();

  // Check for data URL format (data:image/[type];base64,[data])
  const dataUrlMatch = trimmedData.match(IMAGE_FORMAT_PATTERNS.dataUrl);
  if (dataUrlMatch) {
    const mimeType = `image/${dataUrlMatch[1]}`;
    const base64Data = dataUrlMatch[2];
    
    // Validate base64 data
    if (isValidBase64(base64Data)) {
      return {
        format: ImageFormatType.BASE64_DATA_URL,
        isValid: true,
        mimeType,
        extractedData: base64Data,
      };
    } else {
      return {
        format: ImageFormatType.BASE64_DATA_URL,
        isValid: false,
        error: 'Invalid base64 data in data URL',
        mimeType,
      };
    }
  }

  // Check for HTTP/HTTPS URLs
  if (IMAGE_FORMAT_PATTERNS.httpUrl.test(trimmedData)) {
    const lowerData = trimmedData.toLowerCase();
    return {
      format: lowerData.startsWith('https') ? ImageFormatType.HTTPS_URL : ImageFormatType.HTTP_URL,
      isValid: true,
    };
  }

  // Check for file paths (file://, files/, relative paths, or filenames)
  if (IMAGE_FORMAT_PATTERNS.fileUrl.test(trimmedData) ||
      IMAGE_FORMAT_PATTERNS.filesPrefix.test(trimmedData) ||
      IMAGE_FORMAT_PATTERNS.relativePath.test(trimmedData) ||
      IMAGE_FORMAT_PATTERNS.filename.test(trimmedData)) {
    return {
      format: ImageFormatType.FILE_PATH,
      isValid: true,
      extractedData: trimmedData,
    };
  }

  // Check if it's a base64 string without data URL prefix
  // Base64 strings should be at least 20 characters and match base64 pattern
  if (trimmedData.length >= 20 && IMAGE_FORMAT_PATTERNS.base64.test(trimmedData)) {
    if (isValidBase64(trimmedData)) {
      return {
        format: ImageFormatType.BASE64_STRING,
        isValid: true,
        mimeType: 'image/png', // Default MIME type for base64 strings
        extractedData: trimmedData,
      };
    }
  }

  // Unknown format
  return {
    format: ImageFormatType.UNKNOWN,
    isValid: false,
    error: `Unknown image format. Expected base64, data URL, HTTP URL, or file path. Received: ${trimmedData.substring(0, 50)}${trimmedData.length > 50 ? '...' : ''}`,
  };
}

/**
 * Validate if a string is valid base64
 * 
 * @param str - String to validate
 * @returns true if valid base64, false otherwise
 */
export function isValidBase64(str: string): boolean {
  if (!str || typeof str !== 'string') {
    return false;
  }

  // Check if string matches base64 pattern
  if (!IMAGE_FORMAT_PATTERNS.base64.test(str)) {
    return false;
  }

  // Try to decode to verify it's valid base64
  try {
    // Check if the string length is valid for base64
    // Base64 strings should have length that is a multiple of 4 (with padding)
    const paddedLength = str.length + (4 - (str.length % 4)) % 4;
    if (paddedLength % 4 !== 0) {
      return false;
    }

    // Attempt to decode
    atob(str);
    return true;
  } catch (e) {
    return false;
  }
}

/**
 * Extract filename from file path
 * 
 * Handles various file path formats:
 * - file:///path/to/file.png
 * - files/filename.png
 * - ./relative/path.png
 * - ../relative/path.png
 * 
 * @param filePath - The file path to extract filename from
 * @returns Extracted filename or null if invalid
 */
export function extractFilenameFromPath(filePath: string): string | null {
  if (!filePath || typeof filePath !== 'string') {
    return null;
  }

  // Reject paths with .. for security (directory traversal)
  if (filePath.includes('..')) {
    return null;
  }

  // Remove file:// prefix if present
  let path = filePath.replace(/^file:\/\//, '');

  // Get the last part of the path (filename)
  const parts = path.split(/[\/\\]/);
  const filename = parts[parts.length - 1];

  // Validate filename
  if (filename && filename.length > 0) {
    return filename;
  }

  return null;
}

/**
 * Get MIME type from file extension
 * 
 * @param filename - The filename to extract MIME type from
 * @returns MIME type string or 'image/png' as default
 */
export function getMimeTypeFromFilename(filename: string): string {
  if (!filename || typeof filename !== 'string') {
    return 'image/png';
  }

  const extension = filename.split('.').pop()?.toLowerCase();

  const mimeTypes: Record<string, string> = {
    'png': 'image/png',
    'jpg': 'image/jpeg',
    'jpeg': 'image/jpeg',
    'gif': 'image/gif',
    'webp': 'image/webp',
    'svg': 'image/svg+xml',
    'bmp': 'image/bmp',
  };

  return mimeTypes[extension || ''] || 'image/png';
}

/**
 * Validate if a string is a valid HTTP/HTTPS URL
 * 
 * @param url - The URL to validate
 * @returns true if valid HTTP/HTTPS URL, false otherwise
 */
export function isValidHttpUrl(url: string): boolean {
  if (!url || typeof url !== 'string') {
    return false;
  }

  try {
    const urlObj = new URL(url);
    return urlObj.protocol === 'http:' || urlObj.protocol === 'https:';
  } catch (e) {
    return false;
  }
}

/**
 * Validate if a string is a valid file path
 * 
 * @param path - The path to validate
 * @returns true if valid file path, false otherwise
 */
export function isValidFilePath(path: string): boolean {
  if (!path || typeof path !== 'string') {
    return false;
  }

  // Check for file:// prefix
  if (path.startsWith('file://')) {
    return true;
  }

  // Check for files/ prefix
  if (path.startsWith('files/')) {
    return true;
  }

  // Check for relative paths
  if (path.startsWith('./') || path.startsWith('../')) {
    return true;
  }

  // Check for simple filenames with extensions
  if (/^[a-zA-Z0-9_\-]+\.(png|jpg|jpeg|gif|webp|svg|bmp)$/i.test(path)) {
    return true;
  }

  return false;
}

/**
 * Convert base64 string to data URL
 * 
 * Handles:
 * - Base64 strings without data URL prefix
 * - Automatic MIME type detection from base64 data or default to image/png
 * - Edge cases: empty strings, invalid base64
 * 
 * Validates Requirements 4.1
 * 
 * @param base64Data - The base64 string to convert (with or without data URL prefix)
 * @param mimeType - Optional MIME type override. If not provided, will be detected or default to image/png
 * @returns Data URL string (data:image/[type];base64,[data]) or null if invalid
 * @throws Error if base64 data is invalid
 */
export function base64ToDataUrl(base64Data: string, mimeType?: string): string {
  // Validate input
  if (!base64Data || typeof base64Data !== 'string') {
    throw new Error('base64Data must be a non-empty string');
  }

  const trimmedData = base64Data.trim();

  // If already a data URL, return as-is
  if (trimmedData.startsWith('data:')) {
    return trimmedData;
  }

  // Validate base64 string
  if (!isValidBase64(trimmedData)) {
    throw new Error(`Invalid base64 string: ${trimmedData.substring(0, 50)}${trimmedData.length > 50 ? '...' : ''}`);
  }

  // Determine MIME type
  let finalMimeType = mimeType || 'image/png';

  // If MIME type not provided, try to detect from base64 data
  if (!mimeType) {
    finalMimeType = detectMimeTypeFromBase64(trimmedData) || 'image/png';
  }

  // Return data URL
  return `data:${finalMimeType};base64,${trimmedData}`;
}

/**
 * Detect MIME type from base64 image data
 * 
 * Analyzes the base64 data to detect common image format signatures:
 * - PNG: starts with iVBORw0KGgo
 * - JPEG: starts with /9j/
 * - GIF: starts with R0lGODlh
 * - WebP: contains WEBP signature
 * - BMP: starts with Qk
 * 
 * @param base64Data - The base64 string to analyze
 * @returns Detected MIME type or null if unable to detect
 */
export function detectMimeTypeFromBase64(base64Data: string): string | null {
  if (!base64Data || typeof base64Data !== 'string') {
    return null;
  }

  const trimmedData = base64Data.trim();

  // PNG signature: iVBORw0KGgo
  if (trimmedData.startsWith('iVBORw0KGgo')) {
    return 'image/png';
  }

  // JPEG signature: /9j/
  if (trimmedData.startsWith('/9j/')) {
    return 'image/jpeg';
  }

  // GIF signature: R0lGODlh
  if (trimmedData.startsWith('R0lGODlh')) {
    return 'image/gif';
  }

  // WebP signature: UklGRi (RIFF header for WebP)
  if (trimmedData.startsWith('UklGRi')) {
    return 'image/webp';
  }

  // BMP signature: Qk (BM header)
  if (trimmedData.startsWith('Qk')) {
    return 'image/bmp';
  }

  // SVG signature: PHN2ZyAo (<?xml or <svg)
  if (trimmedData.startsWith('PHN2ZyAo') || trimmedData.startsWith('PD94bWw')) {
    return 'image/svg+xml';
  }

  // Unable to detect
  return null;
}

/**
 * Log image format detection for debugging
 * 
 * @param data - The image data that was detected
 * @param result - The detection result
 */
export function logImageDetection(data: string, result: ImageFormatDetectionResult): void {
  const preview = data.substring(0, 50) + (data.length > 50 ? '...' : '');
  
  if (result.isValid) {
    console.log(`[CHART] Detected inline ${result.format} image: ${preview}`);
  } else {
    console.warn(`[CHART] Failed to detect image format: ${result.error}. Data: ${preview}`);
  }
}

/**
 * Cache for converted file paths to avoid repeated API calls
 * Maps: "threadId:filename" -> base64 data URL
 */
const filePathCache = new Map<string, string>();

/**
 * Clear the file path cache
 * Useful for testing or when session changes
 */
export function clearFilePathCache(): void {
  filePathCache.clear();
}

/**
 * Get cache size for debugging
 */
export function getFilePathCacheSize(): number {
  return filePathCache.size;
}

/**
 * Convert file path to base64 data URL
 * 
 * Handles:
 * - File paths (file://, files/, relative paths)
 * - Extracts filename from file paths
 * - Calls GetSessionFileAsBase64 API with threadId and filename
 * - Caches results to avoid repeated API calls
 * - Handles API errors and timeouts gracefully
 * 
 * Validates Requirements 4.2, 2.5
 * 
 * @param filePath - The file path to convert (file://, files/, relative, or filename)
 * @param threadId - The session/thread ID for API calls
 * @param getSessionFileAsBase64 - The API function to call (injected for testability)
 * @param timeoutMs - Timeout in milliseconds for API calls (default: 30000)
 * @returns Promise<string> - Data URL or error state string
 * @throws Error if file path is invalid or API call fails
 */
export async function filePathToBase64(
  filePath: string,
  threadId: string,
  getSessionFileAsBase64: (threadId: string, filename: string) => Promise<string>,
  timeoutMs: number = 30000
): Promise<string> {
  // Validate inputs
  if (!filePath || typeof filePath !== 'string') {
    throw new Error('File path must be a non-empty string');
  }

  if (!threadId || typeof threadId !== 'string') {
    throw new Error('Thread ID must be a non-empty string');
  }

  if (typeof getSessionFileAsBase64 !== 'function') {
    throw new Error('getSessionFileAsBase64 must be a function');
  }

  // Validate file path format
  if (!isValidFilePath(filePath)) {
    throw new Error(`Invalid file path format: ${filePath}`);
  }

  // Extract filename from file path
  const filename = extractFilenameFromPath(filePath);
  if (!filename) {
    throw new Error(`Failed to extract filename from path: ${filePath}`);
  }

  // Check cache first
  const cacheKey = `${threadId}:${filename}`;
  if (filePathCache.has(cacheKey)) {
    console.log(`[CHART] Using cached base64 for file: ${filename}`);
    return filePathCache.get(cacheKey)!;
  }

  try {
    // Call API with timeout
    const base64Data = await Promise.race([
      getSessionFileAsBase64(threadId, filename),
      new Promise<string>((_, reject) =>
        setTimeout(() => reject(new Error(`API call timeout after ${timeoutMs}ms`)), timeoutMs)
      ),
    ]);

    // Validate base64 response
    if (!base64Data || typeof base64Data !== 'string') {
      throw new Error('API returned invalid base64 data');
    }

    // If it's already a data URL, use it directly
    if (base64Data.startsWith('data:')) {
      filePathCache.set(cacheKey, base64Data);
      console.log(`[CHART] Loaded file as base64 data URL: ${filename}`);
      return base64Data;
    }

    // Convert base64 string to data URL
    const mimeType = getMimeTypeFromFilename(filename);
    const dataUrl = `data:${mimeType};base64,${base64Data}`;

    // Cache the result
    filePathCache.set(cacheKey, dataUrl);
    console.log(`[CHART] Loaded file as base64: ${filename}`);

    return dataUrl;
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error(`[CHART] Failed to load file ${filename}: ${errorMessage}`);
    throw new Error(`Failed to load file ${filename}: ${errorMessage}`);
  }
}

/**
 * Convert image data to displayable format
 * 
 * Handles all image format conversions:
 * - Base64 strings → data URLs
 * - File paths → base64 via API
 * - HTTP URLs → pass-through
 * - Unknown formats → error
 * 
 * Validates Requirements 4.1, 4.2, 4.3, 4.4, 4.5
 * 
 * @param imageData - The image data to convert
 * @param threadId - The session/thread ID (required for file paths)
 * @param getSessionFileAsBase64 - The API function to call (injected for testability)
 * @returns Promise<string> - Displayable image URL or error message
 */
export async function convertImageData(
  imageData: string,
  threadId?: string,
  getSessionFileAsBase64?: (threadId: string, filename: string) => Promise<string>
): Promise<string> {
  // Validate input
  if (!imageData || typeof imageData !== 'string') {
    throw new Error('Image data must be a non-empty string');
  }

  // Detect format
  const detection = detectImageFormat(imageData);

  if (!detection.isValid) {
    throw new Error(detection.error || 'Unknown image format');
  }

  try {
    switch (detection.format) {
      case ImageFormatType.BASE64_DATA_URL:
        // Already a data URL, return as-is
        return imageData;

      case ImageFormatType.BASE64_STRING:
        // Convert base64 string to data URL
        return base64ToDataUrl(imageData, detection.mimeType);

      case ImageFormatType.HTTP_URL:
      case ImageFormatType.HTTPS_URL:
        // HTTP URLs pass-through directly
        return imageData;

      case ImageFormatType.FILE_PATH:
        // File path - need to load via API
        if (!threadId) {
          throw new Error('Thread ID required for file path conversion');
        }

        if (!getSessionFileAsBase64) {
          throw new Error('getSessionFileAsBase64 function required for file path conversion');
        }

        return await filePathToBase64(imageData, threadId, getSessionFileAsBase64);

      case ImageFormatType.UNKNOWN:
      default:
        throw new Error(`Unknown image format: ${imageData.substring(0, 50)}`);
    }
  } catch (error) {
    const errorMessage = error instanceof Error ? error.message : String(error);
    console.error(`[CHART] Image conversion failed: ${errorMessage}`);
    throw new Error(`Image conversion failed: ${errorMessage}`);
  }
}
