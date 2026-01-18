package templates

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// RFMTemplate implements RFM (Recency, Frequency, Monetary) analysis
type RFMTemplate struct{}

func init() {
	Register(&RFMTemplate{})
}

func (t *RFMTemplate) Name() string {
	return "rfm"
}

func (t *RFMTemplate) Description() string {
	return "RFM Analysis - Segment customers based on Recency, Frequency, and Monetary value"
}

func (t *RFMTemplate) Keywords() []string {
	return []string{
		"rfm", "rfm analysis", "rfm分析",
		"customer segmentation", "客户分群", "客户细分",
		"recency frequency monetary",
	}
}

func (t *RFMTemplate) RequiredColumns() []string {
	return []string{"customer_id", "order_date", "amount"}
}

func (t *RFMTemplate) CanExecute(tables []TableInfo) bool {
	// Check if we have tables with customer, date, and amount-like columns
	for _, table := range tables {
		hasCustomer := false
		hasDate := false
		hasAmount := false

		for _, col := range table.Columns {
			lower := strings.ToLower(col)
			if strings.Contains(lower, "customer") || strings.Contains(lower, "user") || strings.Contains(lower, "client") {
				hasCustomer = true
			}
			if strings.Contains(lower, "date") || strings.Contains(lower, "time") || strings.Contains(lower, "created") {
				hasDate = true
			}
			if strings.Contains(lower, "amount") || strings.Contains(lower, "total") || strings.Contains(lower, "price") || strings.Contains(lower, "revenue") {
				hasAmount = true
			}
		}

		if hasCustomer && hasDate && hasAmount {
			return true
		}
	}
	return false
}

