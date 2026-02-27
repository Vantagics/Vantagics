/**
 * Input Validation Utilities
 * 
 * 前端输入验证工具函数
 */

export interface ValidationResult {
  valid: boolean;
  error?: string;
}

/**
 * 验证邮箱格式
 */
export function validateEmail(email: string): ValidationResult {
  const trimmed = email.trim();
  if (!trimmed) {
    return { valid: false, error: 'Email is required' };
  }
  
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!emailRegex.test(trimmed)) {
    return { valid: false, error: 'Invalid email format' };
  }
  
  return { valid: true };
}

/**
 * 验证字符串长度
 */
export function validateLength(
  value: string,
  minLength: number,
  maxLength: number,
  fieldName: string = 'Field'
): ValidationResult {
  const length = value.length;
  
  if (length < minLength) {
    return {
      valid: false,
      error: `${fieldName} must be at least ${minLength} characters`,
    };
  }
  
  if (maxLength > 0 && length > maxLength) {
    return {
      valid: false,
      error: `${fieldName} must not exceed ${maxLength} characters`,
    };
  }
  
  return { valid: true };
}

/**
 * 验证必填字段
 */
export function validateRequired(value: string, fieldName: string = 'Field'): ValidationResult {
  if (!value || value.trim() === '') {
    return { valid: false, error: `${fieldName} is required` };
  }
  return { valid: true };
}

/**
 * 验证URL格式
 */
export function validateURL(url: string): ValidationResult {
  const trimmed = url.trim();
  if (!trimmed) {
    return { valid: false, error: 'URL is required' };
  }
  
  if (!trimmed.startsWith('http://') && !trimmed.startsWith('https://')) {
    return { valid: false, error: 'URL must start with http:// or https://' };
  }
  
  try {
    new URL(trimmed);
    return { valid: true };
  } catch {
    return { valid: false, error: 'Invalid URL format' };
  }
}

/**
 * 验证数字范围
 */
export function validateRange(
  value: number,
  min: number,
  max: number,
  fieldName: string = 'Value'
): ValidationResult {
  if (value < min || value > max) {
    return {
      valid: false,
      error: `${fieldName} must be between ${min} and ${max}`,
    };
  }
  return { valid: true };
}

/**
 * 验证枚举值
 */
export function validateEnum(
  value: string,
  allowed: string[],
  fieldName: string = 'Value'
): ValidationResult {
  if (!allowed.includes(value)) {
    return {
      valid: false,
      error: `${fieldName} must be one of: ${allowed.join(', ')}`,
    };
  }
  return { valid: true };
}

/**
 * 验证文件大小
 */
export function validateFileSize(
  size: number,
  maxSize: number,
  fieldName: string = 'File'
): ValidationResult {
  if (size <= 0) {
    return { valid: false, error: `${fieldName} is empty` };
  }
  
  if (size > maxSize) {
    const maxMB = (maxSize / (1024 * 1024)).toFixed(2);
    return {
      valid: false,
      error: `${fieldName} size exceeds maximum of ${maxMB}MB`,
    };
  }
  
  return { valid: true };
}

/**
 * 验证文件扩展名
 */
export function validateFileExtension(
  filename: string,
  allowedExts: string[]
): ValidationResult {
  if (!filename) {
    return { valid: false, error: 'Filename is required' };
  }
  
  const parts = filename.split('.');
  if (parts.length < 2) {
    return { valid: false, error: 'File must have an extension' };
  }
  
  const ext = parts[parts.length - 1].toLowerCase();
  const normalizedAllowed = allowedExts.map(e => e.toLowerCase());
  
  if (!normalizedAllowed.includes(ext)) {
    return {
      valid: false,
      error: `File extension must be one of: ${allowedExts.join(', ')}`,
    };
  }
  
  return { valid: true };
}

/**
 * 验证密码强度
 */
export function validatePassword(password: string): ValidationResult {
  if (password.length < 8) {
    return {
      valid: false,
      error: 'Password must be at least 8 characters',
    };
  }
  
  const hasUpper = /[A-Z]/.test(password);
  const hasLower = /[a-z]/.test(password);
  const hasDigit = /[0-9]/.test(password);
  
  if (!hasUpper || !hasLower || !hasDigit) {
    return {
      valid: false,
      error: 'Password must contain uppercase, lowercase, and digit',
    };
  }
  
  return { valid: true };
}

/**
 * 清理HTML输入（防止XSS）
 */
export function sanitizeHTML(input: string): string {
  // 移除script标签
  let sanitized = input.replace(/<script[^>]*>.*?<\/script>/gi, '');
  
  // 移除事件处理器
  sanitized = sanitized.replace(/\s*on\w+\s*=\s*["'][^"']*["']/gi, '');
  
  // 移除javascript:协议
  sanitized = sanitized.replace(/javascript:/gi, '');
  
  return sanitized;
}

/**
 * 验证JSON格式
 */
export function validateJSON(value: string): ValidationResult {
  if (!value || value.trim() === '') {
    return { valid: true }; // Empty is valid
  }
  
  try {
    JSON.parse(value);
    return { valid: true };
  } catch {
    return { valid: false, error: 'Invalid JSON format' };
  }
}

/**
 * 组合多个验证器
 */
export function combineValidators(
  ...validators: (() => ValidationResult)[]
): ValidationResult {
  for (const validator of validators) {
    const result = validator();
    if (!result.valid) {
      return result;
    }
  }
  return { valid: true };
}
