package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"

	"vantagedata/i18n"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// ---------------------------------------------------------------------------
// Pack loading & validation
// ---------------------------------------------------------------------------

// LoadQuickAnalysisPackByPath loads a .qap file from a given path (no file picker),
// checks encryption, and validates against the target data source.
func (a *App) LoadQuickAnalysisPackByPath(filePath string, dataSourceID string) (*PackLoadResult, error) {
	a.Log(fmt.Sprintf("%s Loading pack by path: %s for datasource: %s", logTagImport, filePath, dataSourceID))

	encrypted, err := IsEncrypted(filePath)
	if err != nil {
		return nil, fmt.Errorf("%s", i18n.T("qap.invalid_file_format", err))
	}

	if encrypted {
		if result := a.tryAutoDecrypt(filePath, dataSourceID); result != nil {
			return result, nil
		}
		return &PackLoadResult{
			IsEncrypted:   true,
			NeedsPassword: true,
			FilePath:      filePath,
		}, nil
	}

	return a.loadAndValidatePack(filePath, dataSourceID, "")
}

// LoadQuickAnalysisPack opens a file dialog for the user to select a .qap file,
// checks if it is encrypted, and if not, parses and validates it against the target data source.
func (a *App) LoadQuickAnalysisPack(dataSourceID string) (*PackLoadResult, error) {
	a.Log(fmt.Sprintf("%s Starting quick analysis pack import", logTagImport))

	filePath, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: i18n.T("qap.load_pack_dialog_title"),
		Filters: []runtime.FileFilter{
			{DisplayName: "Quick Analysis Pack (*.qap)", Pattern: "*.qap"},
		},
	})
	if err != nil || filePath == "" {
		a.Log(fmt.Sprintf("%s User cancelled file selection", logTagImport))
		return nil, nil
	}

	a.Log(fmt.Sprintf("%s Selected file: %s", logTagImport, filePath))

	encrypted, err := IsEncrypted(filePath)
	if err != nil {
		a.Log(fmt.Sprintf("%s Error checking encryption: %v", logTagImport, err))
		return nil, fmt.Errorf("%s", i18n.T("qap.invalid_file_format", err))
	}

	if encrypted {
		if result := a.tryAutoDecrypt(filePath, dataSourceID); result != nil {
			return result, nil
		}
		a.Log(fmt.Sprintf("%s File is encrypted, requesting password", logTagImport))
		return &PackLoadResult{
			IsEncrypted:   true,
			NeedsPassword: true,
			FilePath:      filePath,
		}, nil
	}

	return a.loadAndValidatePack(filePath, dataSourceID, "")
}

