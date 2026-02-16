import { describe, it, expect } from 'vitest';
import { calculateItemCost, calculateTotalCost, CartItem } from './marketplaceCart';
import { main } from '../../wailsjs/go/models';

function makePack(overrides: Partial<main.PackListingInfo> = {}): main.PackListingInfo {
    return new main.PackListingInfo({
        id: 1,
        user_id: 1,
        category_id: 1,
        category_name: 'Test',
        pack_name: 'Test Pack',
        pack_description: 'desc',
        source_name: 'src',
        author_name: 'author',
        share_mode: 'per_use',
        credits_price: 10,
        download_count: 0,
        created_at: '2024-01-01',
        ...overrides,
    });
}

describe('calculateItemCost', () => {
    it('returns price × quantity for per_use', () => {
        const item: CartItem = { pack: makePack({ share_mode: 'per_use', credits_price: 5 }), quantity: 3, isYearly: false };
        expect(calculateItemCost(item)).toBe(15);
    });

    it('returns price × quantity for monthly subscription', () => {
        const item: CartItem = { pack: makePack({ share_mode: 'subscription', credits_price: 10 }), quantity: 6, isYearly: false };
        expect(calculateItemCost(item)).toBe(60);
    });

    it('returns price × 12 × quantity for yearly subscription', () => {
        const item: CartItem = { pack: makePack({ share_mode: 'subscription', credits_price: 10 }), quantity: 2, isYearly: true };
        expect(calculateItemCost(item)).toBe(240);
    });

    it('returns 0 for free packs', () => {
        const item: CartItem = { pack: makePack({ share_mode: 'free', credits_price: 0 }), quantity: 1, isYearly: false };
        expect(calculateItemCost(item)).toBe(0);
    });
});

describe('calculateTotalCost', () => {
    it('returns 0 for empty cart', () => {
        expect(calculateTotalCost([])).toBe(0);
    });

    it('sums costs of all items', () => {
        const items: CartItem[] = [
            { pack: makePack({ share_mode: 'per_use', credits_price: 5 }), quantity: 2, isYearly: false },
            { pack: makePack({ share_mode: 'subscription', credits_price: 10 }), quantity: 3, isYearly: false },
        ];
        // 5*2 + 10*3 = 40
        expect(calculateTotalCost(items)).toBe(40);
    });
});
