package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
)

// StoreCredentials holds OAuth credentials for a store platform
type StoreCredentials struct {
	Platform     string `json:"platform"`      // shopify, woocommerce, magento, etc.
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

// AppDataService manages encrypted application data (read-only from embedded file)
type AppDataService struct {
	key   []byte
	data  *AppData
	mutex sync.RWMutex
	log   func(string)
}

const (
	// encryptionKey is used to decrypt embedded application data (appdata.dat).
	// This is intentionally a simple key as it only provides obfuscation for
	// embedded OAuth credentials, not true security. The actual security comes
	// from the OAuth flow itself and the fact that credentials are per-installation.
	// For production deployments, consider using environment variables or a key vault.
	encryptionKey  = "vantagedata"
	currentVersion = "1.0"
)

//go:embed appdata.dat
var embeddedAppData embed.FS

// NewAppDataService creates a new AppDataService
func NewAppDataService(log func(string)) *AppDataService {
	// Derive 32-byte key from password using SHA-256
	hash := sha256.Sum256([]byte(encryptionKey))

	return &AppDataService{
		key:  hash[:],
		log:  log,
		data: &AppData{Version: currentVersion, Stores: []StoreCredentials{}},
	}
}

// decrypt decrypts data using AES-GCM
func (s *AppDataService) decrypt(encoded string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %v", err)
	}

	block, err := aes.NewCipher(s.key)
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


// Load loads and decrypts the app data from embedded file
func (s *AppDataService) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Read from embedded file
	encoded, err := embeddedAppData.ReadFile("appdata.dat")
	if err != nil {
		s.log("[APPDATA] No embedded data file, using defaults")
		s.data = &AppData{Version: currentVersion, Stores: []StoreCredentials{}}
		return nil
	}

	// Decrypt
	plaintext, err := s.decrypt(string(encoded))
	if err != nil {
		s.log(fmt.Sprintf("[APPDATA] Failed to decrypt embedded data: %v", err))
		s.data = &AppData{Version: currentVersion, Stores: []StoreCredentials{}}
		return nil
	}

	// Parse JSON
	var data AppData
	if err := json.Unmarshal(plaintext, &data); err != nil {
		return fmt.Errorf("failed to parse data: %v", err)
	}

	s.data = &data
	s.log(fmt.Sprintf("[APPDATA] Loaded %d store configurations from embedded data", len(s.data.Stores)))
	return nil
}

// GetStores returns all store configurations (read-only)
func (s *AppDataService) GetStores() []StoreCredentials {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	result := make([]StoreCredentials, len(s.data.Stores))
	copy(result, s.data.Stores)
	return result
}

// GetStore returns a specific store configuration by platform
func (s *AppDataService) GetStore(platform string) *StoreCredentials {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, store := range s.data.Stores {
		if store.Platform == platform {
			return &store
		}
	}
	return nil
}

// GetShopifyConfig returns Shopify OAuth config if configured
func (s *AppDataService) GetShopifyConfig() *ShopifyOAuthConfig {
	store := s.GetStore("shopify")
	if store == nil || !store.Enabled {
		return nil
	}

	return &ShopifyOAuthConfig{
		ClientID:     store.ClientID,
		ClientSecret: store.ClientSecret,
		Scopes:       store.Scopes,
	}
}

// SupportedPlatforms returns list of supported e-commerce platforms
func SupportedPlatforms() []map[string]string {
	return []map[string]string{
		{"id": "shopify", "name": "Shopify", "description": "Shopify e-commerce platform"},
		{"id": "woocommerce", "name": "WooCommerce", "description": "WordPress WooCommerce plugin"},
		{"id": "magento", "name": "Magento", "description": "Adobe Magento Commerce"},
		{"id": "bigcommerce", "name": "BigCommerce", "description": "BigCommerce platform"},
		{"id": "squarespace", "name": "Squarespace", "description": "Squarespace Commerce"},
		{"id": "wix", "name": "Wix", "description": "Wix eCommerce"},
	}
}
