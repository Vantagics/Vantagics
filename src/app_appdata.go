package main

import (
	"fmt"
	"vantagedata/agent"
)

// appDataService holds the singleton instance
var appDataService *agent.AppDataService

// initAppDataService initializes the app data service
func (a *App) initAppDataService() error {
	if appDataService != nil {
		return nil
	}

	appDataService = agent.NewAppDataService(a.Log)
	if err := appDataService.Load(); err != nil {
		return fmt.Errorf("failed to load app data: %v", err)
	}

	return nil
}

// GetShopifyConfigFromAppData returns Shopify config from embedded app data
func (a *App) GetShopifyConfigFromAppData() (*agent.ShopifyOAuthConfig, error) {
	if err := a.initAppDataService(); err != nil {
		return nil, err
	}

	config := appDataService.GetShopifyConfig()
	if config == nil {
		return nil, fmt.Errorf("Shopify not configured or disabled")
	}

	return config, nil
}

// GetStoreConfig returns a specific store configuration by platform
func (a *App) GetStoreConfig(platform string) (*agent.StoreCredentials, error) {
	if err := a.initAppDataService(); err != nil {
		return nil, err
	}

	store := appDataService.GetStore(platform)
	if store == nil {
		return nil, fmt.Errorf("store not found: %s", platform)
	}

	return store, nil
}
