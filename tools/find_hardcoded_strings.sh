#!/bin/bash

# 查找硬编码字符串的脚本
# 用于识别需要国际化的代码

echo "=== 查找前端硬编码字符串 ==="
echo ""

echo "1. 查找中文字符串..."
echo "---"
grep -rn "[\u4e00-\u9fa5]" src/frontend/src/components --include="*.tsx" --include="*.ts" | \
    grep -v "i18n.ts" | \
    grep -v "// " | \
    head -20
echo ""

echo "2. 查找 setError 调用..."
echo "---"
grep -rn "setError\(" src/frontend/src/components --include="*.tsx" | \
    grep -v "t(" | \
    head -10
echo ""

echo "3. 查找 setMessage 调用..."
echo "---"
grep -rn "setMessage\(" src/frontend/src/components --include="*.tsx" | \
    grep -v "t(" | \
    head -10
echo ""

echo "4. 查找 alert 调用..."
echo "---"
grep -rn "alert\(" src/frontend/src/components --include="*.tsx" | \
    grep -v "t(" | \
    head -10
echo ""

echo "=== 查找后端硬编码字符串 ==="
echo ""

echo "5. 查找 Go 代码中的中文字符串..."
echo "---"
grep -rn "[\u4e00-\u9fa5]" src --include="*.go" | \
    grep -v "i18n/" | \
    grep -v "// " | \
    head -20
echo ""

echo "6. 查找 fmt.Errorf 调用..."
echo "---"
grep -rn "fmt\.Errorf\(" src --include="*.go" | \
    grep -v "i18n.T" | \
    head -10
echo ""

echo "7. 查找 JSON 响应中的硬编码消息..."
echo "---"
grep -rn "\"error\":" src --include="*.go" | \
    grep -v "i18n.T" | \
    head -10
echo ""

echo "=== 统计信息 ==="
echo ""
echo "前端组件文件总数: $(find src/frontend/src/components -name "*.tsx" -o -name "*.ts" | wc -l)"
echo "后端 Go 文件总数: $(find src -name "*.go" | wc -l)"
echo ""
echo "可能需要国际化的前端文件:"
grep -rl "[\u4e00-\u9fa5]" src/frontend/src/components --include="*.tsx" --include="*.ts" | \
    grep -v "i18n.ts" | wc -l
echo ""
echo "可能需要国际化的后端文件:"
grep -rl "[\u4e00-\u9fa5]" src --include="*.go" | \
    grep -v "i18n/" | wc -l
echo ""

echo "=== 完成 ==="
echo "请查看上述输出，识别需要迁移的硬编码字符串"
