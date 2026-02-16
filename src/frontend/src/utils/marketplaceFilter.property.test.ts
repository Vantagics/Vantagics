/**
 * Property-Based Tests for marketplaceFilter — filterByShareMode
 *
 * Uses fast-check to verify universal properties across randomized inputs.
 * Each property test runs a minimum of 100 iterations.
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { filterByShareMode, filterByKeyword, sortPacks, ShareModeFilter, SortField, SortDirection } from './marketplaceFilter';
import { main } from '../../wailsjs/go/models';

/** Valid share_mode values for pack listings. */
const SHARE_MODES = ['free', 'per_use', 'subscription'] as const;

/** All filter options including 'all'. */
const FILTER_OPTIONS: ShareModeFilter[] = ['all', 'free', 'per_use', 'subscription'];

/**
 * Smart generator: builds a PackListingInfo with a random share_mode
 * drawn from the valid set.
 */
function packArb(): fc.Arbitrary<main.PackListingInfo> {
    return fc.record({
        id: fc.integer({ min: 1, max: 10000 }),
        user_id: fc.integer({ min: 1, max: 1000 }),
        category_id: fc.integer({ min: 1, max: 100 }),
        category_name: fc.string({ minLength: 1, maxLength: 20 }),
        pack_name: fc.string({ minLength: 1, maxLength: 50 }),
        pack_description: fc.string({ maxLength: 100 }),
        source_name: fc.string({ minLength: 1, maxLength: 20 }),
        author_name: fc.string({ minLength: 1, maxLength: 20 }),
        share_mode: fc.constantFrom(...SHARE_MODES),
        credits_price: fc.integer({ min: 0, max: 10000 }),
        download_count: fc.integer({ min: 0, max: 100000 }),
        created_at: fc.integer({ min: 2020, max: 2025 }).chain(year =>
            fc.integer({ min: 1, max: 12 }).chain(month =>
                fc.integer({ min: 1, max: 28 }).map(day =>
                    `${year}-${String(month).padStart(2, '0')}-${String(day).padStart(2, '0')}`
                )
            )
        ),
    }).map(fields => new main.PackListingInfo(fields));
}

/** Generator for a list of packs (0–30 items). */
const packsArb = fc.array(packArb(), { minLength: 0, maxLength: 30 });

/** Generator for any ShareModeFilter value. */
const filterArb: fc.Arbitrary<ShareModeFilter> = fc.constantFrom(...FILTER_OPTIONS);

/** Generator for non-'all' filter values. */
const specificFilterArb: fc.Arbitrary<ShareModeFilter> = fc.constantFrom('free', 'per_use', 'subscription');

