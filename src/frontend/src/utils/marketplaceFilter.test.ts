import { describe, it, expect } from 'vitest';
import { filterByShareMode, filterByKeyword, sortPacks } from './marketplaceFilter';
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
        share_mode: 'free',
        credits_price: 0,
        download_count: 0,
        created_at: '2024-01-01',
        ...overrides,
    });
}

describe('filterByShareMode', () => {
    const packs = [
        makePack({ id: 1, share_mode: 'free' }),
        makePack({ id: 2, share_mode: 'per_use' }),
        makePack({ id: 3, share_mode: 'subscription' }),
        makePack({ id: 4, share_mode: 'free' }),
    ];

    it('returns all packs when filter is "all"', () => {
        expect(filterByShareMode(packs, 'all')).toEqual(packs);
    });

    it('filters by free', () => {
        const result = filterByShareMode(packs, 'free');
        expect(result).toHaveLength(2);
        expect(result.every(p => p.share_mode === 'free')).toBe(true);
    });

    it('filters by per_use', () => {
        const result = filterByShareMode(packs, 'per_use');
        expect(result).toHaveLength(1);
        expect(result[0].share_mode).toBe('per_use');
    });

    it('returns empty array when no match', () => {
        const freePacks = [makePack({ share_mode: 'free' })];
        expect(filterByShareMode(freePacks, 'subscription')).toEqual([]);
    });

    it('handles empty input', () => {
        expect(filterByShareMode([], 'free')).toEqual([]);
    });
});

describe('filterByKeyword', () => {
    const packs = [
        makePack({ id: 1, pack_name: 'Sales Analysis', pack_description: 'Analyze sales data' }),
        makePack({ id: 2, pack_name: 'Finance Report', pack_description: 'Monthly finance' }),
        makePack({ id: 3, pack_name: 'HR Dashboard', pack_description: 'Employee metrics' }),
    ];

    it('returns all packs for empty keyword', () => {
        expect(filterByKeyword(packs, '')).toEqual(packs);
    });

    it('returns all packs for whitespace-only keyword', () => {
        expect(filterByKeyword(packs, '   ')).toEqual(packs);
    });

    it('matches by pack_name (case-insensitive)', () => {
        const result = filterByKeyword(packs, 'SALES');
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe(1);
    });

    it('matches by pack_description (case-insensitive)', () => {
        const result = filterByKeyword(packs, 'employee');
        expect(result).toHaveLength(1);
        expect(result[0].id).toBe(3);
    });

    it('returns empty when no match', () => {
        expect(filterByKeyword(packs, 'nonexistent')).toEqual([]);
    });

    it('handles empty input', () => {
        expect(filterByKeyword([], 'test')).toEqual([]);
    });
});

describe('sortPacks', () => {
    const packs = [
        makePack({ id: 1, created_at: '2024-03-01', credits_price: 20 }),
        makePack({ id: 2, created_at: '2024-01-01', credits_price: 5 }),
        makePack({ id: 3, created_at: '2024-02-01', credits_price: 50 }),
    ];

    it('sorts by created_at descending', () => {
        const result = sortPacks(packs, 'created_at', 'desc');
        expect(result.map(p => p.id)).toEqual([1, 3, 2]);
    });

    it('sorts by created_at ascending', () => {
        const result = sortPacks(packs, 'created_at', 'asc');
        expect(result.map(p => p.id)).toEqual([2, 3, 1]);
    });

    it('sorts by credits_price descending', () => {
        const result = sortPacks(packs, 'credits_price', 'desc');
        expect(result.map(p => p.id)).toEqual([3, 1, 2]);
    });

    it('sorts by credits_price ascending', () => {
        const result = sortPacks(packs, 'credits_price', 'asc');
        expect(result.map(p => p.id)).toEqual([2, 1, 3]);
    });

    it('does not mutate the input array', () => {
        const original = [...packs];
        sortPacks(packs, 'credits_price', 'desc');
        expect(packs).toEqual(original);
    });

    it('handles empty array', () => {
        expect(sortPacks([], 'created_at', 'desc')).toEqual([]);
    });

    it('handles single element', () => {
        const single = [makePack({ id: 1 })];
        expect(sortPacks(single, 'created_at', 'desc')).toEqual(single);
    });
});
