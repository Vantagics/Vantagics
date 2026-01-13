package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// TrendTemplate implements sales/revenue trend analysis
type TrendTemplate struct{}

func init() {
	Register(&TrendTemplate{})
}

func (t *TrendTemplate) Name() string {
	return "trend"
}

func (t *TrendTemplate) Description() string {
	return "Trend Analysis - Analyze sales, revenue, or order trends over time"
}

func (t *TrendTemplate) Keywords() []string {
	return []string{
		"trend", "趋势", "sales trend", "销售趋势",
		"revenue trend", "营收趋势",
		"monthly sales", "月度销售",
		"yearly growth", "年度增长",
		"time series", "时间序列",
	}
}

func (t *TrendTemplate) RequiredColumns() []string {
	return []string{"date", "amount"}
}

func (t *TrendTemplate) CanExecute(tables []TableInfo) bool {
	for _, table := range tables {
		hasDate := false
		hasAmount := false

		for _, col := range table.Columns {
			lower := strings.ToLower(col)
			if strings.Contains(lower, "date") || strings.Contains(lower, "time") || strings.Contains(lower, "created") {
				hasDate = true
			}
			if strings.Contains(lower, "amount") || strings.Contains(lower, "total") || strings.Contains(lower, "price") || strings.Contains(lower, "revenue") || strings.Contains(lower, "sales") {
				hasAmount = true
			}
		}

		if hasDate && hasAmount {
			return true
		}
	}
	return false
}

