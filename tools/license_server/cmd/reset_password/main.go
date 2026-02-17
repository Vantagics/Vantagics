package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Note: The license_server database is NOT encrypted (uses plain modernc.org/sqlite).
// No database password is needed to open it.

func main() {
	fmt.Println("========================================")
	fmt.Println("  授权服务器 - 管理员密码重置工具")
	fmt.Println("========================================")
	fmt.Println()

	// Find database file
	dbPath := findDBPath()
	if dbPath == "" {
		fmt.Println("错误: 未找到数据库文件 license_server.db")
		fmt.Println("请将此工具放在与 license_server 相同的目录下运行")
		fmt.Println("或使用 -db 参数指定数据库路径:")
		fmt.Println("  reset_password -db /path/to/license_server.db")
		os.Exit(1)
	}

	fmt.Printf("数据库文件: %s\n\n", dbPath)

	// Open database (unencrypted SQLite, same driver as main license server)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Printf("错误: 无法打开数据库: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Verify database connection
	var testVal int
	if err := db.QueryRow("SELECT 1").Scan(&testVal); err != nil {
		fmt.Printf("错误: 数据库连接失败: %v\n", err)
		os.Exit(1)
	}

	// Get current admin username
	var username string
	db.QueryRow("SELECT value FROM settings WHERE key='admin_username'").Scan(&username)
	if username == "" {
		username = "admin"
	}
	fmt.Printf("当前管理员用户名: %s\n\n", username)

	// Get new password from args or prompt
	var newPassword string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		if args[i] == "-db" {
			i++ // skip the -db value
			continue
		}
		newPassword = args[i]
		break
	}

	if newPassword == "" {
		fmt.Print("请输入新密码: ")
		fmt.Scanln(&newPassword)
	}

	newPassword = strings.TrimSpace(newPassword)
	if newPassword == "" {
		fmt.Println("错误: 密码不能为空")
		os.Exit(1)
	}

	if len(newPassword) < 4 {
		fmt.Println("错误: 密码长度不能少于4位")
		os.Exit(1)
	}

	// Update password (store as plaintext to match license_server's getSetting/setSetting pattern)
	_, err = db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES ('admin_password', ?)", newPassword)
	if err != nil {
		fmt.Printf("错误: 更新密码失败: %v\n", err)
		os.Exit(1)
	}

	// Also reset all login locks by clearing the in-memory state hint
	fmt.Println("提示: 登录锁定状态存储在服务器内存中，重启授权服务器即可解除所有登录限制。")

	fmt.Println()
	fmt.Println("✅ 密码重置成功！")
	fmt.Printf("   用户名: %s\n", username)
	fmt.Println("   新密码: (已设置)")
	fmt.Println()
	fmt.Println("注意: 如果授权服务器正在运行，新密码将立即生效（无需重启）。")
}

func findDBPath() string {
	// Check -db argument
	for i, arg := range os.Args[1:] {
		if arg == "-db" && i+2 < len(os.Args) {
			path := os.Args[i+2]
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	// Check current directory
	if _, err := os.Stat("license_server.db"); err == nil {
		return "license_server.db"
	}

	// Check executable directory
	execPath, err := os.Executable()
	if err == nil {
		dbPath := filepath.Join(filepath.Dir(execPath), "license_server.db")
		if _, err := os.Stat(dbPath); err == nil {
			return dbPath
		}
	}

	return ""
}
