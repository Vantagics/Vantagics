/**
 * Property-Based Tests for pack-list rendering
 *
 * Feature: datasource-pack-loader, Property 2: 分析包列表项显示完整信息
 * **Validates: Requirements 1.2**
 *
 * Uses fast-check to verify that for any LocalPackInfo object,
 * the pack-list rendering displays pack_name, description,
 * source_name, author, and created_at.
 */

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import fc from 'fast-check';
import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock Wails bindings BEFORE importing the component
vi.mock('../../wailsjs/go/main/App', () => ({
    LoadQuickAnalysisPack: vi.fn(),
    LoadQuickAnalysisPackWithPassword: vi.fn(),
    ExecuteQuickAnalysisPack: vi.fn(),
    ListLocalQuickAnalysisPacks: vi.fn(),
    LoadQuickAnalysisPackByPath: vi.fn(),
}));

// Mock i18n
vi.mock('../i18n', () => ({
    useLanguage: () => ({
        language: 'English',
        t: (key: string) => key,
    }),
}));

import ImportPackDialog from './ImportPackDialog';
import { ListLocalQuickAnalysisPacks } from '../../wailsjs/go/main/App';

const mockListPacks = vi.mocked(ListLocalQuickAnalysisPacks);

/**
 * Helper: generate a non-empty alphanumeric string suitable for display text.
 * Uses fc.string() with filtering to ensure non-empty, printable content.
 */
const alphaStringArb = (minLen = 2, maxLen = 20) =>
    fc.array(
        fc.constantFrom(...'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789'.split('')),
        { minLength: minLen, maxLength: maxLen }
    ).map(chars => chars.join(''));

/**
 * Arbitrary for generating a LocalPackInfo-like object with non-empty fields
 * to ensure they are visible in the rendered output.
 */
const localPackInfoArb = fc.record({
    file_name: alphaStringArb(3, 15).map(s => s + '.qap'),
    file_path: alphaStringArb(5, 20).map(s => '/tmp/' + s + '.qap'),
    pack_name: alphaStringArb(2, 25),
    description: alphaStringArb(2, 40),
    source_name: alphaStringArb(2, 15),
    author: alphaStringArb(2, 15),
    created_at: fc.integer({
        min: new Date('2020-01-01T00:00:00Z').getTime(),
        max: new Date('2030-12-31T23:59:59Z').getTime(),
    }).map(ts => new Date(ts).toISOString()),
    is_encrypted: fc.boolean(),
});

describe('ImportPackDialog pack-list rendering Property-Based Tests', () => {
    // Feature: datasource-pack-loader, Property 2: 分析包列表项显示完整信息
    describe('Property 2: 分析包列表项显示完整信息', () => {
        beforeEach(() => {
            vi.clearAllMocks();
        });

        /**
         * **Validates: Requirements 1.2**
         *
         * For any LocalPackInfo with non-empty fields, the rendered pack-list
         * should contain pack_name, source_name, author, and created_at.
         * description is also displayed when non-empty.
         */
        it('renders pack_name, description, source_name, author, and created_at for any LocalPackInfo', async () => {
            await fc.assert(
                fc.asyncProperty(localPackInfoArb, async (packInfo) => {
                    mockListPacks.mockResolvedValue([packInfo as any]);

                    const { unmount } = render(
                        <ImportPackDialog
                            isOpen={true}
                            onClose={() => {}}
                            onConfirm={() => {}}
                            dataSourceId="test-ds"
                        />
                    );

                    await waitFor(() => {
                        // pack_name should be displayed
                        expect(screen.getByText(packInfo.pack_name)).toBeInTheDocument();
                    });

                    // description should be displayed (non-empty)
                    expect(screen.getByText(packInfo.description)).toBeInTheDocument();

                    // source_name, author, created_at are rendered in a combined line
                    // The component renders them as separate spans within a <p> tag
                    const container = document.body;
                    const textContent = container.textContent || '';

                    expect(textContent).toContain(packInfo.source_name);
                    expect(textContent).toContain(packInfo.author);

                    // created_at is formatted via formatDate (toLocaleString),
                    // so we check the raw date is parseable and some representation appears
                    const formattedDate = new Date(packInfo.created_at).toLocaleString();
                    expect(textContent).toContain(formattedDate);

                    unmount();
                }),
                { numRuns: 20 }
            );
        });
    });
});
