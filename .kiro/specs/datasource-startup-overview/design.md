# Design Document: Data Source Startup Overview

## Overview

This feature enhances the application startup experience by automatically displaying data source statistics and providing one-click analysis capabilities. When users launch the application, they immediately see key metrics about their data infrastructure (total count, breakdown by driver type) and can initiate analysis with a single click through a smart insight card.

The design integrates with existing components (SmartInsight, DataSourceService, Agent) and follows the established Wails architecture pattern with Go backend and React frontend.

## Architecture

### System Components

```
┌─────────────────────────────────────────────────────────────┐
│                        Frontend (React)                      │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  App.tsx (Startup Orchestration)                       │ │
│  │    - Fetches statistics on mount                       │ │
│  │    - Manages loading/error states                      │ │
│  └────────────────┬───────────────────────────────────────┘ │
│                   │                                          │
│  ┌────────────────▼───────────────────────────────────────┐ │
│  │  DataSourceOverview Component (New)                    │ │
│  │    - Displays total count                              │ │
│  │    - Renders breakdown by driver type                  │ │
│  │    - Handles loading/error states                      │ │
│  └────────────────┬───────────────────────────────────────┘ │
│                   │                                          │
│  ┌────────────────▼───────────────────────────────────────┐ │
│  │  SmartInsight Component (Enhanced)                     │ │
│  │    - Displays "One-Click Analysis" insight             │ │
│  │    - Handles click to trigger analysis                 │ │
│  │    - Shows data source selection modal if needed       │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────┬───────────────────────────────────────┘
                      │ Wails Bindings
┌─────────────────────▼───────────────────────────────────────┐
│                     Backend (Go)                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  App.GetDataSourceStatistics() (New)                   │ │
│  │    - Calls DataSourceService                           │ │
│  │    - Returns statistics structure                      │ │
│  └────────────────┬───────────────────────────────────────┘ │
│                   │                                          │
│  ┌────────────────▼───────────────────────────────────────┐ │
│  │  DataSourceService                                      │ │
│  │    - LoadDataSources() (existing)                      │ │
│  │    - CalculateStatistics() (new)                       │ │
│  │    - Groups by Type field                              │ │
│  └────────────────┬───────────────────────────────────────┘ │
│                   │                                          │
│  ┌────────────────▼───────────────────────────────────────┐ │
│  │  App.StartDataSourceAnalysis(dataSourceID) (New)      │ │
│  │    - Validates data source exists                      │ │
│  │    - Calls existing analysis flow                      │ │
│  │    - Returns analysis session ID                       │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### Data Flow

1. **Startup Flow**:
   - App.tsx mounts → calls `GetDataSourceStatistics()`
   - Backend loads data sources → calculates statistics
   - Frontend receives statistics → renders overview + smart insight

2. **Analysis Flow**:
   - User clicks smart insight → triggers onClick handler
   - If multiple sources: show selection modal
   - If single source: directly call `StartDataSourceAnalysis(id)`
   - Backend initiates analysis → returns session ID
   - Frontend navigates to analysis view or shows progress

## Components and Interfaces

### Backend API (Go)

#### New Method: GetDataSourceStatistics

```go
// DataSourceStatistics holds aggregated statistics about data sources
type DataSourceStatistics struct {
    TotalCount      int                    `json:"total_count"`
    BreakdownByType map[string]int         `json:"breakdown_by_type"`
    DataSources     []DataSourceSummary    `json:"data_sources"`
}

// DataSourceSummary provides minimal info for selection UI
type DataSourceSummary struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Type string `json:"type"`
}

