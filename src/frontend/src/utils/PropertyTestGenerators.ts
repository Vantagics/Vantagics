import fc from 'fast-check';
import { ComponentType } from '../components/PaginationControl';
import { LayoutItem } from './LayoutEngine';
import { ComponentInstance } from './ComponentManager';

/**
 * Property-based test generators for dashboard drag-drop layout
 */

// Basic generators
export const genComponentType = fc.constantFrom(
  ComponentType.METRICS,
  ComponentType.TABLE,
  ComponentType.IMAGE,
  ComponentType.INSIGHTS,
  ComponentType.FILE_DOWNLOAD
);

export const genGridPosition = fc.record({
  x: fc.integer({ min: 0, max: 23 }), // 24-column grid (0-23)
  y: fc.integer({ min: 0, max: 100 }) // Reasonable Y range
});

export const genGridDimensions = fc.record({
  w: fc.integer({ min: 1, max: 24 }), // Width in grid units
  h: fc.integer({ min: 1, max: 20 })  // Height in grid units
});

export const genLayoutItem = fc.record({
  i: fc.string({ minLength: 1, maxLength: 50 }),
  x: fc.integer({ min: 0, max: 23 }),
  y: fc.integer({ min: 0, max: 100 }),
  w: fc.integer({ min: 1, max: 24 }),
  h: fc.integer({ min: 1, max: 20 }),
  minW: fc.option(fc.integer({ min: 1, max: 12 }), { nil: undefined }),
  minH: fc.option(fc.integer({ min: 1, max: 10 }), { nil: undefined }),
  maxW: fc.option(fc.integer({ min: 12, max: 24 }), { nil: undefined }),
  maxH: fc.option(fc.integer({ min: 10, max: 20 }), { nil: undefined }),
  static: fc.boolean(),
  type: genComponentType,
  instanceIdx: fc.integer({ min: 0, max: 10 })
}).map((item): LayoutItem => ({
  ...item,
  // Ensure constraints are valid
  minW: item.minW && item.minW <= item.w ? item.minW : undefined,
  minH: item.minH && item.minH <= item.h ? item.minH : undefined,
  maxW: item.maxW && item.maxW >= item.w ? item.maxW : undefined,
  maxH: item.maxH && item.maxH >= item.h ? item.maxH : undefined
}));

export const genLayoutConfiguration = fc.record({
  id: fc.uuid(),
  userId: fc.uuid(),
  isLocked: fc.boolean(),
  items: fc.array(genLayoutItem, { minLength: 1, maxLength: 20 }),
  createdAt: fc.date(),
  updatedAt: fc.date()
});

// Component data generators
export const genMetricData = fc.record({
  title: fc.string({ minLength: 1, maxLength: 100 }),
  value: fc.string({ minLength: 1, maxLength: 50 }),
  change: fc.oneof(
    fc.string().map(s => `+${Math.abs(parseFloat(s) || 0).toFixed(1)}%`),
    fc.string().map(s => `-${Math.abs(parseFloat(s) || 0).toFixed(1)}%`)
  )
});

export const genTableData = fc.record({
  data: fc.array(
    fc.record({
      id: fc.integer(),
      name: fc.string(),
      value: fc.oneof(fc.string(), fc.integer(), fc.constant(null))
    }),
    { minLength: 0, maxLength: 100 }
  ),
  title: fc.option(fc.string(), { nil: undefined })
});

export const genImageData = fc.record({
  src: fc.oneof(
    fc.constant('data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8/5+hHgAHggJ/PchI7wAAAABJRU5ErkJggg=='),
    fc.string().map(s => `files/${s}.png`),
    fc.string().map(s => `https://example.com/${s}.jpg`)
  ),
  alt: fc.option(fc.string(), { nil: undefined }),
  title: fc.option(fc.string(), { nil: undefined })
});

export const genInsightData = fc.record({
  text: fc.string({ minLength: 1, maxLength: 500 }),
  icon: fc.constantFrom('trending-up', 'user-check', 'alert-circle', 'star', 'info'),
  title: fc.option(fc.string(), { nil: undefined })
});

export const genFileInfo = fc.record({
  id: fc.uuid(),
  name: fc.string({ minLength: 1, maxLength: 100 }).map(s => `${s}.pdf`),
  size: fc.integer({ min: 0, max: 10000000 }), // Up to 10MB
  createdAt: fc.date(),
  downloadUrl: fc.option(fc.string(), { nil: undefined })
});

export const genFileDownloadData = fc.record({
  allFiles: fc.array(genFileInfo, { minLength: 0, maxLength: 50 }),
  userRequestRelatedFiles: fc.array(genFileInfo, { minLength: 0, maxLength: 20 }),
  title: fc.option(fc.string(), { nil: undefined })
});

export const genComponentData = (type: ComponentType) => {
  switch (type) {
    case ComponentType.METRICS:
      return genMetricData;
    case ComponentType.TABLE:
      return genTableData;
    case ComponentType.IMAGE:
      return genImageData;
    case ComponentType.INSIGHTS:
      return genInsightData;
    case ComponentType.FILE_DOWNLOAD:
      return genFileDownloadData;
    default:
      return fc.constant(null);
  }
};

export const genComponentInstance = fc.record({
  id: fc.string({ minLength: 1, maxLength: 50 }),
  type: genComponentType,
  instanceIndex: fc.integer({ min: 0, max: 10 }),
  hasData: fc.boolean(),
  layoutItem: genLayoutItem
}).chain(base => 
  genComponentData(base.type).map(data => ({
    ...base,
    data: base.hasData ? data : null
  } as ComponentInstance))
);

