/**
 * 市场浏览筛选/排序/搜索 — 纯函数，可独立测试
 */
import { main } from '../../wailsjs/go/models';

type PackListingInfo = main.PackListingInfo;

export type ShareModeFilter = 'all' | 'free' | 'per_use' | 'subscription';
export type SortField = 'created_at' | 'credits_price';
export type SortDirection = 'desc' | 'asc';

/**
 * 按付费类型筛选分析包
 * - 'all' 返回全部
 * - 其他值按 share_mode 精确匹配
 */
export function filterByShareMode(packs: PackListingInfo[], filter: ShareModeFilter): PackListingInfo[] {
    if (filter === 'all') {
        return packs;
    }
    return packs.filter(p => p.share_mode === filter);
}

/**
 * 按关键字搜索分析包（不区分大小写）
 * - 空/纯空白关键字返回全部
 * - 否则匹配 pack_name 或 pack_description
 */
export function filterByKeyword(packs: PackListingInfo[], keyword: string): PackListingInfo[] {
    const trimmed = keyword.trim();
    if (!trimmed) {
        return packs;
    }
    const kw = trimmed.toLowerCase();
    return packs.filter(p =>
        p.pack_name.toLowerCase().includes(kw) ||
        (p.pack_description && p.pack_description.toLowerCase().includes(kw))
    );
}

/**
 * 排序分析包（不修改原数组）
 * - created_at: 字符串比较
 * - credits_price: 数值比较
 * - desc = 降序, asc = 升序
 */
export function sortPacks(packs: PackListingInfo[], field: SortField, direction: SortDirection): PackListingInfo[] {
    const sorted = [...packs];
    sorted.sort((a, b) => {
        let cmp: number;
        if (field === 'created_at') {
            cmp = a.created_at.localeCompare(b.created_at);
        } else {
            cmp = a.credits_price - b.credits_price;
        }
        return direction === 'desc' ? -cmp : cmp;
    });
    return sorted;
}