// GetDataSourceStatistics returns aggregated statistics about all data sources
// Validates: Requirements 1.1, 1.2, 1.3, 1.4, 1.5
func (a *App) GetDataSourceStatistics() (*DataSourceStatistics, error) {
    if a.dataSourceService == nil {
        return nil, fmt.Errorf("data source service not initialized")
    }
    
    // Load all data sources
    dataSources, err := a.dataSourceService.LoadDataSources()
    if err != nil {
        return nil, fmt.Errorf("failed to load data sources: %w", err)
    }
    
    // Calculate statistics
    stats := &DataSourceStatistics{
        TotalCount:      len(dataSources),
        BreakdownByType: make(map[string]int),
        DataSources:     make([]DataSourceSummary, 0, len(dataSources)),
    }
    
    // Group by type and build summaries
    for _, ds := range dataSources {
        stats.BreakdownByType[ds.Type]++
        stats.DataSources = append(stats.DataSources, DataSourceSummary{
            ID:   ds.ID,
            Name: ds.Name,
            Type: ds.Type,
        })
    }
    
    return stats, nil
}
```

#### New Method: StartDataSourceAnalysis

```go
// StartDataSourceAnalysis initiates analysis for a specific data source
// Returns the analysis session/thread ID
// Validates: Requirements 4.1, 4.2, 4.5
func (a *App) StartDataSourceAnalysis(dataSourceID string) (string, error) {
    if a.dataSourceService == nil {
        return "", fmt.Errorf("data source service not initialized")
    }
    
    // Validate data source exists
    dataSources, err := a.dataSourceService.LoadDataSources()
    if err != nil {
        return "", fmt.Errorf("failed to load data sources: %w", err)
    }
    
    var targetDS *agent.DataSource
    for _, ds := range dataSources {
        if ds.ID == dataSourceID {
            targetDS = &ds
            break
        }
    }
    
    if targetDS == nil {
        return "", fmt.Errorf("data source not found: %s", dataSourceID)
    }
    
    // Create a new chat thread for this analysis
    threadID := fmt.Sprintf("ds-analysis-%s-%d", dataSourceID, time.Now().UnixMilli())
    
    // Construct analysis prompt
    prompt := fmt.Sprintf("请分析数据源 '%s' (%s)，提供数据概览、关键指标和洞察。", 
        targetDS.Name, targetDS.Type)
    
    // Use existing SendMessage to initiate analysis
    // This leverages the existing analysis infrastructure
    _, err = a.SendMessage(threadID, prompt, "", "")
    if err != nil {
        return "", fmt.Errorf("failed to start analysis: %w", err)
    }
    
    a.Log(fmt.Sprintf("[DATASOURCE-ANALYSIS] Started analysis for %s (thread: %s)", 
        dataSourceID, threadID))
    
    return threadID, nil
}
```

### Frontend Components (TypeScript/React)

#### New Component: DataSourceOverview

```typescript
// src/frontend/src/components/DataSourceOverview.tsx

interface DataSourceStatistics {
    total_count: number;
    breakdown_by_type: Record<string, number>;
    data_sources: DataSourceSummary[];
}

interface DataSourceSummary {
    id: string;
    name: string;
    type: string;
}

interface DataSourceOverviewProps {
    onAnalyzeClick?: (dataSourceId: string) => void;
}

const DataSourceOverview: React.FC<DataSourceOverviewProps> = ({ onAnalyzeClick }) => {
    const [statistics, setStatistics] = useState<DataSourceStatistics | null>(null);
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    
    useEffect(() => {
        loadStatistics();
    }, []);
    
    const loadStatistics = async () => {
        try {
            setLoading(true);
            setError(null);
            const stats = await GetDataSourceStatistics();
            setStatistics(stats);
        } catch (err) {
            setError(err instanceof Error ? err.message : 'Failed to load statistics');
        } finally {
            setLoading(false);
        }
    };
    
    if (loading) {
        return (
            <div className="data-source-overview loading">
                <div className="spinner" />
                <p>加载数据源信息...</p>
            </div>
        );
    }
    
    if (error) {
        return (
            <div className="data-source-overview error">
                <p className="error-message">{error}</p>
                <button onClick={loadStatistics}>重试</button>
            </div>
        );
    }
    
    if (!statistics || statistics.total_count === 0) {
        return (
            <div className="data-source-overview empty">
                <p>暂无数据源</p>
            </div>
        );
    }
    
    return (
        <div className="data-source-overview">
            <div className="overview-header">
                <h3>数据源概览</h3>
                <div className="total-count">
                    <span className="label">总数:</span>
                    <span className="value">{statistics.total_count}</span>
                </div>
            </div>
            
            <div className="breakdown">
                <h4>按类型统计</h4>
                <div className="breakdown-list">
                    {Object.entries(statistics.breakdown_by_type).map(([type, count]) => (
                        <div key={type} className="breakdown-item">
                            <span className="type-name">{type}</span>
                            <span className="type-count">{count}</span>
                        </div>
                    ))}
                </div>
            </div>
            
            {/* Smart Insight for One-Click Analysis */}
            <DataSourceAnalysisInsight 
                statistics={statistics}
                onAnalyzeClick={onAnalyzeClick}
            />
        </div>
    );
};
```

#### New Component: DataSourceAnalysisInsight

```typescript
// src/frontend/src/components/DataSourceAnalysisInsight.tsx

