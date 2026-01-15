/**
 * System Log Utility
 * Writes debug logs to system.log in user cache directory
 */

import { WriteSystemLog } from '../../wailsjs/go/main/App';

export type LogLevel = 'DEBUG' | 'INFO' | 'WARN' | 'ERROR';

/**
 * Write a log entry to system.log
 * @param level Log level (DEBUG, INFO, WARN, ERROR)
 * @param source Source component (e.g., 'App', 'ChatSidebar', 'Dashboard')
 * @param message Log message
 */
export async function systemLog(level: LogLevel, source: string, message: string): Promise<void> {
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
