# Dashboard Image Display Fix - Quick Reference

## Key Files

| File | Purpose |
|------|---------|
| `src/frontend/src/utils/ImageConverter.ts` | Image format detection and conversion |
| `src/frontend/src/utils/ImageConverter.test.ts` | ImageConverter unit tests |
| `src/frontend/src/components/DraggableImageComponent.tsx` | Dashboard image component |
| `src/frontend/src/components/DraggableImageComponent.test.tsx` | Component tests |
| `src/frontend/src/components/ImageModal.tsx` | Full-screen image modal |

## API Reference

### ImageConverter Functions

#### `detectImageFormat(data: string): ImageFormatDetectionResult`
Detects the format of image data.

```typescript
const result = detectImageFormat('data:image/png;base64,iVBORw0KGgo...');
// Returns: { format: 'base64_data_url', isValid: true, mimeType: 'image/png' }
```

#### `base64ToDataUrl(base64Data: string, mimeType?: string): string`
Converts base64 string to data URL.

```typescript
const dataUrl = base64ToDataUrl('iVBORw0KGgo...', 'image/png');
// Returns: 'data:image/png;base64,iVBORw0KGgo...'
```

#### `filePathToBase64(filePath: string, threadId: string, getSessionFileAsBase64: Function, timeoutMs?: number): Promise<string>`
Converts file path to base64 data URL via API.

```typescript
const dataUrl = await filePathToBase64(
  'files/image.png',
  'thread-123',
  GetSessionFileAsBase64,
  30000
);
// Returns: 'data:image/png;base64,...'
```

#### `convertImageData(imageData: string, threadId?: string, getSessionFileAsBase64?: Function): Promise<string>`
Main conversion function handling all image formats.

```typescript
// Base64 string
const result1 = await convertImageData('iVBORw0KGgo...');

// File path
const result2 = await convertImageData('files/image.png', 'thread-123', GetSessionFileAsBase64);

// HTTP URL
const result3 = await convertImageData('https://example.com/image.png');
```

#### `extractFilenameFromPath(filePath: string): string | null`
Extracts filename from file path.

```typescript
const filename = extractFilenameFromPath('files/subfolder/image.png');
// Returns: 'image.png'
```

#### `getMimeTypeFromFilename(filename: string): string`
Gets MIME type from filename.

```typescript
const mimeType = getMimeTypeFromFilename('image.jpg');
// Returns: 'image/jpeg'
```

#### `isValidBase64(str: string): boolean`
Validates if string is valid base64.

```typescript
const isValid = isValidBase64('iVBORw0KGgo...');
// Returns: true
```

#### `isValidHttpUrl(url: string): boolean`
Validates if string is valid HTTP/HTTPS URL.

```typescript
const isValid = isValidHttpUrl('https://example.com/image.png');
// Returns: true
```

#### `isValidFilePath(path: string): boolean`
Validates if string is valid file path.

```typescript
const isValid = isValidFilePath('files/image.png');
// Returns: true
```

#### `detectMimeTypeFromBase64(base64Data: string): string | null`
Detects MIME type from base64 data.

```typescript
const mimeType = detectMimeTypeFromBase64('iVBORw0KGgo...');
// Returns: 'image/png'
```

#### `clearFilePathCache(): void`
Clears the file path conversion cache.

```typescript
clearFilePathCache();
```

#### `getFilePathCacheSize(): number`
Gets the current cache size.

```typescript
const size = getFilePathCacheSize();
// Returns: 5
```

## Component Usage

### DraggableImageComponent

```typescript
<DraggableImageComponent
  instance={componentInstance}
  isEditMode={false}
  isLocked={false}
  onDragStart={handleDragStart}
  onDrag={handleDrag}
  onDragStop={handleDragStop}
  onResize={handleResize}
  onResizeStop={handleResizeStop}
  onRemove={handleRemove}
  threadId="thread-123"
/>
```

**Props**:
- `instance`: ComponentInstance - Component configuration
- `isEditMode`: boolean - Edit mode flag
- `isLocked`: boolean - Lock state
- `onDragStart`: (id: string) => void - Drag start handler
- `onDrag`: (id: string, x: number, y: number) => void - Drag handler
- `onDragStop`: (id: string, x: number, y: number) => void - Drag stop handler
- `onResize`: (id: string, width: number, height: number) => void - Resize handler
- `onResizeStop`: (id: string, width: number, height: number) => void - Resize stop handler
- `onRemove`: (id: string) => void - Remove handler (optional)
- `threadId`: string - Session/thread ID (optional)

### ImageModal

```typescript
<ImageModal
  isOpen={true}
  imageUrl="data:image/png;base64,..."
  onClose={handleClose}
/>
```