interface DataSourceAnalysisInsightProps {
    statistics: DataSourceStatistics;
    onAnalyzeClick?: (dataSourceId: string) => void;
}

const DataSourceAnalysisInsight: React.FC<DataSourceAnalysisInsightProps> = ({ 
    statistics, 
    onAnalyzeClick 
}) => {
    const [showSelection, setShowSelection] = useState(false);
    const [analyzing, setAnalyzing] = useState(false);
    
    const handleAnalyzeClick = async () => {
        // If multiple data sources, show selection modal
        if (statistics.data_sources.length > 1) {
            setShowSelection(true);
            return;
        }
        
        // If single data source, analyze directly
        if (statistics.data_sources.length === 1) {
            await startAnalysis(statistics.data_sources[0].id);
        }
    };
    
    const startAnalysis = async (dataSourceId: string) => {
        try {
            setAnalyzing(true);
            const threadId = await StartDataSourceAnalysis(dataSourceId);
            
            // Notify parent or navigate to analysis view
            if (onAnalyzeClick) {
                onAnalyzeClick(dataSourceId);
            }
            
            // Could also navigate to chat with the thread ID
            // navigate(`/chat/${threadId}`);
            
        } catch (err) {
            console.error('Failed to start analysis:', err);
            alert('分析启动失败: ' + (err instanceof Error ? err.message : '未知错误'));
        } finally {
            setAnalyzing(false);
            setShowSelection(false);
        }
    };
    
    const insightText = statistics.total_count === 1
        ? `发现 1 个数据源 (${statistics.data_sources[0].type})，点击开始智能分析`
        : `发现 ${statistics.total_count} 个数据源，点击选择并开始智能分析`;
    
    return (
        <>
            <SmartInsight
                text={insightText}
                icon="trending-up"
                onClick={handleAnalyzeClick}
            />
            
            {showSelection && (
                <DataSourceSelectionModal
                    dataSources={statistics.data_sources}
                    onSelect={startAnalysis}
                    onCancel={() => setShowSelection(false)}
                />
            )}
        </>
    );
};
```

#### New Component: DataSourceSelectionModal

```typescript
// src/frontend/src/components/DataSourceSelectionModal.tsx

interface DataSourceSelectionModalProps {
    dataSources: DataSourceSummary[];
    onSelect: (dataSourceId: string) => void;
    onCancel: () => void;
}

const DataSourceSelectionModal: React.FC<DataSourceSelectionModalProps> = ({
    dataSources,
    onSelect,
    onCancel
}) => {
    return (
        <div className="modal-overlay" onClick={onCancel}>
            <div className="modal-content" onClick={(e) => e.stopPropagation()}>
                <h3>选择要分析的数据源</h3>
                
                <div className="data-source-list">
                    {dataSources.map((ds) => (
                        <div 
                            key={ds.id}
                            className="data-source-item"
                            onClick={() => onSelect(ds.id)}
                        >
                            <div className="ds-name">{ds.name}</div>
                            <div className="ds-type">{ds.type}</div>
                        </div>
                    ))}
                </div>
                
                <button className="cancel-button" onClick={onCancel}>
                    取消
                </button>
            </div>
        </div>
    );
};
```

### Integration with App.tsx

```typescript
// src/frontend/src/App.tsx (modifications)