func (t *TrendTemplate) Execute(ctx context.Context, executor DataExecutor, dataSourceID string, onProgress ProgressCallback) (*TemplateResult, error) {
	if onProgress != nil {
		onProgress("schema", 10, "Getting database schema...", 1, 5)
	}

	// Step 1: Get schema
	tables, err := executor.GetSchema(ctx, dataSourceID)
	if err != nil {
		return &TemplateResult{Success: false, Error: fmt.Sprintf("Failed to get schema: %v", err)}, nil
	}

	// Step 2: Find best table for trend analysis
	if onProgress != nil {
		onProgress("analysis", 20, "Identifying data tables...", 2, 5)
	}

	var targetTable string
	var dateCol, amountCol string

	for _, table := range tables {
		for _, col := range table.Columns {
			lower := strings.ToLower(col)
			if dateCol == "" && (strings.Contains(lower, "orderdate") || strings.Contains(lower, "order_date") || strings.Contains(lower, "date") || strings.Contains(lower, "created")) {
				dateCol = col
				targetTable = table.Name
			}
			if amountCol == "" && (strings.Contains(lower, "totalamount") || strings.Contains(lower, "total_amount") || strings.Contains(lower, "amount") || strings.Contains(lower, "revenue") || strings.Contains(lower, "sales")) {
				amountCol = col
			}
		}
		if dateCol != "" && amountCol != "" {
			break
		}
	}

	if targetTable == "" || dateCol == "" || amountCol == "" {
		return &TemplateResult{
			Success: false,
			Error:   "Could not identify suitable columns for trend analysis. Need date and amount columns.",
		}, nil
	}

	// Step 3: Execute trend query
	if onProgress != nil {
		onProgress("query", 40, "Executing trend query...", 3, 5)
	}

	trendSQL := fmt.Sprintf(`
		SELECT
			%s as Date,
			%s as Amount
		FROM %s
		WHERE %s IS NOT NULL AND %s IS NOT NULL
		ORDER BY %s
	`, dateCol, amountCol, targetTable, dateCol, amountCol, dateCol)

	data, err := executor.ExecuteSQL(ctx, dataSourceID, trendSQL)
	if err != nil {
		return &TemplateResult{Success: false, Error: fmt.Sprintf("SQL query failed: %v", err)}, nil
	}

	if len(data) == 0 {
		return &TemplateResult{Success: false, Error: "No data returned from query"}, nil
	}

	// Step 4: Execute Python analysis
	if onProgress != nil {
		onProgress("analysis", 60, "Analyzing trends...", 4, 5)
	}

	dataJSON, _ := json.Marshal(data)
	pythonCode := fmt.Sprintf(`
import json
import pandas as pd
import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
# Configure Chinese font support
plt.rcParams['font.sans-serif'] = ['SimHei', 'Microsoft YaHei', 'DejaVu Sans', 'Arial Unicode MS']
plt.rcParams['axes.unicode_minus'] = False  # Fix minus sign display
from datetime import datetime

# Load data
data = json.loads('''%s''')
df = pd.DataFrame(data)

# Convert date column
df['Date'] = pd.to_datetime(df['Date'])
df['Amount'] = pd.to_numeric(df['Amount'], errors='coerce')

# Aggregate by month
df['YearMonth'] = df['Date'].dt.to_period('M')
monthly = df.groupby('YearMonth').agg({
    'Amount': ['sum', 'count', 'mean']
}).reset_index()
monthly.columns = ['YearMonth', 'TotalAmount', 'OrderCount', 'AvgAmount']
monthly['YearMonth'] = monthly['YearMonth'].astype(str)

# Calculate growth rates
monthly['Growth'] = monthly['TotalAmount'].pct_change() * 100

# Print summary
print("=" * 50)
print("TREND ANALYSIS RESULTS")
print("=" * 50)
print(f"\nData Range: {df['Date'].min().strftime('%%Y-%%m-%%d')} to {df['Date'].max().strftime('%%Y-%%m-%%d')}")
print(f"Total Records: {len(df):,}")
print(f"Total Amount: ${df['Amount'].sum():,.2f}")
print(f"Average per Transaction: ${df['Amount'].mean():,.2f}")

print("\nMonthly Summary (Last 12 months):")
recent = monthly.tail(12)
print(recent.to_string(index=False))

# Calculate YoY growth if we have enough data
if len(monthly) >= 12:
    current_year = monthly['TotalAmount'].tail(12).sum()
    previous_year = monthly['TotalAmount'].iloc[-24:-12].sum() if len(monthly) >= 24 else monthly['TotalAmount'].head(12).sum()
    yoy_growth = ((current_year - previous_year) / previous_year * 100) if previous_year > 0 else 0
    print(f"\nYear-over-Year Growth: {yoy_growth:.1f}%%")

# Create visualization
fig, axes = plt.subplots(2, 2, figsize=(14, 10))

# 1. Monthly revenue trend
ax1 = axes[0, 0]
ax1.plot(range(len(monthly)), monthly['TotalAmount'], 'b-o', linewidth=2, markersize=4)
ax1.set_title('Monthly Revenue Trend')
ax1.set_ylabel('Revenue')
ax1.set_xlabel('Month')
step = max(1, len(monthly) // 10)
ax1.set_xticks(range(0, len(monthly), step))
ax1.set_xticklabels(monthly['YearMonth'].iloc[::step], rotation=45)
ax1.grid(True, alpha=0.3)

# 2. Order count trend
ax2 = axes[0, 1]
ax2.bar(range(len(monthly)), monthly['OrderCount'], color='green', alpha=0.7)
ax2.set_title('Monthly Order Count')
ax2.set_ylabel('Orders')
ax2.set_xlabel('Month')
ax2.set_xticks(range(0, len(monthly), step))
ax2.set_xticklabels(monthly['YearMonth'].iloc[::step], rotation=45)
ax2.grid(True, alpha=0.3)

# 3. Growth rate
ax3 = axes[1, 0]
colors = ['green' if x >= 0 else 'red' for x in monthly['Growth'].fillna(0)]
ax3.bar(range(len(monthly)), monthly['Growth'].fillna(0), color=colors, alpha=0.7)
ax3.axhline(y=0, color='black', linestyle='-', linewidth=0.5)
ax3.set_title('Month-over-Month Growth Rate')
ax3.set_ylabel('Growth (%%)')
ax3.set_xlabel('Month')
ax3.set_xticks(range(0, len(monthly), step))
ax3.set_xticklabels(monthly['YearMonth'].iloc[::step], rotation=45)
ax3.grid(True, alpha=0.3)

# 4. Average order value
ax4 = axes[1, 1]
ax4.plot(range(len(monthly)), monthly['AvgAmount'], 'purple', linewidth=2, marker='s', markersize=4)
ax4.set_title('Average Order Value')
ax4.set_ylabel('Avg Amount')
ax4.set_xlabel('Month')
ax4.set_xticks(range(0, len(monthly), step))
ax4.set_xticklabels(monthly['YearMonth'].iloc[::step], rotation=45)
ax4.grid(True, alpha=0.3)

plt.tight_layout()
plt.savefig('chart.png', dpi=150, bbox_inches='tight')
print("\nVisualization saved to chart.png")

# Save CSV
monthly.to_csv('trend_results.csv', index=False)
print("Detailed results saved to trend_results.csv")
`, string(dataJSON))

	output, err := executor.ExecutePython(ctx, pythonCode, "")
	if err != nil {
		return &TemplateResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("Python execution failed: %v", err),
		}, nil
	}

	if onProgress != nil {
		onProgress("complete", 100, "Trend analysis complete", 5, 5)
	}

	return &TemplateResult{
		Success: true,
		Output:  output,
	}, nil
}