func (t *RFMTemplate) Execute(ctx context.Context, executor DataExecutor, dataSourceID string, onProgress ProgressCallback) (*TemplateResult, error) {
	if onProgress != nil {
		onProgress("schema", 10, "Getting database schema...", 1, 6)
	}

	// Step 1: Get schema
	tables, err := executor.GetSchema(ctx, dataSourceID)
	if err != nil {
		return &TemplateResult{Success: false, Error: fmt.Sprintf("Failed to get schema: %v", err)}, nil
	}

	// Step 2: Find best table for RFM
	if onProgress != nil {
		onProgress("analysis", 20, "Identifying data tables...", 2, 6)
	}

	var targetTable string
	var customerCol, dateCol, amountCol string

	for _, table := range tables {
		for _, col := range table.Columns {
			lower := strings.ToLower(col)
			if customerCol == "" && (strings.Contains(lower, "customerid") || strings.Contains(lower, "customer_id") || strings.Contains(lower, "userid") || strings.Contains(lower, "user_id")) {
				customerCol = col
				targetTable = table.Name
			}
			if dateCol == "" && (strings.Contains(lower, "orderdate") || strings.Contains(lower, "order_date") || strings.Contains(lower, "date") || strings.Contains(lower, "created")) {
				dateCol = col
			}
			if amountCol == "" && (strings.Contains(lower, "totalamount") || strings.Contains(lower, "total_amount") || strings.Contains(lower, "amount") || strings.Contains(lower, "total")) {
				amountCol = col
			}
		}
		if customerCol != "" && dateCol != "" && amountCol != "" {
			break
		}
	}

	if targetTable == "" || customerCol == "" || dateCol == "" || amountCol == "" {
		return &TemplateResult{
			Success: false,
			Error:   "Could not identify suitable columns for RFM analysis. Need customer ID, date, and amount columns.",
		}, nil
	}

	// Step 3: Execute RFM SQL query
	if onProgress != nil {
		onProgress("query", 40, "Executing RFM query...", 3, 6)
	}

	rfmSQL := fmt.Sprintf(`
		SELECT
			%s as CustomerID,
			%s as OrderDate,
			%s as TotalAmount
		FROM %s
		WHERE %s IS NOT NULL AND %s IS NOT NULL
	`, customerCol, dateCol, amountCol, targetTable, customerCol, amountCol)

	data, err := executor.ExecuteSQL(ctx, dataSourceID, rfmSQL)
	if err != nil {
		return &TemplateResult{Success: false, Error: fmt.Sprintf("SQL query failed: %v", err)}, nil
	}

	if len(data) == 0 {
		return &TemplateResult{Success: false, Error: "No data returned from query"}, nil
	}

	// Step 4: Execute Python analysis
	if onProgress != nil {
		onProgress("analysis", 60, "Running RFM analysis...", 4, 6)
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

# Load data
data = json.loads('''%s''')
df = pd.DataFrame(data)

# Convert date column
df['OrderDate'] = pd.to_datetime(df['OrderDate'])

# Calculate RFM
ref_date = df['OrderDate'].max()
rfm = df.groupby('CustomerID').agg({
    'OrderDate': lambda x: (ref_date - x.max()).days,
    'TotalAmount': ['count', 'sum']
}).reset_index()

rfm.columns = ['CustomerID', 'Recency', 'Frequency', 'Monetary']

# Score RFM (1-5 scale, handle duplicates)
rfm['R_Score'] = pd.qcut(rfm['Recency'].rank(method='first'), q=5, labels=[5,4,3,2,1], duplicates='drop')
rfm['F_Score'] = pd.qcut(rfm['Frequency'].rank(method='first'), q=5, labels=[1,2,3,4,5], duplicates='drop')
rfm['M_Score'] = pd.qcut(rfm['Monetary'].rank(method='first'), q=5, labels=[1,2,3,4,5], duplicates='drop')

# Convert to numeric for calculations
rfm['R_Score'] = pd.to_numeric(rfm['R_Score'])
rfm['F_Score'] = pd.to_numeric(rfm['F_Score'])
rfm['M_Score'] = pd.to_numeric(rfm['M_Score'])

# Calculate total RFM score
rfm['RFM_Score'] = rfm['R_Score'] + rfm['F_Score'] + rfm['M_Score']

# Segment customers
def segment(row):
    if row['RFM_Score'] >= 12:
        return 'Champions'
    elif row['RFM_Score'] >= 9:
        return 'Loyal'
    elif row['RFM_Score'] >= 6:
        return 'Potential'
    elif row['RFM_Score'] >= 4:
        return 'At Risk'
    else:
        return 'Lost'

rfm['Segment'] = rfm.apply(segment, axis=1)

# Print summary
print("=" * 50)
print("RFM ANALYSIS RESULTS")
print("=" * 50)
print(f"\nTotal Customers Analyzed: {len(rfm)}")
print("\nRFM Statistics:")
print(rfm[['Recency', 'Frequency', 'Monetary']].describe().round(2))
print("\nCustomer Segments:")
segment_summary = rfm.groupby('Segment').agg({
    'CustomerID': 'count',
    'Monetary': 'sum'
}).rename(columns={'CustomerID': 'Count', 'Monetary': 'Revenue'})
segment_summary['Pct'] = (segment_summary['Count'] / len(rfm) * 100).round(1)
print(segment_summary.sort_values('Revenue', ascending=False))

# Create visualization
fig, axes = plt.subplots(1, 2, figsize=(14, 5))

# Pie chart of segments
colors = {'Champions': '#2ecc71', 'Loyal': '#3498db', 'Potential': '#f1c40f', 'At Risk': '#e67e22', 'Lost': '#e74c3c'}
segment_counts = rfm['Segment'].value_counts()
axes[0].pie(segment_counts.values, labels=segment_counts.index, autopct='%%1.1f%%%%',
            colors=[colors.get(s, '#95a5a6') for s in segment_counts.index])
axes[0].set_title('Customer Segments Distribution')

# Bar chart of revenue by segment
revenue_by_segment = rfm.groupby('Segment')['Monetary'].sum().sort_values(ascending=False)
bars = axes[1].bar(revenue_by_segment.index, revenue_by_segment.values,
                   color=[colors.get(s, '#95a5a6') for s in revenue_by_segment.index])
axes[1].set_title('Revenue by Customer Segment')
axes[1].set_ylabel('Total Revenue')
axes[1].tick_params(axis='x', rotation=45)

plt.tight_layout()
plt.savefig('chart.png', dpi=150, bbox_inches='tight')
print("\nVisualization saved to chart.png")

# Save CSV
rfm.to_csv('rfm_results.csv', index=False)
print("Detailed results saved to rfm_results.csv")
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
		onProgress("complete", 100, "RFM analysis complete", 6, 6)
	}

	return &TemplateResult{
		Success: true,
		Output:  output,
	}, nil
}