function App() {
    // ... existing state ...
    
    // Add data source overview to main UI
    return (
        <div className="app">
            <Sidebar />
            
            <main className="main-content">
                {/* Add overview at the top of main content */}
                <DataSourceOverview 
                    onAnalyzeClick={(dsId) => {
                        // Handle analysis start - could open chat sidebar
                        // or navigate to analysis view
                        console.log('Starting analysis for:', dsId);
                    }}
                />
                
                {/* Existing dashboard components */}
                <DraggableDashboard />
            </main>
        </div>
    );
}
```

## Data Models

### DataSourceStatistics (Go)

```go
type DataSourceStatistics struct {
    TotalCount      int                    `json:"total_count"`
    BreakdownByType map[string]int         `json:"breakdown_by_type"`
    DataSources     []DataSourceSummary    `json:"data_sources"`
}
```

- `TotalCount`: Total number of configured data sources
- `BreakdownByType`: Map of driver type to count (e.g., {"mysql": 3, "postgresql": 2})
- `DataSources`: List of data source summaries for selection UI

### DataSourceSummary (Go)

```go
type DataSourceSummary struct {
    ID   string `json:"id"`
    Name string `json:"name"`
    Type string `json:"type"`
}
```

Minimal information needed for selection UI and analysis initiation.

### Frontend TypeScript Interfaces

```typescript
interface DataSourceStatistics {
    total_count: number;
    breakdown_by_type: Record<string, number>;
    data_sources: DataSourceSummary[];
}

