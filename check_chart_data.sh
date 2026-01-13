#!/bin/bash
# Check if chart_data is attached to user messages in a session

SESSION_ID=$1
if [ -z "$SESSION_ID" ]; then
    echo "Usage: $0 <session_id>"
    echo "Example: $0 1768185469755427700"
    exit 1
fi

HISTORY_FILE="/c/Users/ma139/RapidBI/sessions/$SESSION_ID/history.json"

if [ ! -f "$HISTORY_FILE" ]; then
    echo "❌ Session not found: $SESSION_ID"
    exit 1
fi

echo "🔍 Checking session: $SESSION_ID"
echo ""

# Check for chart files
echo "📁 Chart files in session:"
jq -r '.files[]? | select(.type=="image") | "  ✓ \(.name) (\(.size) bytes)"' "$HISTORY_FILE" 2>/dev/null || echo "  (none)"

echo ""
echo "💬 User messages with chart_data:"
jq -r '.messages[]? | select(.role=="user" and .chart_data != null) | "  ✓ Message ID: \(.id)\n    Content: \(.content[:60])...\n    Chart Type: \(.chart_data.type)"' "$HISTORY_FILE" 2>/dev/null || echo "  ❌ No user messages have chart_data attached"

echo ""
echo "Summary:"
CHART_FILES=$(jq '[.files[]? | select(.type=="image")] | length' "$HISTORY_FILE" 2>/dev/null || echo 0)
USER_MSG_WITH_CHARTS=$(jq '[.messages[]? | select(.role=="user" and .chart_data != null)] | length' "$HISTORY_FILE" 2>/dev/null || echo 0)

echo "  Chart files: $CHART_FILES"
echo "  User messages with chart_data: $USER_MSG_WITH_CHARTS"

if [ "$CHART_FILES" -gt 0 ] && [ "$USER_MSG_WITH_CHARTS" -gt 0 ]; then
    echo "  ✅ Chart data is properly attached!"
elif [ "$CHART_FILES" -gt 0 ] && [ "$USER_MSG_WITH_CHARTS" -eq 0 ]; then
    echo "  ❌ Charts exist but NOT attached to user messages (bug still present)"
else
    echo "  ℹ️  No charts in this session"
fi
