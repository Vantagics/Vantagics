/**
 * System Log Utility
 * Writes debug logs to system.log in user cache directory
 */

import { WriteSystemLog } from '../../wailsjs/go/main/App';

export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

// 日志级别优先级
const LOG_LEVEL_PRIORITY: Record<LogLevel, number> = {
    'DEBUG': 0,
    'INFO': 1,
    'WARN': 2,
    'ERROR': 3,
};

// 最小日志级别 - 只有 >= 此级别的日志才会被写入
// 生产环境设为 'WARN'，调试时可改为 'DEBUG' 或 'INFO'
let minLogLevel: LogLevel = 'DEBUG';

/**
 * 设置最小日志级别
 * @param level 最小日志级别
 */
export function setMinLogLevel(level: LogLevel): void {
    minLogLevel = level;
}

/**
 * 获取当前最小日志级别
 */
export function getMinLogLevel(): LogLevel {
    return minLogLevel;
}

/**
 * Write a log entry to system.log
 * @param level Log level (DEBUG, INFO, WARN, ERROR)
 * @param source Source component (e.g., 'App', 'ChatSidebar', 'Dashboard')
 * @param message Log message
 */
export async function systemLog(level: LogLevel, source: string, message: string): Promise<void> {
    // 检查日志级别
    if (LOG_LEVEL_PRIORITY[level] < LOG_LEVEL_PRIORITY[minLogLevel]) {
        return; // 跳过低于最小级别的日志
    }
    
    try {
        await WriteSystemLog(level, source, message);
    } catch (error) {
        // Fallback to console if system log fails
        console.error('[SystemLog] Failed to write to system.log:', error);
        console.log(`[${level}] [${source}] ${message}`);
    }
}

/**
 * Convenience methods for different log levels
 */
export const SystemLog = {
    debug: (source: string, message: string) => systemLog('DEBUG', source, message),
    info: (source: string, message: string) => systemLog('INFO', source, message),
    warn: (source: string, message: string) => systemLog('WARN', source, message),
    error: (source: string, message: string) => systemLog('ERROR', source, message),
};

/**
 * Create a logger for a specific component
 * @param componentName Name of the component
 */
export function createLogger(componentName: string) {
    return {
        debug: (message: string) => SystemLog.debug(componentName, message),
        info: (message: string) => SystemLog.info(componentName, message),
        warn: (message: string) => SystemLog.warn(componentName, message),
        error: (message: string) => SystemLog.error(componentName, message),
    };
}