// LoadQuickAnalysisPackWithPassword loads an encrypted .qap file using the provided password.
func (a *App) LoadQuickAnalysisPackWithPassword(filePath string, dataSourceID string, password string) (*PackLoadResult, error) {
	a.Log(fmt.Sprintf("%s Loading encrypted pack with password: %s", logTagImport, filePath))
	return a.loadAndValidatePack(filePath, dataSourceID, password)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// tryAutoDecrypt attempts to decrypt an encrypted pack using stored/persisted passwords.
// Returns the PackLoadResult on success, or nil if auto-decrypt is not possible.
func (a *App) tryAutoDecrypt(filePath, dataSourceID string) *PackLoadResult {
	// Check in-memory password cache
	if storedPwd, ok := a.packPasswords[filePath]; ok && storedPwd != "" {
		a.Log(fmt.Sprintf("%s Found stored password for encrypted pack, attempting auto-decrypt", logTagImport))
		result, err := a.loadAndValidatePack(filePath, dataSourceID, storedPwd)
		if err == nil {
			return result
		}
		a.Log(fmt.Sprintf("%s Auto-decrypt with stored password failed: %v", logTagImport, err))
	}

	// Check persistent store (survives app restarts)
	if a.packPasswordStore != nil {
		if persistedPwd, ok := a.packPasswordStore.GetPassword(filePath); ok && persistedPwd != "" {
			a.Log(fmt.Sprintf("%s Found persisted password for encrypted pack, attempting auto-decrypt", logTagImport))
			a.packPasswords[filePath] = persistedPwd
			result, err := a.loadAndValidatePack(filePath, dataSourceID, persistedPwd)
			if err == nil {
				return result
			}
			a.Log(fmt.Sprintf("%s Auto-decrypt with persisted password failed: %v", logTagImport, err))
		}
	}

	return nil
}

// loadAndValidatePack is the shared logic for loading, parsing, and validating a .qap file.
func (a *App) loadAndValidatePack(filePath string, dataSourceID string, password string) (*PackLoadResult, error) {
	// Resolve password from stores if not provided
	password = a.resolvePackPassword(filePath, password)

	// 1. Unpack from ZIP
	jsonData, err := UnpackFromZip(filePath, password)
	if err != nil {
		a.Log(fmt.Sprintf("%s Error unpacking: %v", logTagImport, err))
		if err == ErrWrongPassword {
			return nil, fmt.Errorf("%s", i18n.T("qap.wrong_password"))
		}
		return nil, fmt.Errorf("%s", i18n.T("qap.invalid_file_format", err))
	}

	// 2. Parse JSON into QuickAnalysisPack
	var pack QuickAnalysisPack
	if err := json.Unmarshal(jsonData, &pack); err != nil {
		a.Log(fmt.Sprintf("%s Error parsing JSON: %v", logTagImport, err))
		return nil, fmt.Errorf("%s", i18n.T("qap.invalid_file_format", err))
	}

	// 3. Validate file type and format version
	if pack.FileType != qapFileType {
		a.Log(fmt.Sprintf("%s Invalid file type", logTagImport))
		return nil, fmt.Errorf("%s", i18n.T("qap.invalid_pack_file"))
	}
	if pack.FormatVersion != "" && pack.FormatVersion != qapFormatVersion {
		a.Log(fmt.Sprintf("%s Unsupported format version: %s", logTagImport, pack.FormatVersion))
		return nil, fmt.Errorf("%s", i18n.T("qap.unsupported_version", pack.FormatVersion))
	}

	// 4. Resolve listing ID from filename or license store
	a.resolveListingID(&pack, filePath)

	a.Log(fmt.Sprintf("%s Parsed pack: %s by %s, %d steps, listing_id=%d",
		logTagImport, pack.Metadata.SourceName, pack.Metadata.Author, len(pack.ExecutableSteps), pack.Metadata.ListingID))

	if len(pack.ExecutableSteps) == 0 {
		a.Log(fmt.Sprintf("%s Pack has no executable steps", logTagImport))
		return nil, fmt.Errorf("%s", i18n.T("qap.no_executable_steps"))
	}

	// 5. Collect target data source schema and validate
	targetSchema, err := a.collectTargetSchemaForPack(dataSourceID, pack.SchemaRequirements)
	if err != nil {
		a.Log(fmt.Sprintf("%s Error collecting target schema: %v", logTagImport, err))
		return nil, fmt.Errorf("%s", i18n.T("qap.schema_fetch_failed", err))
	}

	validation := ValidateSchema(pack.SchemaRequirements, targetSchema)
	a.Log(fmt.Sprintf("%s Schema validation: compatible=%v, missing_tables=%d, missing_columns=%d",
		logTagImport, validation.Compatible, len(validation.MissingTables), len(validation.MissingColumns)))

	if !validation.Compatible {
		a.Log(fmt.Sprintf("%s Schema incompatible: missing tables: %v", logTagImport, validation.MissingTables))
	}

	return &PackLoadResult{
		Pack:             &pack,
		Validation:       validation,
		IsEncrypted:      password != "",
		FilePath:         filePath,
		HasPythonSteps:   a.packHasPythonSteps(&pack),
		PythonConfigured: a.isPythonConfigured(),
	}, nil
}

// resolvePackPassword returns the password to use for unpacking, checking stored passwords
// if the provided password is empty.
func (a *App) resolvePackPassword(filePath, password string) string {
	if password != "" {
		return password
	}
	if storedPwd, ok := a.packPasswords[filePath]; ok && storedPwd != "" {
		a.Log(fmt.Sprintf("%s Using stored marketplace password for auto-decrypt", logTagImport))
		return storedPwd
	}
	if a.packPasswordStore != nil {
		if persistedPwd, ok := a.packPasswordStore.GetPassword(filePath); ok && persistedPwd != "" {
			a.Log(fmt.Sprintf("%s Using persisted marketplace password for auto-decrypt", logTagImport))
			a.packPasswords[filePath] = persistedPwd
			return persistedPwd
		}
	}
	return password
}

// resolveListingID attempts to populate the pack's ListingID from the filename or license store.
func (a *App) resolveListingID(pack *QuickAnalysisPack, filePath string) {
	if pack.Metadata.ListingID == 0 {
		if extractedID := extractListingIDFromFilePath(filePath); extractedID > 0 {
			pack.Metadata.ListingID = extractedID
			a.Log(fmt.Sprintf("%s Extracted listing_id=%d from filename", logTagImport, extractedID))
		}
	}
	if pack.Metadata.ListingID == 0 && a.usageLicenseStore != nil {
		for _, lic := range a.usageLicenseStore.GetAllLicenses() {
			if lic.PackName == pack.Metadata.PackName || lic.PackName == pack.Metadata.SourceName {
				pack.Metadata.ListingID = lic.ListingID
				a.Log(fmt.Sprintf("%s Resolved listing_id=%d from license store by pack name", logTagImport, lic.ListingID))
				break
			}
		}
	}
}

// extractListingIDFromFilePath extracts the marketplace listing ID from a file path.
// Marketplace downloads are saved as "marketplace_pack_{listingID}.qap".
var marketplacePackFileRe = regexp.MustCompile(`^marketplace_pack_(\d+)\.qap$`)

func extractListingIDFromFilePath(filePath string) int64 {
	base := filepath.Base(filePath)
	m := marketplacePackFileRe.FindStringSubmatch(base)
	if len(m) < 2 {
		return 0
	}
	id, err := strconv.ParseInt(m[1], 10, 64)
	if err != nil {
		return 0
	}
	return id
}
