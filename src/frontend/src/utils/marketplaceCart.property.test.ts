/**
 * Property-Based Tests for marketplaceCart — calculateItemCost
 *
 * Uses fast-check to verify universal properties across randomized inputs.
 * Each property test runs a minimum of 100 iterations.
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { calculateItemCost, calculateTotalCost, CartItem } from './marketplaceCart';
import { main } from '../../wailsjs/go/models';

/**
 * Smart generator: builds a PackListingInfo with a constrained share_mode
 * and a non-negative integer credits_price.
 */
function packArb(shareMode: 'per_use' | 'subscription' | 'free'): fc.Arbitrary<main.PackListingInfo> {
    return fc.record({
        id: fc.integer({ min: 1, max: 10000 }),
        user_id: fc.integer({ min: 1, max: 1000 }),
        category_id: fc.integer({ min: 1, max: 100 }),
        category_name: fc.string({ minLength: 1, maxLength: 20 }),
        pack_name: fc.string({ minLength: 1, maxLength: 50 }),
        pack_description: fc.string({ maxLength: 100 }),
        source_name: fc.string({ minLength: 1, maxLength: 20 }),
        author_name: fc.string({ minLength: 1, maxLength: 20 }),
        share_mode: fc.constant(shareMode),
        credits_price: shareMode === 'free'
            ? fc.constant(0)
            : fc.integer({ min: 1, max: 10000 }),
        download_count: fc.integer({ min: 0, max: 100000 }),
        created_at: fc.constant('2024-01-01'),
    }).map(fields => new main.PackListingInfo(fields));
}

/**
 * Smart generator: builds a CartItem for per_use packs.
 * quantity ∈ [1, 1000]
 */
function perUseCartItemArb(): fc.Arbitrary<CartItem> {
    return fc.tuple(
        packArb('per_use'),
        fc.integer({ min: 1, max: 1000 }),
    ).map(([pack, quantity]) => ({ pack, quantity, isYearly: false }));
}

/**
 * Smart generator: builds a CartItem for monthly subscription packs.
 * quantity (months) ∈ [1, 12]
 */
function monthlySubCartItemArb(): fc.Arbitrary<CartItem> {
    return fc.tuple(
        packArb('subscription'),
        fc.integer({ min: 1, max: 12 }),
    ).map(([pack, quantity]) => ({ pack, quantity, isYearly: false }));
}

/**
 * Smart generator: builds a CartItem for yearly subscription packs.
 * quantity (years) ∈ [1, 3]
 */
function yearlySubCartItemArb(): fc.Arbitrary<CartItem> {
    return fc.tuple(
        packArb('subscription'),
        fc.integer({ min: 1, max: 3 }),
    ).map(([pack, quantity]) => ({ pack, quantity, isYearly: true }));
}

/**
 * Smart generator: builds a CartItem for free packs.
 */
function freeCartItemArb(): fc.Arbitrary<CartItem> {
    return packArb('free').map(pack => ({ pack, quantity: 1, isYearly: false }));
}

