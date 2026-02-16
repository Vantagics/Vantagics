package main

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"crypto/sha256"
	"encoding/hex"
)

//go:embed vantagedata.exe
var exeBytes []byte

//go:embed duckdb.dll
var dllBytes []byte

func main() {
	// 1. 创建唯一的临时目录（基于内容的哈希，避免重复释放）
	h := sha256.New()
	h.Write(exeBytes)
	versionHash := hex.EncodeToString(h.Sum(nil))[:8]
	
	tempDir := filepath.Join(os.TempDir(), "VantageData_Runtime_"+versionHash)
	_ = os.MkdirAll(tempDir, 0755)

	exePath := filepath.Join(tempDir, "vantagedata.exe")
	dllPath := filepath.Join(tempDir, "duckdb.dll")

	// 2. 释放文件
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		os.WriteFile(exePath, exeBytes, 0755)
		os.WriteFile(dllPath, dllBytes, 0644)
	}

	// 3. 运行主程序
	cmd := exec.Command(exePath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	// 设置工作目录，确保主程序能找到自己的配置
	// 如果希望配置保存在 EXE 旁边，可以改用其他路径
	
	err := cmd.Run()
	if err != nil {
		fmt.Printf("程序运行结束: %v
", err)
		os.Exit(1)
	}
}
