/**
 * FilePreviewRenderer
 * 
 * 将后端返回的文件预览数据（JSON）渲染为 Canvas 图片（data URL）。
 * 支持表格预览（Excel/CSV）和幻灯片预览（PPT）。
 */

export interface TablePreviewData {
  type: 'table';
  title: string;
  headers: string[];
  rows: string[][];
  totalRows: number;
  totalCols: number;
}

export interface SlidePreviewItem {
  title: string;
  texts: string[];
}

export interface SlidesPreviewData {
  type: 'slides';
  title: string;
  slides: SlidePreviewItem[];
}

export type FilePreviewData = TablePreviewData | SlidesPreviewData;

const PREVIEW_WIDTH = 320;
const PREVIEW_HEIGHT = 200;
const DPR = 2; // High DPI rendering

/**
 * Render file preview data to a data URL image
 */
export function renderFilePreview(jsonStr: string): string | null {
  try {
    const data: FilePreviewData = JSON.parse(jsonStr);
    if (data.type === 'table') {
      return renderTablePreview(data as TablePreviewData);
    } else if (data.type === 'slides') {
      return renderSlidesPreview(data as SlidesPreviewData);
    }
    return null;
  } catch {
    return null;
  }
}

/**
 * Render a table preview (Excel/CSV) to canvas data URL
 */
function renderTablePreview(data: TablePreviewData): string | null {
  const canvas = document.createElement('canvas');
  canvas.width = PREVIEW_WIDTH * DPR;
  canvas.height = PREVIEW_HEIGHT * DPR;
  const ctx = canvas.getContext('2d');
  if (!ctx) return null;

  ctx.scale(DPR, DPR);

  // Background
  ctx.fillStyle = '#ffffff';
  ctx.fillRect(0, 0, PREVIEW_WIDTH, PREVIEW_HEIGHT);

  const cols = data.headers.length;
  if (cols === 0) return null;

  const headerH = 24;
  const rowH = 20;
  const padX = 8;
  const padY = 6;
  const colW = Math.floor((PREVIEW_WIDTH - padX * 2) / cols);

  // Header background
  ctx.fillStyle = '#3b82f6';
  ctx.fillRect(padX, padY, PREVIEW_WIDTH - padX * 2, headerH);

  // Header text
  ctx.fillStyle = '#ffffff';
  ctx.font = 'bold 10px system-ui, -apple-system, sans-serif';
  ctx.textBaseline = 'middle';
  for (let i = 0; i < cols; i++) {
    const x = padX + i * colW + 4;
    const text = truncateText(ctx, data.headers[i] || '', colW - 8);
    ctx.fillText(text, x, padY + headerH / 2);
  }

  // Data rows
  const rows = data.rows || [];
  ctx.font = '9px system-ui, -apple-system, sans-serif';
  for (let r = 0; r < rows.length; r++) {
    const y = padY + headerH + r * rowH;
    if (y + rowH > PREVIEW_HEIGHT - 20) break;

    // Alternating row background
    ctx.fillStyle = r % 2 === 0 ? '#f8fafc' : '#f1f5f9';
    ctx.fillRect(padX, y, PREVIEW_WIDTH - padX * 2, rowH);

    // Row text
    ctx.fillStyle = '#334155';
    for (let c = 0; c < cols; c++) {
      const x = padX + c * colW + 4;
      const val = (rows[r] && rows[r][c]) || '';
      const text = truncateText(ctx, val, colW - 8);
      ctx.fillText(text, x, y + rowH / 2);
    }

    // Column separators
    ctx.strokeStyle = '#e2e8f0';
    ctx.lineWidth = 0.5;
    for (let c = 1; c < cols; c++) {
      const x = padX + c * colW;
      ctx.beginPath();
      ctx.moveTo(x, y);
      ctx.lineTo(x, y + rowH);
      ctx.stroke();
    }
  }

  // Footer with total info
  const footerY = PREVIEW_HEIGHT - 16;
  ctx.fillStyle = '#94a3b8';
  ctx.font = '8px system-ui, -apple-system, sans-serif';
  ctx.textBaseline = 'bottom';
  const info = `${data.totalRows} 行 × ${data.totalCols} 列`;
  ctx.fillText(info, padX, footerY + 10);

  // Border
  ctx.strokeStyle = '#e2e8f0';
  ctx.lineWidth = 1;
  ctx.strokeRect(0.5, 0.5, PREVIEW_WIDTH - 1, PREVIEW_HEIGHT - 1);

  return canvas.toDataURL('image/png');
}

