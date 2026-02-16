/**
 * Property-Based Tests for Purchased Badge Visibility
 *
 * Feature: marketplace-purchased-badge, Property 4: 标记可见性与 purchased 字段一致
 *
 * For any Pack_Listing, when `purchased` is `true`, the rendered output should
 * contain the purchased badge element; when `purchased` is `false`, the rendered
 * output should NOT contain the purchased badge element. Additionally, regardless
 * of the `purchased` value, the download button or add-to-cart button should
 * always be present.
 *
 * **Validates: Requirements 3.1, 3.2, 3.5**
 *
 * Uses fast-check for property-based testing with minimum 100 iterations.
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { shouldShowPurchasedBadge, getActionButtonType } from './MarketBrowsePage';

// Arbitrary for share_mode values used in the marketplace
const shareModeArb = fc.constantFrom('free', 'per_use', 'subscription');

// Arbitrary for generating random PackListingInfo-like data
const packListingArb = fc.record({
    purchased: fc.boolean(),
    share_mode: shareModeArb,
    pack_name: fc.string({ minLength: 1, maxLength: 100 }),
    credits_price: fc.integer({ min: 0, max: 10000 }),
    download_count: fc.integer({ min: 0, max: 100000 }),
});

describe('MarketBrowsePage Purchased Badge Property-Based Tests', () => {
    // ─────────────────────────────────────────────────────────────────────────
    // Feature: marketplace-purchased-badge, Property 4: 标记可见性与 purchased 字段一致
    // ─────────────────────────────────────────────────────────────────────────
    describe('Property 4: 标记可见性与 purchased 字段一致', () => {
        /**
         * **Validates: Requirements 3.1, 3.2**
         *
         * For any pack listing, the purchased badge is visible if and only if
         * the `purchased` field is `true`.
         */
        it('badge visibility matches purchased field exactly', () => {
            fc.assert(
                fc.property(
                    packListingArb,
                    (pack) => {
                        const badgeVisible = shouldShowPurchasedBadge(pack.purchased);
                        expect(badgeVisible).toBe(pack.purchased);
                    }
                ),
                { numRuns: 200 }
            );
        });

        /**
         * **Validates: Requirements 3.1**
         *
         * When purchased is true, the badge is always visible regardless of
         * other pack properties (share_mode, price, name, etc.).
         */
        it('badge is always visible when purchased is true', () => {
            fc.assert(
                fc.property(
                    shareModeArb,
                    fc.string({ minLength: 1, maxLength: 100 }),
                    fc.integer({ min: 0, max: 10000 }),
                    (shareMode, packName, creditsPrice) => {
                        const badgeVisible = shouldShowPurchasedBadge(true);
                        expect(badgeVisible).toBe(true);
                    }
                ),
                { numRuns: 100 }
            );
        });

        /**
         * **Validates: Requirements 3.2**
         *
         * When purchased is false, the badge is never visible regardless of
         * other pack properties.
         */
        it('badge is never visible when purchased is false', () => {
            fc.assert(
                fc.property(
                    shareModeArb,
                    fc.string({ minLength: 1, maxLength: 100 }),
                    fc.integer({ min: 0, max: 10000 }),
                    (shareMode, packName, creditsPrice) => {
                        const badgeVisible = shouldShowPurchasedBadge(false);
                        expect(badgeVisible).toBe(false);
                    }
                ),
                { numRuns: 100 }
            );
        });

        /**
         * **Validates: Requirements 3.5**
         *
         * For any pack listing, an action button (download or add-to-cart) is
         * always present regardless of the purchased status.
         */
        it('action button is always present regardless of purchased status', () => {
            fc.assert(
                fc.property(
                    packListingArb,
                    (pack) => {
                        const buttonType = getActionButtonType(pack.share_mode);
                        // Button type must always be one of the two valid types
                        expect(['download', 'add_to_cart']).toContain(buttonType);
                    }
                ),
                { numRuns: 200 }
            );
        });

        /**
         * **Validates: Requirements 3.5**
         *
         * Free packs always show download button, paid packs always show
         * add-to-cart button — independent of purchased status.
         */
        it('free packs show download, paid packs show add-to-cart, independent of purchased', () => {
            fc.assert(
                fc.property(
                    fc.boolean(), // purchased
                    shareModeArb,
                    (purchased, shareMode) => {
                        const buttonType = getActionButtonType(shareMode);
                        if (shareMode === 'free') {
                            expect(buttonType).toBe('download');
                        } else {
                            expect(buttonType).toBe('add_to_cart');
                        }
                    }
                ),
                { numRuns: 100 }
            );
        });
    });
});