**Props**:
- `isOpen`: boolean - Modal visibility
- `imageUrl`: string - Image URL to display
- `onClose`: () => void - Close handler

## Supported Image Formats

| Format | Example | Handling |
|--------|---------|----------|
| Base64 Data URL | `data:image/png;base64,iVBORw0KGgo...` | Pass-through |
| Base64 String | `iVBORw0KGgo...` | Convert to data URL |
| HTTP URL | `http://example.com/image.png` | Pass-through |
| HTTPS URL | `https://example.com/image.png` | Pass-through |
| File Path | `files/image.png` | API call → base64 → data URL |
| File URL | `file:///path/to/image.png` | Extract filename → API call |
| Relative Path | `./images/image.png` | Extract filename → API call |

## Error Handling

### Common Errors

| Error | Cause | Solution |
|-------|-------|----------|
| "Invalid base64 string" | Malformed base64 data | Verify base64 encoding |
| "Invalid file path format" | Unsupported path format | Use supported path format |
| "Thread ID required" | Missing threadId for file path | Provide threadId |
| "API call timeout" | API response too slow | Check network/API |
| "File not found" | File doesn't exist | Verify file path |
| "Unknown image format" | Unsupported format | Use supported format |

### Error Handling Pattern

```typescript
try {
  const imageUrl = await convertImageData(imageData, threadId, GetSessionFileAsBase64);
  setImageSrc(imageUrl);
} catch (error) {
  console.error('Image conversion failed:', error);
  setError(true);
  // Display error state to user
}
```

## Testing

### Running Tests

```bash
# Run all ImageConverter tests
npm test -- ImageConverter.test.ts

# Run DraggableImageComponent tests
npm test -- DraggableImageComponent.test.ts

# Run with coverage
npm test -- --coverage
```

### Writing Tests

```typescript
import { convertImageData, clearFilePathCache } from '../utils/ImageConverter';

describe('Image Conversion', () => {
  beforeEach(() => {
    clearFilePathCache();
  });

  it('should convert base64 to data URL', async () => {
    const result = await convertImageData('iVBORw0KGgo...');
    expect(result).toMatch(/^data:image\/png;base64,/);
  });
});
```

## Performance Tips

1. **Cache Management**: File path conversions are automatically cached
2. **Lazy Loading**: Images load only when component mounts
3. **Error Handling**: Errors don't block UI rendering
4. **Timeout**: API calls timeout after 30 seconds

## Debugging

### Enable Logging

```typescript
// ImageConverter logs with [CHART] prefix
console.log('[CHART] Detected inline base64_data_url image: data:image/png...');
console.warn('[CHART] Failed to detect image format: Unknown image format...');
console.error('[CHART] Image conversion failed: Invalid base64 string');
```

### Check Cache

```typescript
import { getFilePathCacheSize } from '../utils/ImageConverter';

console.log('Cache size:', getFilePathCacheSize());
```

### Verify Image Data

```typescript
import { detectImageFormat } from '../utils/ImageConverter';

const result = detectImageFormat(imageData);
console.log('Format:', result.format);
console.log('Valid:', result.isValid);
console.log('Error:', result.error);
```

## Common Patterns

### Display Image from Analysis Response

```typescript
// In App.tsx event handler
EventsOn("dashboard-update", (payload: any) => {
  if (payload.type === 'image') {
    setActiveChart({
      type: 'image',
      data: payload.data  // Can be any supported format
    });
  }
});
```

### Handle File Upload

```typescript
// Convert file to base64
const reader = new FileReader();
reader.onload = (e) => {
  const base64 = e.target?.result as string;
  setImageData(base64);
};
reader.readAsDataURL(file);
```

### Display HTTP Image

```typescript
// HTTP URLs pass through directly
const imageUrl = 'https://example.com/image.png';
setImageData(imageUrl);
```

### Load File from Session

```typescript
// File paths are converted via API
const imageUrl = 'files/analysis-result.png';
const converted = await convertImageData(imageUrl, threadId, GetSessionFileAsBase64);
setImageData(converted);
```

## Troubleshooting Checklist

- [ ] Image source is in supported format
- [ ] ThreadId is provided for file paths
- [ ] File exists if using file path
- [ ] Network connection is working
- [ ] API endpoint is responding
- [ ] Session ID matches current session
- [ ] Browser console shows no errors
- [ ] Image MIME type is correct
- [ ] Base64 data is valid
- [ ] URL is properly formatted

## Related Documentation

- [Requirements](./requirements.md)
- [Design](./design.md)
- [Implementation Summary](./IMPLEMENTATION_SUMMARY.md)