// Drag operation generators
export const genDragOperation = fc.record({
  componentId: fc.string({ minLength: 1, maxLength: 50 }),
  startPosition: genGridPosition,
  endPosition: genGridPosition,
  isValid: fc.boolean()
});

export const genResizeOperation = fc.record({
  componentId: fc.string({ minLength: 1, maxLength: 50 }),
  startDimensions: genGridDimensions,
  endDimensions: genGridDimensions,
  isValid: fc.boolean()
});

// Mode state generators
export const genModeState = fc.record({
  isEditMode: fc.boolean(),
  isLocked: fc.boolean()
});

// Pagination generators
export const genPaginationState = fc.record({
  currentPage: fc.integer({ min: 0, max: 10 }),
  totalPages: fc.integer({ min: 1, max: 11 }),
  instancesPerPage: fc.integer({ min: 1, max: 5 })
}).filter(state => state.currentPage < state.totalPages);

// Collision detection generators
export const genOverlappingItems = fc.tuple(genLayoutItem, genLayoutItem)
  .filter(([item1, item2]) => {
    // Generate items that actually overlap
    const overlap = !(
      item1.x + item1.w <= item2.x ||
      item2.x + item2.w <= item1.x ||
      item1.y + item1.h <= item2.y ||
      item2.y + item2.h <= item1.y
    );
    return overlap && item1.i !== item2.i;
  });

export const genNonOverlappingItems = fc.tuple(genLayoutItem, genLayoutItem)
  .filter(([item1, item2]) => {
    // Generate items that don't overlap
    const overlap = !(
      item1.x + item1.w <= item2.x ||
      item2.x + item2.w <= item1.x ||
      item1.y + item1.h <= item2.y ||
      item2.y + item2.h <= item1.y
    );
    return !overlap && item1.i !== item2.i;
  });

// Viewport size generators for responsive testing
export const genViewportSize = fc.record({
  width: fc.integer({ min: 320, max: 3840 }), // Mobile to 4K
  height: fc.integer({ min: 240, max: 2160 })
});

// Grid configuration generators
export const genGridConfig = fc.record({
  columns: fc.constantFrom(12, 24), // Common grid systems
  rowHeight: fc.integer({ min: 20, max: 100 }),
  margin: fc.tuple(
    fc.integer({ min: 0, max: 20 }),
    fc.integer({ min: 0, max: 20 })
  ),
  containerPadding: fc.tuple(
    fc.integer({ min: 0, max: 40 }),
    fc.integer({ min: 0, max: 40 })
  )
});

// Utility generators
export const genValidPosition = (maxX: number = 23, maxY: number = 100) =>
  fc.record({
    x: fc.integer({ min: 0, max: maxX }),
    y: fc.integer({ min: 0, max: maxY })
  });

export const genInvalidPosition = fc.oneof(
  fc.record({
    x: fc.integer({ min: -100, max: -1 }), // Negative X
    y: fc.integer({ min: 0, max: 100 })
  }),
  fc.record({
    x: fc.integer({ min: 24, max: 100 }), // X beyond grid
    y: fc.integer({ min: 0, max: 100 })
  }),
  fc.record({
    x: fc.integer({ min: 0, max: 23 }),
    y: fc.integer({ min: -100, max: -1 }) // Negative Y
  })
);

export const genValidDimensions = (maxW: number = 24, maxH: number = 20) =>
  fc.record({
    w: fc.integer({ min: 1, max: maxW }),
    h: fc.integer({ min: 1, max: maxH })
  });

export const genInvalidDimensions = fc.oneof(
  fc.record({
    w: fc.integer({ min: -10, max: 0 }), // Invalid width
    h: fc.integer({ min: 1, max: 20 })
  }),
  fc.record({
    w: fc.integer({ min: 1, max: 24 }),
    h: fc.integer({ min: -10, max: 0 }) // Invalid height
  }),
  fc.record({
    w: fc.integer({ min: 25, max: 100 }), // Width beyond grid
    h: fc.integer({ min: 1, max: 20 })
  })
);

// Complex scenario generators
export const genLayoutWithCollisions = fc.array(genLayoutItem, { minLength: 2, maxLength: 10 })
  .map(items => {
    // Force some items to overlap by modifying positions
    if (items.length >= 2) {
      items[1] = { ...items[1], x: items[0].x, y: items[0].y };
    }
    return items;
  });

export const genLayoutWithoutCollisions = fc.array(genLayoutItem, { minLength: 1, maxLength: 10 })
  .map(items => {
    // Ensure no overlaps by spacing items out
    return items.map((item, index) => ({
      ...item,
      x: (index * 6) % 24, // Space items 6 units apart horizontally
      y: Math.floor((index * 6) / 24) * 8 // Move to next row when needed
    }));
  });

export default {
  genComponentType,
  genGridPosition,
  genGridDimensions,
  genLayoutItem,
  genLayoutConfiguration,
  genMetricData,
  genTableData,
  genImageData,
  genInsightData,
  genFileDownloadData,
  genComponentData,
  genComponentInstance,
  genDragOperation,
  genResizeOperation,
  genModeState,
  genPaginationState,
  genOverlappingItems,
  genNonOverlappingItems,
  genViewportSize,
  genGridConfig,
  genValidPosition,
  genInvalidPosition,
  genValidDimensions,
  genInvalidDimensions,
  genLayoutWithCollisions,
  genLayoutWithoutCollisions
};