/**
 * Render a slides preview (PPT) to canvas data URL
 */
function renderSlidesPreview(data: SlidesPreviewData): string | null {
  const canvas = document.createElement('canvas');
  canvas.width = PREVIEW_WIDTH * DPR;
  canvas.height = PREVIEW_HEIGHT * DPR;
  const ctx = canvas.getContext('2d');
  if (!ctx) return null;

  ctx.scale(DPR, DPR);

  // Background - slide-like gradient
  ctx.fillStyle = '#f0f4ff';
  ctx.fillRect(0, 0, PREVIEW_WIDTH, PREVIEW_HEIGHT);

  // Top accent bar
  ctx.fillStyle = '#3b82f6';
  ctx.fillRect(0, 0, PREVIEW_WIDTH, 4);

  const slides = data.slides || [];
  if (slides.length === 0) {
    // Empty PPT
    ctx.fillStyle = '#94a3b8';
    ctx.font = '12px system-ui, -apple-system, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'middle';
    ctx.fillText('空演示文稿', PREVIEW_WIDTH / 2, PREVIEW_HEIGHT / 2);
    return canvas.toDataURL('image/png');
  }

  // Render first slide prominently
  const firstSlide = slides[0];
  const padX = 16;
  let y = 20;

  // Title
  if (firstSlide.title) {
    ctx.fillStyle = '#1e40af';
    ctx.font = 'bold 14px system-ui, -apple-system, sans-serif';
    ctx.textAlign = 'center';
    ctx.textBaseline = 'top';
    const title = truncateText(ctx, firstSlide.title, PREVIEW_WIDTH - padX * 2);
    ctx.fillText(title, PREVIEW_WIDTH / 2, y);
    y += 24;
  }

  // Subtitle / body texts
  ctx.textAlign = 'left';
  ctx.font = '10px system-ui, -apple-system, sans-serif';
  ctx.fillStyle = '#475569';
  const maxTexts = 5;
  const texts = firstSlide.texts || [];
  for (let i = 0; i < Math.min(texts.length, maxTexts); i++) {
    if (y > PREVIEW_HEIGHT - 40) break;
    const text = truncateText(ctx, texts[i], PREVIEW_WIDTH - padX * 2 - 10);
    ctx.fillText('• ' + text, padX + 4, y);
    y += 16;
  }

  // Show slide count indicator at bottom
  if (slides.length > 1) {
    const indicatorY = PREVIEW_HEIGHT - 20;
    ctx.fillStyle = '#e2e8f0';
    ctx.fillRect(0, indicatorY - 4, PREVIEW_WIDTH, 24);

    // Slide dots
    const dotSize = 6;
    const dotGap = 10;
    const totalDotsWidth = slides.length * dotSize + (slides.length - 1) * (dotGap - dotSize);
    let dotX = (PREVIEW_WIDTH - totalDotsWidth) / 2;
    for (let i = 0; i < Math.min(slides.length, 8); i++) {
      ctx.beginPath();
      ctx.arc(dotX + dotSize / 2, indicatorY + 6, dotSize / 2, 0, Math.PI * 2);
      ctx.fillStyle = i === 0 ? '#3b82f6' : '#cbd5e1';
      ctx.fill();
      dotX += dotGap;
    }
  }

  // Bottom accent bar
  ctx.fillStyle = '#3b82f6';
  ctx.fillRect(0, PREVIEW_HEIGHT - 3, PREVIEW_WIDTH, 3);

  // Border
  ctx.strokeStyle = '#e2e8f0';
  ctx.lineWidth = 1;
  ctx.strokeRect(0.5, 0.5, PREVIEW_WIDTH - 1, PREVIEW_HEIGHT - 1);

  return canvas.toDataURL('image/png');
}

/**
 * Truncate text to fit within maxWidth pixels
 */
function truncateText(ctx: CanvasRenderingContext2D, text: string, maxWidth: number): string {
  if (ctx.measureText(text).width <= maxWidth) return text;
  let truncated = text;
  while (truncated.length > 0 && ctx.measureText(truncated + '..').width > maxWidth) {
    truncated = truncated.slice(0, -1);
  }
  return truncated + '..';
}
