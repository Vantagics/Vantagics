# 构建优化指南

## 快速参考

### 最快的开发构建
```cmd
build_parallel.bat --skip-frontend
```
仅当 Go 代码改变时使用，跳过前端重新构建。

### 标准快速构建
```cmd
build_parallel.bat
```
并行编译主程序和工具，不生成 NSIS 安装包。

### 完整构建（含安装包）
```cmd
build_parallel.bat --with-nsis
```
生成完整的 NSIS 安装程序。

### 传统构建方式
```cmd
build.bat build --fast
```
使用原有脚本的快速模式。

## 构建脚本对比

### build.bat（优化版）
- 移除了 `-clean` 标志，利用增量编译
- 工具并行编译（2 个进程）
- 支持 `--fast`、`--skip-frontend`、`--skip-tools`、`--skip-nsis` 标志
- 适合需要精细控制的场景

### build_parallel.bat（极速版）
- 主程序和工具完全并行（3 个进程）
- 默认跳过 NSIS 安装包生成
- 最大化利用多核 CPU
- 适合日常开发的快速迭代

## 性能提升

### 首次构建
- 原始: ~3-5 分钟
- 优化后: ~2-3 分钟（并行编译）

### 增量构建（仅改 Go 代码）
- 原始: ~2-3 分钟（因为 -clean）
- 优化后: ~30-60 秒（利用缓存 + 并行）

### 增量构建（仅改前端）
- 使用 `--skip-frontend` 无效，需要正常构建
- 但仍比原始快 40-50%

## Go 编译器多线程说明

Go 编译器默认已经使用多核：
- 包级别并行编译（不同包同时编译）
- 函数级别并行编译（同一包内的函数）
- 链接阶段也是并行的

我们的优化主要在：
1. 移除 `-clean` 利用编译缓存
2. 主程序和工具并行构建
3. 减少不必要的步骤（NSIS、前端重建）

## 环境变量优化

可以在系统环境变量中设置：

```cmd
REM 设置 Go 构建缓存位置（可选，默认已优化）
set GOCACHE=C:\Users\YourName\AppData\Local\go-build

REM 设置 Go 模块缓存（可选）
set GOMODCACHE=C:\Users\YourName\go\pkg\mod
```

## 进一步优化建议

### 1. 使用 SSD
确保项目在 SSD 上，而不是机械硬盘。

### 2. 排除杀毒软件扫描
将以下目录添加到杀毒软件排除列表：
- 项目目录
- `%GOPATH%`
- `%GOCACHE%`
- `%TEMP%`

### 3. 使用 RAM Disk（高级）
将 `%TEMP%` 和 Go 缓存放在 RAM Disk 可进一步提速 10-20%。

### 4. 前端构建优化
在 `src/frontend/vite.config.ts` 中可以调整：
```typescript
export default defineConfig({
  build: {
    minify: 'esbuild', // 比 terser 快
    sourcemap: false,  // 开发时可以禁用
  }
})
```

## 故障排除

### 构建失败
如果增量构建失败，运行完整清理：
```cmd
build.bat clean
build.bat
```

### 缓存问题
清理 Go 缓存：
```cmd
go clean -cache -modcache -testcache
```

### 并行构建冲突
如果并行构建出现问题，回退到串行：
```cmd
build.bat build --fast
```

## 最佳实践

1. **日常开发**: 使用 `build_parallel.bat`
2. **仅改 Go**: 使用 `build_parallel.bat --skip-frontend`
3. **发布版本**: 使用 `build.bat clean` 然后 `build_parallel.bat --with-nsis`
4. **CI/CD**: 使用 `build.bat` 确保可重现性

## 性能监控

查看编译时间：
```cmd
REM PowerShell
Measure-Command { .\build_parallel.bat }

REM CMD
echo %time% & build_parallel.bat & echo %time%
```

查看 Go 编译详情：
```cmd
set VERBOSE=1
build.bat
```