describe('marketplaceFilter Property-Based Tests', () => {
    // ─────────────────────────────────────────────────────────────────────
    // Feature: marketplace-browse-purchase, Property 1: 付费类型筛选正确性
    // ─────────────────────────────────────────────────────────────────────
    describe('Property 1: 付费类型筛选正确性', () => {
        /**
         * **Validates: Requirements 3.2, 3.3**
         *
         * For any pack list and any ShareModeFilter:
         * - When filter is 'all', the result contains all packs from the input.
         * - When filter is a specific share_mode, every pack in the result
         *   has share_mode matching the filter value.
         */

        it('filter "all" returns all packs unchanged', () => {
            fc.assert(
                fc.property(packsArb, (packs) => {
                    const result = filterByShareMode(packs, 'all');
                    expect(result).toHaveLength(packs.length);
                    expect(result).toEqual(packs);
                }),
                { numRuns: 100 },
            );
        });

        it('specific filter: every result pack matches the selected share_mode', () => {
            fc.assert(
                fc.property(packsArb, specificFilterArb, (packs, filter) => {
                    const result = filterByShareMode(packs, filter);
                    for (const pack of result) {
                        expect(pack.share_mode).toBe(filter);
                    }
                }),
                { numRuns: 100 },
            );
        });

        it('specific filter: result is a subset — no packs are invented', () => {
            fc.assert(
                fc.property(packsArb, specificFilterArb, (packs, filter) => {
                    const result = filterByShareMode(packs, filter);
                    expect(result.length).toBeLessThanOrEqual(packs.length);
                    for (const pack of result) {
                        expect(packs).toContain(pack);
                    }
                }),
                { numRuns: 100 },
            );
        });

        it('specific filter: result contains ALL matching packs (completeness)', () => {
            fc.assert(
                fc.property(packsArb, specificFilterArb, (packs, filter) => {
                    const result = filterByShareMode(packs, filter);
                    const expectedCount = packs.filter(p => p.share_mode === filter).length;
                    expect(result).toHaveLength(expectedCount);
                }),
                { numRuns: 100 },
            );
        });
    });

    // ─────────────────────────────────────────────────────────────────────
    // Feature: marketplace-browse-purchase, Property 2: 关键字搜索正确性
    // ─────────────────────────────────────────────────────────────────────
    describe('Property 2: 关键字搜索正确性', () => {
        /**
         * **Validates: Requirements 5.2, 5.3, 5.4**
         *
         * For any pack list and any search keyword (including empty and mixed case):
         * - Every pack in the result has pack_name or pack_description containing
         *   the keyword (case-insensitive).
         * - When keyword is empty, the result contains all packs.
         */

        /** Generator for search keywords including empty, whitespace-only, and mixed case. */
        const keywordArb = fc.oneof(
            fc.constant(''),
            fc.constant('   '),
            fc.string({ minLength: 1, maxLength: 20 }),
        );

        it('empty/whitespace keyword returns all packs', () => {
            const emptyKeywordArb = fc.constantFrom('', '  ', '\t', ' \n ');
            fc.assert(
                fc.property(packsArb, emptyKeywordArb, (packs, keyword) => {
                    const result = filterByKeyword(packs, keyword);
                    expect(result).toHaveLength(packs.length);
                    expect(result).toEqual(packs);
                }),
                { numRuns: 100 },
            );
        });

        it('every result pack contains the keyword in pack_name or pack_description (case-insensitive)', () => {
            fc.assert(
                fc.property(packsArb, keywordArb, (packs, keyword) => {
                    const result = filterByKeyword(packs, keyword);
                    const trimmed = keyword.trim().toLowerCase();
                    if (!trimmed) return; // empty keyword tested separately
                    for (const pack of result) {
                        const nameMatch = pack.pack_name.toLowerCase().includes(trimmed);
                        const descMatch = pack.pack_description
                            ? pack.pack_description.toLowerCase().includes(trimmed)
                            : false;
                        expect(nameMatch || descMatch).toBe(true);
                    }
                }),
                { numRuns: 100 },
            );
        });

        it('result is a subset — no packs are invented', () => {
            fc.assert(
                fc.property(packsArb, keywordArb, (packs, keyword) => {
                    const result = filterByKeyword(packs, keyword);
                    expect(result.length).toBeLessThanOrEqual(packs.length);
                    for (const pack of result) {
                        expect(packs).toContain(pack);
                    }
                }),
                { numRuns: 100 },
            );
        });

        it('result contains ALL matching packs (completeness)', () => {
            fc.assert(
                fc.property(packsArb, keywordArb, (packs, keyword) => {
                    const result = filterByKeyword(packs, keyword);
                    const trimmed = keyword.trim().toLowerCase();
                    const expectedCount = !trimmed
                        ? packs.length
                        : packs.filter(p =>
                            p.pack_name.toLowerCase().includes(trimmed) ||
                            (p.pack_description && p.pack_description.toLowerCase().includes(trimmed))
                        ).length;
                    expect(result).toHaveLength(expectedCount);
                }),
                { numRuns: 100 },
            );
        });

        it('search is case-insensitive: same results regardless of keyword casing', () => {
            fc.assert(
                fc.property(packsArb, fc.string({ minLength: 1, maxLength: 15 }), (packs, keyword) => {
                    const lower = filterByKeyword(packs, keyword.toLowerCase());
                    const upper = filterByKeyword(packs, keyword.toUpperCase());
                    const mixed = filterByKeyword(packs, keyword);
                    expect(lower).toEqual(upper);
                    expect(lower).toEqual(mixed);
                }),
                { numRuns: 100 },
            );
        });
    });

    // ─────────────────────────────────────────────────────────────────────
    // Feature: marketplace-browse-purchase, Property 3: 排序正确性
    // ─────────────────────────────────────────────────────────────────────
    describe('Property 3: 排序正确性', () => {
        /**
         * **Validates: Requirements 4.4**
         *
         * For any pack list and any sort configuration (by created_at or
         * credits_price, ascending or descending), adjacent elements in the
         * sorted result should satisfy the ordering relation for the chosen
         * sort direction.
         */

        /** Generator for sort field. */
        const sortFieldArb: fc.Arbitrary<SortField> = fc.constantFrom('created_at', 'credits_price');

        /** Generator for sort direction. */
        const sortDirectionArb: fc.Arbitrary<SortDirection> = fc.constantFrom('desc', 'asc');

        it('adjacent elements satisfy the ordering relation for the chosen direction', () => {
            fc.assert(
                fc.property(packsArb, sortFieldArb, sortDirectionArb, (packs, field, direction) => {
                    const sorted = sortPacks(packs, field, direction);
                    for (let i = 0; i < sorted.length - 1; i++) {
                        if (field === 'created_at') {
                            const cmp = sorted[i].created_at.localeCompare(sorted[i + 1].created_at);
                            if (direction === 'asc') {
                                expect(cmp).toBeLessThanOrEqual(0);
                            } else {
                                expect(cmp).toBeGreaterThanOrEqual(0);
                            }
                        } else {
                            if (direction === 'asc') {
                                expect(sorted[i].credits_price).toBeLessThanOrEqual(sorted[i + 1].credits_price);
                            } else {
                                expect(sorted[i].credits_price).toBeGreaterThanOrEqual(sorted[i + 1].credits_price);
                            }
                        }
                    }
                }),
                { numRuns: 100 },
            );
        });

        it('sorting preserves all elements (no items added or lost)', () => {
            fc.assert(
                fc.property(packsArb, sortFieldArb, sortDirectionArb, (packs, field, direction) => {
                    const sorted = sortPacks(packs, field, direction);
                    expect(sorted).toHaveLength(packs.length);
                    for (const pack of sorted) {
                        expect(packs).toContain(pack);
                    }
                }),
                { numRuns: 100 },
            );
        });

        it('sorting does not mutate the original array', () => {
            fc.assert(
                fc.property(packsArb, sortFieldArb, sortDirectionArb, (packs, field, direction) => {
                    const original = [...packs];
                    sortPacks(packs, field, direction);
                    expect(packs).toEqual(original);
                }),
                { numRuns: 100 },
            );
        });
    });
});
