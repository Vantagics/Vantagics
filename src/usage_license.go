package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// UsageLicense 本地使用权限记录
type UsageLicense struct {
	ListingID     int64  `json:"listing_id"`
	PackName      string `json:"pack_name"`
	PricingModel  string `json:"pricing_model"`  // free, per_use, time_limited, subscription
	RemainingUses int    `json:"remaining_uses"` // per_use 模式
	ExpiresAt     string `json:"expires_at"`     // time_limited/subscription 模式，RFC3339
	BillingCycle  string `json:"billing_cycle"`  // subscription 模式：monthly/yearly
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// usageLicenseFileData is the JSON file structure for persisting licenses.
type usageLicenseFileData struct {
	Licenses map[string]*UsageLicense `json:"licenses"`
}

// UsageLicenseStore 管理本地使用权限的持久化存储
type UsageLicenseStore struct {
	mu       sync.RWMutex
	filePath string
	licenses map[int64]*UsageLicense // listing_id -> license
}

// NewUsageLicenseStore creates a new UsageLicenseStore with the default file path
// (~/.vantagedata/marketplace_licenses.json).
func NewUsageLicenseStore() (*UsageLicenseStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	filePath := filepath.Join(home, ".vantagedata", "marketplace_licenses.json")
	return &UsageLicenseStore{
		filePath: filePath,
		licenses: make(map[int64]*UsageLicense),
	}, nil
}

// Load reads the license store from the JSON file on disk.
// If the file does not exist, the store remains empty (no error).
// If the file is corrupted, a warning is logged and the store is reset to empty.
func (s *UsageLicenseStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist yet — start with empty store
			s.licenses = make(map[int64]*UsageLicense)
			return nil
		}
		return fmt.Errorf("failed to read license file: %w", err)
	}

	var fileData usageLicenseFileData
	if err := json.Unmarshal(data, &fileData); err != nil {
		// Corrupted file — log warning and reset to empty
		fmt.Printf("[UsageLicenseStore] warning: corrupted license file %s, resetting: %v\n", s.filePath, err)
		s.licenses = make(map[int64]*UsageLicense)
		return nil
	}

	s.licenses = make(map[int64]*UsageLicense, len(fileData.Licenses))
	for _, lic := range fileData.Licenses {
		if lic != nil {
			s.licenses[lic.ListingID] = lic
		}
	}
	return nil
}

// Save writes the current license store to the JSON file on disk.
// It creates the parent directory if it does not exist.
func (s *UsageLicenseStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Build file data with string keys for JSON
	fileData := usageLicenseFileData{
		Licenses: make(map[string]*UsageLicense, len(s.licenses)),
	}
	for id, lic := range s.licenses {
		fileData.Licenses[fmt.Sprintf("%d", id)] = lic
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal license data: %w", err)
	}

	if err := os.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write license file: %w", err)
	}
	return nil
}

// GetLicense returns the UsageLicense for the given listing ID, or nil if not found.
func (s *UsageLicenseStore) GetLicense(listingID int64) *UsageLicense {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.licenses[listingID]
}

// SetLicense adds or updates a license in the store.
func (s *UsageLicenseStore) SetLicense(license *UsageLicense) {
	if license == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.licenses[license.ListingID] = license
}

// DeleteLicense removes the license for the given listing ID.
func (s *UsageLicenseStore) DeleteLicense(listingID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.licenses, listingID)
}

// GetAllLicenses returns all licenses in the store.
func (s *UsageLicenseStore) GetAllLicenses() []*UsageLicense {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*UsageLicense, 0, len(s.licenses))
	for _, lic := range s.licenses {
		result = append(result, lic)
	}
	return result
}

// CheckPermission checks whether the license for the given listing ID allows execution.
// If no license is found, it is treated as free/untracked and allowed.
func (s *UsageLicenseStore) CheckPermission(listingID int64) (allowed bool, reason string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	lic, ok := s.licenses[listingID]
	if !ok {
		return true, ""
	}

	switch lic.PricingModel {
	case "free":
		return true, ""
	case "per_use":
		if lic.RemainingUses > 0 {
			return true, ""
		}
		return false, "使用次数已用完，请重新购买"
	case "time_limited", "subscription":
		expiresAt, err := time.Parse(time.RFC3339, lic.ExpiresAt)
		if err != nil {
			// Cannot parse expiry — be permissive
			return true, ""
		}
		if time.Now().Before(expiresAt) {
			return true, ""
		}
		return false, "使用权限已过期，请续费"
	default:
		// Unknown pricing model — be permissive
		return true, ""
	}
}

// ConsumeUse decrements remaining_uses by 1 for a per_use license.
// Returns an error if remaining_uses is already 0.
// For non-per_use licenses or missing licenses, this is a no-op.
func (s *UsageLicenseStore) ConsumeUse(listingID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	lic, ok := s.licenses[listingID]
	if !ok || lic.PricingModel != "per_use" {
		return nil
	}

	if lic.RemainingUses <= 0 {
		return fmt.Errorf("no remaining uses")
	}

	lic.RemainingUses--
	lic.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	return nil
}