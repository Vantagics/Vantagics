package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// TimeTool provides local time and date information
type TimeTool struct {
	logFunc func(string)
}

// TimeToolInput defines the input parameters for the time tool
type TimeToolInput struct {
	// Query type: "current_time", "current_date", "datetime", "weekday", "timestamp", "timezone"
	QueryType string `json:"query_type" jsonschema:"description=Type of time query: current_time (时间), current_date (日期), datetime (日期时间), weekday (星期), timestamp (时间戳), timezone (时区)"`
	// Optional timezone (e.g., "Asia/Shanghai", "America/New_York")
	Timezone string `json:"timezone,omitempty" jsonschema:"description=Optional timezone name (e.g., Asia/Shanghai, America/New_York). If not specified, uses local timezone."`
}

// TimeToolOutput defines the output of the time tool
type TimeToolOutput struct {
	CurrentTime   string `json:"current_time,omitempty"`
	CurrentDate   string `json:"current_date,omitempty"`
	DateTime      string `json:"datetime,omitempty"`
	Weekday       string `json:"weekday,omitempty"`
	WeekdayCN     string `json:"weekday_cn,omitempty"`
	Timestamp     int64  `json:"timestamp,omitempty"`
	Timezone      string `json:"timezone,omitempty"`
	TimezoneOffset string `json:"timezone_offset,omitempty"`
	Year          int    `json:"year,omitempty"`
	Month         int    `json:"month,omitempty"`
	Day           int    `json:"day,omitempty"`
	Hour          int    `json:"hour,omitempty"`
	Minute        int    `json:"minute,omitempty"`
	Second        int    `json:"second,omitempty"`
}

// NewTimeTool creates a new time tool instance
func NewTimeTool(logFunc func(string)) *TimeTool {
	return &TimeTool{
		logFunc: logFunc,
	}
}

// Info returns the tool information
func (t *TimeTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "get_local_time",
		Desc: "Get current local time, date, weekday, timestamp, or timezone information. Use this tool when user asks about current time, date, what day it is, etc. This provides accurate local system time without needing internet access.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"query_type": {
				Type:     schema.String,
				Desc:     "Type of time query: 'current_time' for time only, 'current_date' for date only, 'datetime' for both, 'weekday' for day of week, 'timestamp' for Unix timestamp, 'timezone' for timezone info",
				Required: true,
			},
			"timezone": {
				Type:     schema.String,
				Desc:     "Optional timezone name (e.g., 'Asia/Shanghai', 'America/New_York'). If not specified, uses local system timezone.",
				Required: false,
			},
		}),
	}, nil
}

// InvokableRun executes the time tool
func (t *TimeTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	t.log("[TIME-TOOL] Invoked with args: %s", argumentsInJSON)

	var input TimeToolInput
	if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
		return "", fmt.Errorf("failed to parse input: %v", err)
	}

	// Get the time in the specified timezone or local timezone
	var now time.Time
	var loc *time.Location
	var err error

	if input.Timezone != "" {
		loc, err = time.LoadLocation(input.Timezone)
		if err != nil {
			t.log("[TIME-TOOL] Invalid timezone %s, using local: %v", input.Timezone, err)
			loc = time.Local
		}
	} else {
		loc = time.Local
	}
	now = time.Now().In(loc)

	// Build output based on query type
	output := TimeToolOutput{
		Year:      now.Year(),
		Month:     int(now.Month()),
		Day:       now.Day(),
		Hour:      now.Hour(),
		Minute:    now.Minute(),
		Second:    now.Second(),
		Timestamp: now.Unix(),
		Timezone:  loc.String(),
	}

	// Get timezone offset
	_, offset := now.Zone()
	hours := offset / 3600
	minutes := (offset % 3600) / 60
	if minutes < 0 {
		minutes = -minutes
	}
	output.TimezoneOffset = fmt.Sprintf("UTC%+d:%02d", hours, minutes)

	// Get weekday in both English and Chinese
	weekdays := []string{"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"}
	weekdaysCN := []string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}
	output.Weekday = weekdays[now.Weekday()]
	output.WeekdayCN = weekdaysCN[now.Weekday()]

	// Format based on query type
	switch input.QueryType {
	case "current_time":
		output.CurrentTime = now.Format("15:04:05")
	case "current_date":
		output.CurrentDate = now.Format("2006-01-02")
	case "datetime":
		output.DateTime = now.Format("2006-01-02 15:04:05")
		output.CurrentTime = now.Format("15:04:05")
		output.CurrentDate = now.Format("2006-01-02")
	case "weekday":
		// Already set above
	case "timestamp":
		// Already set above
	case "timezone":
		// Already set above
	default:
		// Return everything for unknown query type
		output.DateTime = now.Format("2006-01-02 15:04:05")
		output.CurrentTime = now.Format("15:04:05")
		output.CurrentDate = now.Format("2006-01-02")
	}

	// Marshal output to JSON
	result, err := json.Marshal(output)
	if err != nil {
		return "", fmt.Errorf("failed to marshal output: %v", err)
	}

	t.log("[TIME-TOOL] Result: %s", string(result))
	return string(result), nil
}

// log logs a message using the provided log function
func (t *TimeTool) log(format string, args ...interface{}) {
	if t.logFunc != nil {
		t.logFunc(fmt.Sprintf(format, args...))
	}
}
