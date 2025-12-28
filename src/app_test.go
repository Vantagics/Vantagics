package main

import (
	"encoding/json"
	"testing"
)

func TestDashboardDataSerialization(t *testing.T) {
	// This test expects DashboardData, Metric, and Insight to be defined in app.go
	// Since we haven't defined them yet, this should fail to compile or run.
	
	data := DashboardData{
		Metrics: []Metric{
			{Title: "Total Sales", Value: "$12,345", Change: "+15%"},
			{Title: "Active Users", Value: "1,234", Change: "+5%"},
		},
		Insights: []Insight{
			{Text: "Sales increased by 15% this week!", Icon: "trending-up"},
			{Text: "User engagement is at an all-time high.", Icon: "star"},
		},
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("Failed to marshal DashboardData: %v", err)
	}

	var decoded DashboardData
	err = json.Unmarshal(jsonData, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal DashboardData: %v", err)
	}

	if len(decoded.Metrics) != 2 {
		t.Errorf("Expected 2 metrics, got %d", len(decoded.Metrics))
	}

	if decoded.Metrics[0].Title != "Total Sales" {
		t.Errorf("Expected 'Total Sales', got '%s'", decoded.Metrics[0].Title)
	}
}

func TestGetDashboardData(t *testing.T) {
	app := NewApp()
	data := app.GetDashboardData()

	if len(data.Metrics) == 0 {
		t.Error("Expected metrics to be populated")
	}

	if len(data.Insights) == 0 {
		t.Error("Expected insights to be populated")
	}
}
