// appdata_manager is a command-line tool for managing encrypted store credentials
// Usage:
//   appdata_manager list                    - List all store credentials
//   appdata_manager add <platform>          - Add a new store credential
//   appdata_manager edit <platform>         - Edit an existing store credential
//   appdata_manager delete <platform>       - Delete a store credential
//   appdata_manager export <file>           - Export credentials to JSON (decrypted)
//   appdata_manager import <file>           - Import credentials from JSON

package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	appDataFileName = "appdata.dat"
	encryptionKey   = "vantagedata"
	currentVersion  = "1.0"
)

// StoreCredentials holds OAuth credentials for a store platform
type StoreCredentials struct {
	Platform     string `json:"platform"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	APIKey       string `json:"api_key,omitempty"`
	APISecret    string `json:"api_secret,omitempty"`
	Scopes       string `json:"scopes,omitempty"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	Enabled      bool   `json:"enabled"`
	Description  string `json:"description,omitempty"`
}

// AppData holds all encrypted application data
type AppData struct {
	Version     string             `json:"version"`
	Stores      []StoreCredentials `json:"stores"`
	LastUpdated string             `json:"last_updated"`
}

var key []byte

func init() {
	hash := sha256.Sum256([]byte(encryptionKey))
	key = hash[:]
}

func encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %v", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %v", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %v", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}

	return plaintext, nil
}

func getDataPath() string {
	// 输出到当前目录，方便使用
	return appDataFileName
}

func loadData() (*AppData, error) {
	path := getDataPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &AppData{Version: currentVersion, Stores: []StoreCredentials{}}, nil
	}

	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read data file: %v", err)
	}

	plaintext, err := decrypt(string(encoded))
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %v", err)
	}

	var data AppData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return nil, fmt.Errorf("failed to parse data: %v", err)
	}

	return &data, nil
}

func saveData(data *AppData) error {
	plaintext, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %v", err)
	}

	encoded, err := encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %v", err)
	}

	path := getDataPath()

	if err := os.WriteFile(path, []byte(encoded), 0600); err != nil {
		return fmt.Errorf("failed to write data file: %v", err)
	}

	return nil
}

func readInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func listCredentials() {
	data, err := loadData()
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	if len(data.Stores) == 0 {
		fmt.Println("No store credentials configured.")
		return
	}

	fmt.Printf("\n%-15s %-20s %-10s %s\n", "Platform", "Client ID", "Enabled", "Description")
	fmt.Println(strings.Repeat("-", 70))

	for _, store := range data.Stores {
		clientID := store.ClientID
		if len(clientID) > 18 {
			clientID = clientID[:15] + "..."
		}
		enabled := "No"
		if store.Enabled {
			enabled = "Yes"
		}
		fmt.Printf("%-15s %-20s %-10s %s\n", store.Platform, clientID, enabled, store.Description)
	}
	fmt.Println()
}

func addCredential(platform string) {
	data, err := loadData()
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	// Check if platform already exists
	for _, store := range data.Stores {
		if store.Platform == platform {
			fmt.Printf("Platform '%s' already exists. Use 'edit' command to modify.\n", platform)
			return
		}
	}

	store := StoreCredentials{
		Platform: platform,
		Enabled:  true,
	}

	fmt.Printf("\nAdding credentials for: %s\n", platform)
	fmt.Println(strings.Repeat("-", 40))

	store.ClientID = readInput("Client ID: ")
	store.ClientSecret = readPassword("Client Secret: ")
	store.APIKey = readInput("API Key (optional): ")
	store.APISecret = readPassword("API Secret (optional): ")
	store.Scopes = readInput("Scopes (optional): ")
	store.RedirectURI = readInput("Redirect URI (optional): ")
	store.Description = readInput("Description (optional): ")

	enabledStr := readInput("Enabled (y/n) [y]: ")
	store.Enabled = enabledStr == "" || strings.ToLower(enabledStr) == "y"

	data.Stores = append(data.Stores, store)

	if err := saveData(data); err != nil {
		fmt.Printf("Error saving data: %v\n", err)
		return
	}

	fmt.Printf("\nCredentials for '%s' added successfully.\n", platform)
}

