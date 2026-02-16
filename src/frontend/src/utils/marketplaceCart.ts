/**
 * 购物车核心逻辑 — 纯函数，可独立测试
 */
import { main } from '../../wailsjs/go/models';

type PackListingInfo = main.PackListingInfo;

export interface CartItem {
    pack: PackListingInfo;
    quantity: number;      // 按次收费：购买次数；订阅制：月数
    isYearly: boolean;     // 订阅制：是否年订阅
}

/**
 * 计算单个购物车项的费用
 * - per_use: credits_price × quantity
 * - subscription (monthly): credits_price × quantity
 * - subscription (yearly): credits_price × 12 × quantity
 * - free: 0
 */
export function calculateItemCost(item: CartItem): number {
    if (item.pack.share_mode === 'per_use') {
        return item.pack.credits_price * item.quantity;
    }
    if (item.pack.share_mode === 'subscription') {
        if (item.isYearly) {
            return item.pack.credits_price * 12 * item.quantity;
        }
        return item.pack.credits_price * item.quantity;
    }
    return 0;
}

/**
 * 计算购物车所有项的总费用
 */
export function calculateTotalCost(items: CartItem[]): number {
    return items.reduce((sum, item) => sum + calculateItemCost(item), 0);
}