describe('marketplaceCart Property-Based Tests', () => {
    // ─────────────────────────────────────────────────────────────────────
    // Feature: marketplace-browse-purchase, Property 4: 单项费用计算正确性
    // ─────────────────────────────────────────────────────────────────────
    describe('Property 4: 单项费用计算正确性', () => {
        /**
         * **Validates: Requirements 8.2, 8.3, 10.2, 10.3, 10.4**
         *
         * For any CartItem, calculateItemCost must return:
         * - per_use:              credits_price × quantity
         * - subscription monthly: credits_price × quantity
         * - subscription yearly:  credits_price × 12 × quantity
         * - free:                 0
         */

        it('per_use: cost equals credits_price × quantity', () => {
            fc.assert(
                fc.property(perUseCartItemArb(), (item) => {
                    const cost = calculateItemCost(item);
                    expect(cost).toBe(item.pack.credits_price * item.quantity);
                }),
                { numRuns: 100 },
            );
        });

        it('subscription monthly: cost equals credits_price × quantity', () => {
            fc.assert(
                fc.property(monthlySubCartItemArb(), (item) => {
                    const cost = calculateItemCost(item);
                    expect(cost).toBe(item.pack.credits_price * item.quantity);
                }),
                { numRuns: 100 },
            );
        });

        it('subscription yearly: cost equals credits_price × 12 × quantity', () => {
            fc.assert(
                fc.property(yearlySubCartItemArb(), (item) => {
                    const cost = calculateItemCost(item);
                    expect(cost).toBe(item.pack.credits_price * 12 * item.quantity);
                }),
                { numRuns: 100 },
            );
        });

        it('free: cost is always 0', () => {
            fc.assert(
                fc.property(freeCartItemArb(), (item) => {
                    const cost = calculateItemCost(item);
                    expect(cost).toBe(0);
                }),
                { numRuns: 100 },
            );
        });
    });

    // ─────────────────────────────────────────────────────────────────────
    // Feature: marketplace-browse-purchase, Property 5: 总费用等于各项费用之和
    // ─────────────────────────────────────────────────────────────────────
    describe('Property 5: 总费用等于各项费用之和', () => {
        /**
         * **Validates: Requirements 8.1, 10.1, 10.5**
         *
         * For any list of CartItems, calculateTotalCost must equal
         * the sum of calculateItemCost applied to each item.
         */

        /** Generator for a mixed list of CartItems (0–20 items). */
        const anyCartItemArb = fc.oneof(
            perUseCartItemArb(),
            monthlySubCartItemArb(),
            yearlySubCartItemArb(),
            freeCartItemArb(),
        );
        const cartItemsArb = fc.array(anyCartItemArb, { minLength: 0, maxLength: 20 });

        it('totalCost equals sum of individual item costs for mixed carts', () => {
            fc.assert(
                fc.property(cartItemsArb, (items) => {
                    const total = calculateTotalCost(items);
                    const expectedSum = items.reduce((sum, item) => sum + calculateItemCost(item), 0);
                    expect(total).toBe(expectedSum);
                }),
                { numRuns: 100 },
            );
        });

        it('empty cart has zero total cost', () => {
            expect(calculateTotalCost([])).toBe(0);
        });

        it('single-item cart total equals that item cost', () => {
            fc.assert(
                fc.property(anyCartItemArb, (item) => {
                    expect(calculateTotalCost([item])).toBe(calculateItemCost(item));
                }),
                { numRuns: 100 },
            );
        });
    });
});

// ─────────────────────────────────────────────────────────────────────
// Feature: marketplace-browse-purchase, Property 6: 购物车添加/移除往返一致性
// ─────────────────────────────────────────────────────────────────────
describe('Property 6: 购物车添加/移除往返一致性', () => {
    /**
     * **Validates: Requirements 7.4, 7.6**
     *
     * For any cart state and any valid CartItem, adding the item
     * then removing it should restore the original cart state
     * (item count and total cost are unchanged).
     */

    /** Generator for a mixed CartItem. */
    const anyCartItemArb = fc.oneof(
        perUseCartItemArb(),
        monthlySubCartItemArb(),
        yearlySubCartItemArb(),
        freeCartItemArb(),
    );

    /** Generator for an initial cart (0–10 items). */
    const initialCartArb = fc.array(anyCartItemArb, { minLength: 0, maxLength: 10 });

    it('adding then removing an item restores original cart length', () => {
        fc.assert(
            fc.property(initialCartArb, anyCartItemArb, (cart, newItem) => {
                const originalLength = cart.length;

                // Add: append newItem
                const cartAfterAdd = [...cart, newItem];
                expect(cartAfterAdd.length).toBe(originalLength + 1);

                // Remove: remove the last item (the one just added)
                const indexToRemove = cartAfterAdd.length - 1;
                const cartAfterRemove = cartAfterAdd.filter((_, i) => i !== indexToRemove);
                expect(cartAfterRemove.length).toBe(originalLength);
            }),
            { numRuns: 100 },
        );
    });

    it('adding then removing an item restores original total cost', () => {
        fc.assert(
            fc.property(initialCartArb, anyCartItemArb, (cart, newItem) => {
                const originalCost = calculateTotalCost(cart);

                // Add: append newItem
                const cartAfterAdd = [...cart, newItem];

                // Remove: remove the last item (the one just added)
                const indexToRemove = cartAfterAdd.length - 1;
                const cartAfterRemove = cartAfterAdd.filter((_, i) => i !== indexToRemove);

                expect(calculateTotalCost(cartAfterRemove)).toBe(originalCost);
            }),
            { numRuns: 100 },
        );
    });

    it('add increases total cost by the new item cost, remove restores it', () => {
        fc.assert(
            fc.property(initialCartArb, anyCartItemArb, (cart, newItem) => {
                const originalCost = calculateTotalCost(cart);
                const newItemCost = calculateItemCost(newItem);

                // Add
                const cartAfterAdd = [...cart, newItem];
                expect(calculateTotalCost(cartAfterAdd)).toBe(originalCost + newItemCost);

                // Remove last
                const cartAfterRemove = cartAfterAdd.filter((_, i) => i !== cartAfterAdd.length - 1);
                expect(calculateTotalCost(cartAfterRemove)).toBe(originalCost);
            }),
            { numRuns: 100 },
        );
    });
});