interface DataSourceSummary {
    id: string;
    name: string;
    type: string;
}
```

## Correctness Properties

*A property is a characteristic or behavior that should hold true across all valid executions of a system—essentially, a formal statement about what the system should do. Properties serve as the bridge between human-readable specifications and machine-verifiable correctness guarantees.*

### Property 1: Statistics Calculation Correctness

*For any* list of data sources, the calculated statistics should satisfy:
- Total count equals the length of the data source list
- Sum of all breakdown counts equals the total count
- Each data source appears in exactly one type category
- Breakdown contains an entry for each unique driver type present

**Validates: Requirements 1.2, 1.3, 1.4**

### Property 2: Statistics Rendering Completeness

*For any* valid DataSourceStatistics object, the rendered UI should display:
- The total count value
- All driver types from the breakdown
- The count for each driver type
- All required UI elements (header, breakdown section)

**Validates: Requirements 2.2, 2.3, 2.4**

### Property 3: Smart Insight Structure Completeness

*For any* generated smart insight for data source analysis, the insight should contain:
- A non-empty descriptive text mentioning data sources
- An icon identifier
- An onClick handler function
- Text that reflects the actual data source count

**Validates: Requirements 3.1, 3.2, 3.3**

### Property 4: Selection UI Completeness

*For any* list of data sources with length > 1, when the selection modal is displayed, it should:
- Show all data sources from the list
- Display both name and type for each data source
- Provide a cancel option
- Have click handlers for each data source

**Validates: Requirements 6.1, 6.2, 6.4**

### Property 5: Analysis Initiation Correctness

*For any* valid data source ID, calling StartDataSourceAnalysis should:
- Return a non-empty thread ID on success
- Return an error if the data source doesn't exist
- Create a new chat thread
- Log the analysis initiation

**Validates: Requirements 4.1, 4.2**

## Error Handling

### Backend Error Scenarios

1. **Data Source Service Not Initialized**
   - Return error: "data source service not initialized"
   - Frontend displays error message with retry option

2. **Failed to Load Data Sources**
   - Log error with details
   - Return error to frontend
   - Frontend shows error message: "无法加载数据源信息"

3. **Data Source Not Found (Analysis)**
   - Return error: "data source not found: {id}"
   - Frontend displays: "数据源不存在，请刷新后重试"

4. **Analysis Initiation Failed**
   - Log error with details
   - Return error to frontend
   - Frontend shows: "分析启动失败: {error details}"

### Frontend Error Scenarios

1. **Statistics Fetch Failed**
   - Display error message in overview component
   - Show retry button
   - Log error to console

2. **Analysis Start Failed**
   - Show alert with error message
   - Reset analyzing state
   - Close selection modal

3. **Network Timeout**
   - Show loading indicator for up to 5 seconds
   - If timeout, show error with retry option

### Graceful Degradation

- If statistics fetch fails, app continues to load other components
- If smart insight generation fails, overview still displays statistics
- If analysis fails to start, user can retry or manually navigate to data source

## Testing Strategy

### Unit Tests

Unit tests focus on specific examples, edge cases, and error conditions:

**Backend (Go)**:
- Test `GetDataSourceStatistics()` with empty data source list
- Test `GetDataSourceStatistics()` with single data source
- Test `GetDataSourceStatistics()` with multiple types
- Test `StartDataSourceAnalysis()` with invalid ID
- Test `StartDataSourceAnalysis()` with valid ID
- Test error handling when service not initialized

**Frontend (TypeScript/React)**:
- Test DataSourceOverview renders loading state
- Test DataSourceOverview renders error state with retry button
- Test DataSourceOverview renders empty state
- Test DataSourceAnalysisInsight with single data source (no modal)
- Test DataSourceAnalysisInsight with multiple sources (shows modal)
- Test DataSourceSelectionModal renders all data sources
- Test modal cancel functionality

### Property-Based Tests

Property tests verify universal properties across all inputs. Each test should run a minimum of 100 iterations.

**Backend Property Tests (Go)**:

1. **Property 1: Statistics Calculation Correctness**
   - Generate random lists of data sources with various types
   - Verify total count = list length
   - Verify sum of breakdown = total count
   - Verify each type in breakdown appears in data sources
   - **Tag**: Feature: datasource-startup-overview, Property 1: Statistics calculation correctness

2. **Property 5: Analysis Initiation Correctness**
   - Generate random valid data source IDs
   - Verify thread ID is non-empty on success
   - Verify error for non-existent IDs
   - **Tag**: Feature: datasource-startup-overview, Property 5: Analysis initiation correctness

**Frontend Property Tests (TypeScript)**:

1. **Property 2: Statistics Rendering Completeness**
   - Generate random DataSourceStatistics objects
   - Verify all breakdown entries are rendered
   - Verify total count is displayed
   - **Tag**: Feature: datasource-startup-overview, Property 2: Statistics rendering completeness

2. **Property 3: Smart Insight Structure Completeness**
   - Generate random statistics with varying counts
   - Verify insight text reflects count
   - Verify onClick handler exists
   - **Tag**: Feature: datasource-startup-overview, Property 3: Smart insight structure completeness

3. **Property 4: Selection UI Completeness**
   - Generate random lists of data sources (length > 1)
   - Verify all sources appear in modal
   - Verify name and type displayed for each
   - **Tag**: Feature: datasource-startup-overview, Property 4: Selection UI completeness

### Integration Tests

- Test full flow: startup → fetch statistics → display → click insight → start analysis
- Test with real database containing various data sources
- Test error recovery flows
- Test UI navigation after analysis starts

### Testing Tools

- **Go**: Use `testing` package for unit tests, `gopter` or `rapid` for property-based testing
- **TypeScript/React**: Use Jest + React Testing Library for unit tests, `fast-check` for property-based testing
- **Integration**: Use Wails test utilities for end-to-end testing

### Test Configuration

All property-based tests must:
- Run minimum 100 iterations per test
- Include a comment tag referencing the design property
- Use appropriate generators for test data
- Verify all aspects of the property in a single test
