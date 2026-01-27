package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// LocationTool provides device location information using OS location services
type LocationTool struct {
	logFunc        func(string)
	configLocation *ConfiguredLocation // Fallback location from user settings
}

// ConfiguredLocation represents user-configured location from settings
type ConfiguredLocation struct {
	Country   string
	City      string
	Latitude  float64
	Longitude float64
}

// LocationData stores the current device location (set by frontend via Geolocation API)
type LocationData struct {
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	Accuracy         float64 `json:"accuracy"`          // meters
	Altitude         float64 `json:"altitude,omitempty"` // meters, may be 0 if unavailable
	AltitudeAccuracy float64 `json:"altitude_accuracy,omitempty"`
	Heading          float64 `json:"heading,omitempty"`  // degrees from north
	Speed            float64 `json:"speed,omitempty"`    // meters per second
	Timestamp        int64   `json:"timestamp"`          // Unix timestamp in milliseconds
	City             string  `json:"city,omitempty"`     // Reverse geocoded city name
	Country          string  `json:"country,omitempty"`  // Reverse geocoded country
	Address          string  `json:"address,omitempty"`  // Full address if available
	Available        bool    `json:"available"`          // Whether location is available
	Error            string  `json:"error,omitempty"`    // Error message if location unavailable
}

// Global location storage with mutex for thread safety
var (
	currentLocation LocationData
	locationMutex   sync.RWMutex
)

// UpdateLocation updates the stored location (called from frontend)
func UpdateLocation(data LocationData) {
	locationMutex.Lock()
	defer locationMutex.Unlock()
	currentLocation = data
}

// GetCurrentLocation returns the current stored location
func GetCurrentLocation() LocationData {
	locationMutex.RLock()
	defer locationMutex.RUnlock()
	return currentLocation
}

// LocationToolInput defines the input parameters for the location tool
type LocationToolInput struct {
	// Query type: "coordinates", "city", "full" (default)
	QueryType string `json:"query_type,omitempty" jsonschema:"description=Type of location query: coordinates (经纬度), city (城市), full (完整信息). Default is full."`
}

// LocationToolOutput defines the output of the location tool
type LocationToolOutput struct {
	Available        bool    `json:"available"`
	Latitude         float64 `json:"latitude,omitempty"`
	Longitude        float64 `json:"longitude,omitempty"`
	Accuracy         float64 `json:"accuracy_meters,omitempty"`
	City             string  `json:"city,omitempty"`
	Country          string  `json:"country,omitempty"`
	Address          string  `json:"address,omitempty"`
	Timestamp        int64   `json:"timestamp,omitempty"`
	Error            string  `json:"error,omitempty"`
	HumanReadable    string  `json:"human_readable,omitempty"`
	Source           string  `json:"source,omitempty"` // "device", "config", or "unavailable"
}

// NewLocationTool creates a new location tool instance
func NewLocationTool(logFunc func(string)) *LocationTool {
	return &LocationTool{
		logFunc: logFunc,
	}
}

// NewLocationToolWithConfig creates a new location tool with configured fallback location
func NewLocationToolWithConfig(logFunc func(string), configLoc *ConfiguredLocation) *LocationTool {
	return &LocationTool{
		logFunc:        logFunc,
		configLocation: configLoc,
	}
}

// Info returns the tool information
func (t *LocationTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_device_location",
		Desc: "Get the current device location using OS location services. Returns latitude, longitude, city, and country. Use this when user asks about their location, nearby places, local weather, etc. This provides the user's actual geographic location.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query_type": {
				Type:     schema.String,
				Desc:     "Type of location query: 'coordinates' for lat/lng only, 'city' for city/country, 'full' for all info (default)",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the location tool
func (t *LocationTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	t.log("[LOCATION-TOOL] Invoked with args: %s", argumentsInJSON)

	var input LocationToolInput
	if argumentsInJSON != "" && argumentsInJSON != "{}" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
			t.log("[LOCATION-TOOL] Failed to parse input, using defaults: %v", err)
		}
	}

	if input.QueryType == "" {
		input.QueryType = "full"
	}

	// Get current location from storage (device location)
	loc := GetCurrentLocation()

	output := LocationToolOutput{
		Available: loc.Available,
		Timestamp: loc.Timestamp,
		Source:    "device",
	}

	// If device location is available, use it
	if loc.Available {
		output.Latitude = loc.Latitude
		output.Longitude = loc.Longitude
		output.Accuracy = loc.Accuracy
		output.City = loc.City
		output.Country = loc.Country
		output.Address = loc.Address

		// Build human readable string
		if loc.City != "" && loc.Country != "" {
			output.HumanReadable = fmt.Sprintf("当前位置: %s, %s (精度: %.0f米)", loc.City, loc.Country, loc.Accuracy)
		} else if loc.Address != "" {
			output.HumanReadable = fmt.Sprintf("当前位置: %s (精度: %.0f米)", loc.Address, loc.Accuracy)
		} else {
			output.HumanReadable = fmt.Sprintf("当前位置: 纬度 %.6f, 经度 %.6f (精度: %.0f米)", loc.Latitude, loc.Longitude, loc.Accuracy)
		}
		t.log("[LOCATION-TOOL] Using device location: %s, %s", loc.City, loc.Country)
	} else if t.configLocation != nil && t.configLocation.City != "" {
		// Fallback to configured location from settings
		output.Available = true
		output.Source = "config"
		output.City = t.configLocation.City
		output.Country = t.configLocation.Country
		output.Latitude = t.configLocation.Latitude
		output.Longitude = t.configLocation.Longitude
		output.Accuracy = 0 // Config location has no accuracy
		output.HumanReadable = fmt.Sprintf("用户设置位置: %s, %s", t.configLocation.City, t.configLocation.Country)
		t.log("[LOCATION-TOOL] Using configured location: %s, %s", t.configLocation.City, t.configLocation.Country)
	} else {
		// No location available
		output.Available = false
		output.Source = "unavailable"
		output.Error = loc.Error
		if output.Error == "" {
			output.Error = "Location not available. User may need to grant location permission or configure location in settings."
		}
		// Provide clear guidance for LLM when location is unavailable
		output.HumanReadable = fmt.Sprintf("无法获取位置信息: %s。请直接询问用户所在城市，或使用默认城市（如北京）进行查询。", output.Error)
		t.log("[LOCATION-TOOL] No location available")
	}

	// Filter output based on query type
	var result interface{}
	switch input.QueryType {
	case "coordinates":
		result = map[string]interface{}{
			"available":       output.Available,
			"latitude":        output.Latitude,
			"longitude":       output.Longitude,
			"accuracy_meters": output.Accuracy,
			"source":          output.Source,
			"error":           output.Error,
		}
	case "city":
		result = map[string]interface{}{
			"available":      output.Available,
			"city":           output.City,
			"country":        output.Country,
			"source":         output.Source,
			"human_readable": output.HumanReadable,
			"error":          output.Error,
		}
	default:
		result = output
	}

	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %v", err)
	}

	t.log("[LOCATION-TOOL] Result: %s", string(resultJSON))
	return string(resultJSON), nil
}

// log logs a message using the provided log function
func (t *LocationTool) log(format string, args ...interface{}) {
	if t.logFunc != nil {
		t.logFunc(fmt.Sprintf(format, args...))
	}
}
