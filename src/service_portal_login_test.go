package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"vantagedata/agent"
	"vantagedata/config"
	"vantagedata/logger"
)

// TestServicePortalLogin_LicenseNotActivated_NilClient tests that ServicePortalLogin
// returns an error when licenseClient is nil.
// Validates: Requirements 3.5
func TestServicePortalLogin_LicenseNotActivated_NilClient(t *testing.T) {
	app := &App{
		licenseClient: nil,
		logger:        logger.NewLogger(),
	}

	_, err := app.ServicePortalLogin()
	if err == nil {
		t.Fatal("expected error when licenseClient is nil, got nil")
	}
	if err.Error() != "license not activated" {
		t.Errorf("expected error 'license not activated', got %q", err.Error())
	}
}

// TestServicePortalLogin_LicenseNotActivated_NoData tests that ServicePortalLogin
// returns an error when licenseClient exists but has no activation data.
// Validates: Requirements 3.5
func TestServicePortalLogin_LicenseNotActivated_NoData(t *testing.T) {
	lc := agent.NewLicenseClient(func(s string) {})
	// LicenseClient with no activation data → IsActivated() returns false

	app := &App{
		licenseClient: lc,
		logger:        logger.NewLogger(),
	}

	_, err := app.ServicePortalLogin()
	if err == nil {
		t.Fatal("expected error when license is not activated, got nil")
	}
	if err.Error() != "license not activated" {
		t.Errorf("expected error 'license not activated', got %q", err.Error())
	}
}

// TestServicePortalLogin_SNEmpty tests that ServicePortalLogin returns an error
// when the license is activated but SN is empty.
// Validates: Requirements 3.6
func TestServicePortalLogin_SNEmpty(t *testing.T) {
	lc := agent.NewLicenseClient(func(s string) {})
	// Activate with data but leave SN empty
	lc.SetActivationForTest(&agent.ActivationData{
		LLMType: "test",
	}, "")

	app := &App{
		licenseClient: lc,
		logger:        logger.NewLogger(),
	}

	_, err := app.ServicePortalLogin()
	if err == nil {
		t.Fatal("expected error when SN is empty, got nil")
	}
	if err.Error() != "SN not available" {
		t.Errorf("expected error 'SN not available', got %q", err.Error())
	}
}

// TestServicePortalLogin_EmailEmpty tests that ServicePortalLogin returns an error
// when the license is activated and SN is set, but email is empty in config.
// Validates: Requirements 3.7
func TestServicePortalLogin_EmailEmpty(t *testing.T) {
	lc := agent.NewLicenseClient(func(s string) {})
	lc.SetActivationForTest(&agent.ActivationData{
		LLMType: "test",
	}, "TEST-SN-12345")

	// Create a temp directory with a config file that has empty LicenseEmail
	tmpDir := t.TempDir()
	cfg := config.Config{
		LicenseEmail: "",
	}
	cfgData, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "config.json"), cfgData, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	app := &App{
		licenseClient: lc,
		storageDir:    tmpDir,
		logger:        logger.NewLogger(),
	}

	_, err = app.ServicePortalLogin()
	if err == nil {
		t.Fatal("expected error when email is empty, got nil")
	}
	if err.Error() != "email not available" {
		t.Errorf("expected error 'email not available', got %q", err.Error())
	}
}
