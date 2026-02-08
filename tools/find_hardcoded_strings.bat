@echo off
REM 查找硬编码字符串的脚本 (Windows版本)
REM 用于识别需要国际化的代码

echo === 查找前端硬编码字符串 ===
echo.

echo 1. 查找 setError 调用...
echo ---
findstr /S /N /R "setError(" src\frontend\src\components\*.tsx 2>nul | findstr /V "t(" | more
echo.

echo 2. 查找 setMessage 调用...
echo ---
findstr /S /N /R "setMessage(" src\frontend\src\components\*.tsx 2>nul | findstr /V "t(" | more
echo.

echo 3. 查找 alert 调用...
echo ---
findstr /S /N /R "alert(" src\frontend\src\components\*.tsx 2>nul | findstr /V "t(" | more
echo.

echo === 查找后端硬编码字符串 ===
echo.

echo 4. 查找 fmt.Errorf 调用...
echo ---
findstr /S /N /R "fmt\.Errorf(" src\*.go 2>nul | findstr /V "i18n.T" | more
echo.

echo 5. 查找 JSON 响应中的硬编码消息...
echo ---
findstr /S /N /R "\"error\":" src\*.go 2>nul | findstr /V "i18n.T" | more
echo.

echo === 完成 ===
echo 请查看上述输出，识别需要迁移的硬编码字符串
pause
