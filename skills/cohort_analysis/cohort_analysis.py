import json
import pandas as pd
import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import seaborn as sns
from datetime import datetime

print("=" * 60)
print("COHORT ANALYSIS - User Retention & Lifecycle")
print("=" * 60)

# This is a template - placeholders will be replaced by the system
# Available placeholders: {{table}}, {{user_id}}, {{date}}, {{event}}
# SQL data will be injected as JSON

# Sample data structure (will be replaced with actual data from SQL query)
# data = [{"user_id": 1, "date": "2024-01-15", "event": "purchase"}, ...]

# For template demonstration:
data_json = '''{{data}}'''  # This will be injected by the system

if data_json == '{{data}}':
    print("Note: This is a template. Data will be injected when executed.")
    print("\nExpected data format:")
    print("  - user_id: User identifier")
    print("  - date: Event date (YYYY-MM-DD format)")
    print("  - event: Event type (optional)")
    exit(0)

# Parse data
df = pd.DataFrame(json.loads(data_json))

# Convert date to datetime
df['date'] = pd.to_datetime(df['date'])

# Extract cohort month (first activity month per user)
df['cohort_month'] = df.groupby('user_id')['date'].transform('min').dt.to_period('M')
df['activity_month'] = df['date'].dt.to_period('M')

# Calculate periods since cohort
df['period_number'] = (df['activity_month'] - df['cohort_month']).apply(lambda x: x.n)

# Build cohort analysis table
cohort_data = df.groupby(['cohort_month', 'period_number'])['user_id'].nunique().reset_index()
cohort_pivot = cohort_data.pivot(index='cohort_month', columns='period_number', values='user_id')

# Calculate cohort sizes
cohort_sizes = cohort_pivot.iloc[:, 0]

# Calculate retention rates
retention = cohort_pivot.divide(cohort_sizes, axis=0) * 100

print("\n📊 Cohort Retention Analysis")
print("-" * 60)
print(f"Total Cohorts Analyzed: {len(cohort_sizes)}")
print(f"Date Range: {df['date'].min().date()} to {df['date'].max().date()}")
print(f"Total Users: {df['user_id'].nunique()}")

print("\n📈 Cohort Sizes:")
print(cohort_sizes.head(10))

print("\n📈 Retention Rates (%):")
print(retention.head(10).round(1))

# Calculate average retention by period
avg_retention = retention.mean(axis=0)
print("\n📊 Average Retention by Period:")
for period, rate in avg_retention.items():
    if period == 0:
        print(f"  Period {period} (Cohort Month): {rate:.1f}%")
    else:
        print(f"  Period {period} (Month +{period}): {rate:.1f}%")

# Create visualization
fig, axes = plt.subplots(2, 1, figsize=(14, 12))

# Heatmap of retention rates
sns.heatmap(retention.iloc[:, :min(12, retention.shape[1])],
            annot=True, fmt='.0f', cmap='RdYlGn',
            cbar_kws={'label': 'Retention %'},
            ax=axes[0])
axes[0].set_title('Cohort Retention Heatmap (%)', fontsize=14, fontweight='bold')
axes[0].set_xlabel('Months Since First Activity')
axes[0].set_ylabel('Cohort Month')

# Retention curves
colors = plt.cm.viridis(np.linspace(0, 1, min(10, len(retention))))
for idx, (cohort, row) in enumerate(retention.head(10).iterrows()):
    axes[1].plot(row.index, row.values, marker='o', label=str(cohort),
                color=colors[idx], linewidth=2, markersize=6)

axes[1].set_title('Retention Curves by Cohort', fontsize=14, fontweight='bold')
axes[1].set_xlabel('Months Since First Activity')
axes[1].set_ylabel('Retention Rate (%)')
axes[1].legend(title='Cohort', bbox_to_anchor=(1.05, 1), loc='upper left')
axes[1].grid(True, alpha=0.3)
axes[1].set_ylim(0, 105)

plt.tight_layout()
plt.savefig('cohort_retention.png', dpi=150, bbox_inches='tight')
print("\n✅ Visualization saved: cohort_retention.png")

# Save detailed results
output_df = retention.round(2)
output_df.to_csv('cohort_retention_details.csv')
print("✅ Detailed results saved: cohort_retention_details.csv")

# Calculate key insights
print("\n💡 Key Insights:")
first_month_retention = avg_retention.get(1, 0)
third_month_retention = avg_retention.get(3, 0)
sixth_month_retention = avg_retention.get(6, 0)

print(f"  • Month 1 Retention: {first_month_retention:.1f}%")
print(f"  • Month 3 Retention: {third_month_retention:.1f}%")
print(f"  • Month 6 Retention: {sixth_month_retention:.1f}%")

if first_month_retention > 0:
    retention_drop = 100 - first_month_retention
    print(f"  • Month 1 Drop-off: {retention_drop:.1f}%")

# Find best and worst cohorts
best_cohort = retention.iloc[:, 1:6].mean(axis=1).idxmax()
worst_cohort = retention.iloc[:, 1:6].mean(axis=1).idxmin()
print(f"  • Best Performing Cohort: {best_cohort}")
print(f"  • Needs Improvement Cohort: {worst_cohort}")

print("\n" + "=" * 60)
print("Analysis Complete!")
print("=" * 60)