func editCredential(platform string) {
	data, err := loadData()
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	var storeIdx = -1
	for i, store := range data.Stores {
		if store.Platform == platform {
			storeIdx = i
			break
		}
	}

	if storeIdx == -1 {
		fmt.Printf("Platform '%s' not found.\n", platform)
		return
	}

	store := &data.Stores[storeIdx]

	fmt.Printf("\nEditing credentials for: %s\n", platform)
	fmt.Println("Press Enter to keep current value")
	fmt.Println(strings.Repeat("-", 40))

	if input := readInput(fmt.Sprintf("Client ID [%s]: ", store.ClientID)); input != "" {
		store.ClientID = input
	}
	if input := readPassword("Client Secret [****]: "); input != "" {
		store.ClientSecret = input
	}
	if input := readInput(fmt.Sprintf("API Key [%s]: ", store.APIKey)); input != "" {
		store.APIKey = input
	}
	if input := readPassword("API Secret [****]: "); input != "" {
		store.APISecret = input
	}
	if input := readInput(fmt.Sprintf("Scopes [%s]: ", store.Scopes)); input != "" {
		store.Scopes = input
	}
	if input := readInput(fmt.Sprintf("Redirect URI [%s]: ", store.RedirectURI)); input != "" {
		store.RedirectURI = input
	}
	if input := readInput(fmt.Sprintf("Description [%s]: ", store.Description)); input != "" {
		store.Description = input
	}

	enabledDefault := "n"
	if store.Enabled {
		enabledDefault = "y"
	}
	if input := readInput(fmt.Sprintf("Enabled (y/n) [%s]: ", enabledDefault)); input != "" {
		store.Enabled = strings.ToLower(input) == "y"
	}

	if err := saveData(data); err != nil {
		fmt.Printf("Error saving data: %v\n", err)
		return
	}

	fmt.Printf("\nCredentials for '%s' updated successfully.\n", platform)
}

func deleteCredential(platform string) {
	data, err := loadData()
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	var newStores []StoreCredentials
	found := false
	for _, store := range data.Stores {
		if store.Platform == platform {
			found = true
		} else {
			newStores = append(newStores, store)
		}
	}

	if !found {
		fmt.Printf("Platform '%s' not found.\n", platform)
		return
	}

	confirm := readInput(fmt.Sprintf("Are you sure you want to delete '%s'? (y/n): ", platform))
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Cancelled.")
		return
	}

	data.Stores = newStores

	if err := saveData(data); err != nil {
		fmt.Printf("Error saving data: %v\n", err)
		return
	}

	fmt.Printf("Credentials for '%s' deleted successfully.\n", platform)
}

func exportCredentials(filename string) {
	data, err := loadData()
	if err != nil {
		fmt.Printf("Error loading data: %v\n", err)
		return
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling data: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, jsonData, 0600); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Printf("Credentials exported to '%s' (UNENCRYPTED - handle with care!)\n", filename)
}

func importCredentials(filename string) {
	jsonData, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading file: %v\n", err)
		return
	}

	var data AppData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		fmt.Printf("Error parsing JSON: %v\n", err)
		return
	}

	if err := saveData(&data); err != nil {
		fmt.Printf("Error saving data: %v\n", err)
		return
	}

	fmt.Printf("Imported %d store credentials from '%s'\n", len(data.Stores), filename)
}

func printUsage() {
	fmt.Println(`
AppData Manager - Manage encrypted store credentials

Usage:
  appdata_manager list                    List all store credentials
  appdata_manager add <platform>          Add a new store credential
  appdata_manager edit <platform>         Edit an existing store credential
  appdata_manager delete <platform>       Delete a store credential
  appdata_manager export <file>           Export credentials to JSON (decrypted)
  appdata_manager import <file>           Import credentials from JSON

Supported platforms:
  shopify, woocommerce, magento, bigcommerce, squarespace, wix

Examples:
  appdata_manager add shopify
  appdata_manager edit shopify
  appdata_manager delete woocommerce
  appdata_manager export backup.json
  appdata_manager import backup.json
`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	command := os.Args[1]

	switch command {
	case "list":
		listCredentials()
	case "add":
		if len(os.Args) < 3 {
			fmt.Println("Usage: appdata_manager add <platform>")
			return
		}
		addCredential(os.Args[2])
	case "edit":
		if len(os.Args) < 3 {
			fmt.Println("Usage: appdata_manager edit <platform>")
			return
		}
		editCredential(os.Args[2])
	case "delete":
		if len(os.Args) < 3 {
			fmt.Println("Usage: appdata_manager delete <platform>")
			return
		}
		deleteCredential(os.Args[2])
	case "export":
		if len(os.Args) < 3 {
			fmt.Println("Usage: appdata_manager export <file>")
			return
		}
		exportCredentials(os.Args[2])
	case "import":
		if len(os.Args) < 3 {
			fmt.Println("Usage: appdata_manager import <file>")
			return
		}
		importCredentials(os.Args[2])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
	}
}
