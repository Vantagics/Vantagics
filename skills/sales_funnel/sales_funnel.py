import json
import pandas as pd
import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt

print("=" * 60)
print("SALES FUNNEL ANALYSIS - Conversion Optimization")
print("=" * 60)

# Template placeholders: {{data}}, {{table}}, {{user_id}}, {{stage}}, {{date}}
data_json = '''{{data}}'''

if data_json == '{{data}}':
    print("Note: This is a template. Data will be injected when executed.")
    print("\nExpected data format:")
    print("  - user_id: Lead/Customer identifier")
    print("  - stage: Funnel stage name")
    print("  - date: Timestamp of stage entry")
    exit(0)

# Parse data
df = pd.DataFrame(json.loads(data_json))

# Convert date
df['date'] = pd.to_datetime(df['date'])

# Define standard funnel stages (can be customized)
funnel_stages = {{stages}}.split(',') if '{{stages}}' != '' else df['stage'].unique().tolist()

print(f"\n📊 Analyzing Funnel with {len(funnel_stages)} stages:")
for i, stage in enumerate(funnel_stages, 1):
    print(f"  {i}. {stage}")

# Calculate metrics for each stage
funnel_metrics = []
total_leads = df['user_id'].nunique()

for i, stage in enumerate(funnel_stages):
    stage_users = df[df['stage'] == stage]['user_id'].nunique()

    conversion_rate = (stage_users / total_leads * 100) if total_leads > 0 else 0

    if i > 0:
        prev_stage_users = funnel_metrics[i-1]['users']
        stage_conversion = (stage_users / prev_stage_users * 100) if prev_stage_users > 0 else 0
    else:
        stage_conversion = 100.0

    funnel_metrics.append({
        'stage': stage,
        'users': stage_users,
        'overall_conversion': conversion_rate,
        'stage_conversion': stage_conversion
    })

funnel_df = pd.DataFrame(funnel_metrics)

print("\n📈 Funnel Metrics:")
print("-" * 60)
print(f"{'Stage':<20} {'Users':<10} {'Stage Conv%':<12} {'Overall Conv%':<12}")
print("-" * 60)
for _, row in funnel_df.iterrows():
    print(f"{row['stage']:<20} {row['users']:<10} {row['stage_conversion']:>10.1f}% {row['overall_conversion']:>13.1f}%")

# Calculate drop-offs
print("\n📉 Drop-off Analysis:")
print("-" * 60)
for i in range(len(funnel_df) - 1):
    current_users = funnel_df.iloc[i]['users']
    next_users = funnel_df.iloc[i + 1]['users']
    dropoff = current_users - next_users
    dropoff_rate = (dropoff / current_users * 100) if current_users > 0 else 0

    print(f"{funnel_df.iloc[i]['stage']} → {funnel_df.iloc[i+1]['stage']}:")
    print(f"  Lost: {dropoff} users ({dropoff_rate:.1f}%)")

# Create visualization
fig, axes = plt.subplots(1, 2, figsize=(16, 6))

# Funnel chart
stages = funnel_df['stage'].tolist()
users = funnel_df['users'].tolist()
colors = plt.cm.Blues(np.linspace(0.4, 0.9, len(stages)))

# Create funnel
for i, (stage, count, color) in enumerate(zip(stages, users, colors)):
    width = count / users[0] if users[0] > 0 else 0
    axes[0].barh(i, width, color=color, edgecolor='white', linewidth=2)
    axes[0].text(width/2, i, f'{stage}\n{count} ({funnel_df.iloc[i]["overall_conversion"]:.1f}%)',
                ha='center', va='center', fontweight='bold', fontsize=10)

axes[0].set_yticks([])
axes[0].set_xlim(0, 1)
axes[0].set_title('Sales Funnel - User Flow', fontsize=14, fontweight='bold')
axes[0].set_xlabel('Conversion Rate')
axes[0].invert_yaxis()

# Conversion rates bar chart
stage_conversions = funnel_df['stage_conversion'].tolist()
x = np.arange(len(stages))
bars = axes[1].bar(x, stage_conversions, color=colors, edgecolor='black', linewidth=1.5)

# Add value labels on bars
for i, (bar, val) in enumerate(zip(bars, stage_conversions)):
    height = bar.get_height()
    axes[1].text(bar.get_x() + bar.get_width()/2., height,
                f'{val:.1f}%', ha='center', va='bottom', fontweight='bold', fontsize=10)

axes[1].set_xticks(x)
axes[1].set_xticklabels(stages, rotation=45, ha='right')
axes[1].set_ylabel('Stage Conversion Rate (%)')
axes[1].set_title('Stage-to-Stage Conversion Rates', fontsize=14, fontweight='bold')
axes[1].set_ylim(0, 110)
axes[1].grid(axis='y', alpha=0.3)

plt.tight_layout()
plt.savefig('sales_funnel.png', dpi=150, bbox_inches='tight')
print("\n✅ Visualization saved: sales_funnel.png")

# Save results
funnel_df.to_csv('funnel_analysis.csv', index=False)
print("✅ Detailed metrics saved: funnel_analysis.csv")

# Key insights
print("\n💡 Key Insights:")
total_conversion = (funnel_df.iloc[-1]['users'] / funnel_df.iloc[0]['users'] * 100) if funnel_df.iloc[0]['users'] > 0 else 0
print(f"  • Overall Conversion Rate: {total_conversion:.1f}%")
print(f"  • Total Leads: {funnel_df.iloc[0]['users']}")
print(f"  • Closed Deals: {funnel_df.iloc[-1]['users']}")

# Find biggest drop-off
biggest_dropoff_idx = funnel_df['stage_conversion'].iloc[1:].idxmin()
if not pd.isna(biggest_dropoff_idx):
    print(f"  • Biggest Drop-off: {funnel_df.iloc[biggest_dropoff_idx]['stage']} ({funnel_df.iloc[biggest_dropoff_idx]['stage_conversion']:.1f}% conversion)")

# Find best conversion
best_stage_idx = funnel_df['stage_conversion'].iloc[1:].idxmax()
if not pd.isna(best_stage_idx):
    print(f"  • Best Conversion: {funnel_df.iloc[best_stage_idx]['stage']} ({funnel_df.iloc[best_stage_idx]['stage_conversion']:.1f}% conversion)")

print("\n" + "=" * 60)
print("Analysis Complete!")
print("=" * 60)
