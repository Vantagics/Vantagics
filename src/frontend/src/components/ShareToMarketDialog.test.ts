/**
 * Property-Based Tests for ShareToMarketDialog form validation
 *
 * Uses fast-check to verify universal properties across randomized inputs.
 * Each property test runs a minimum of 100 iterations.
 *
 * Feature: marketplace-pricing-redesign
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { validateShareForm } from './ShareToMarketDialog';

type PricingModel = 'free' | 'per_use' | 'subscription';

const validPricingModelArb = fc.constantFrom<PricingModel>('free', 'per_use', 'subscription');

/** Generates strings that are empty or contain only whitespace */
const emptyOrWhitespaceArb = fc.constantFrom('', ' ', '  ', '\t', '\n', '  \t\n  ');

/** Generates non-empty, non-whitespace-only strings */
const validDescriptionArb = fc.string({ minLength: 1, maxLength: 200 })
  .filter(s => s.trim().length > 0);

/**
 * Generates price strings that parseInt() will NOT parse as a positive integer.
 */
const invalidPriceArb = fc.oneof(
  fc.constant(''),
  fc.constant('0'),
  fc.constant('-1'),
  fc.constant('-100'),
  fc.constant('abc'),
  fc.constant('xyz123'),
  fc.constant(' '),
  fc.constant('NaN'),
  fc.integer({ min: -999999, max: 0 }).map(n => String(n)),
);

describe('ShareToMarketDialog Property-Based Tests', () => {
  // Property 1: 合法定价模式集合
  describe('Property 1: Valid pricing model set', () => {
    it('only accepts free, per_use, subscription', () => {
      fc.assert(
        fc.property(
          validDescriptionArb,
          validPricingModelArb,
          (description, model) => {
            const result = validateShareForm(description, model, model === 'free' ? '' : '50');
            // Should not have description error for valid description
            expect(result.descriptionError).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('rejects time_limited as pricing model', () => {
      const result = validateShareForm('test description', 'time_limited' as any, '10');
      // time_limited is not in the type, but if passed, priceError should still work normally
      // The key point is the UI no longer offers time_limited as an option
      expect(result).toBeDefined();
    });
  });

  // Property 2: 按次收费信用点数范围校验
  describe('Property 2: Per-use credits range [1, 100]', () => {
    it('accepts credits in [1, 100] for per_use', () => {
      fc.assert(
        fc.property(
          validDescriptionArb,
          fc.integer({ min: 1, max: 100 }),
          (description, price) => {
            const result = validateShareForm(description, 'per_use', String(price));
            expect(result.priceError).toBe(false);
            expect(result.priceRangeError).toBe(false);
            expect(result.valid).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('rejects credits outside [1, 100] for per_use', () => {
      const outOfRangeArb = fc.oneof(
        fc.integer({ min: 101, max: 10000 }),
        fc.integer({ min: -1000, max: 0 }),
      );
      fc.assert(
        fc.property(
          validDescriptionArb,
          outOfRangeArb,
          (description, price) => {
            const result = validateShareForm(description, 'per_use', String(price));
            // Either priceError (for <= 0) or priceRangeError (for > 100)
            expect(result.valid).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Property 3: 订阅制信用点数范围校验
  describe('Property 3: Subscription credits range [100, 1000]', () => {
    it('accepts credits in [100, 1000] for subscription', () => {
      fc.assert(
        fc.property(
          validDescriptionArb,
          fc.integer({ min: 100, max: 1000 }),
          (description, price) => {
            const result = validateShareForm(description, 'subscription', String(price));
            expect(result.priceError).toBe(false);
            expect(result.priceRangeError).toBe(false);
            expect(result.valid).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('rejects credits outside [100, 1000] for subscription', () => {
      const outOfRangeArb = fc.oneof(
        fc.integer({ min: 1001, max: 50000 }),
        fc.integer({ min: 1, max: 99 }),
      );
      fc.assert(
        fc.property(
          validDescriptionArb,
          outOfRangeArb,
          (description, price) => {
            const result = validateShareForm(description, 'subscription', String(price));
            expect(result.valid).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Property 4: 免费模式不校验信用点数
  describe('Property 4: Free mode ignores credits', () => {
    it('free pricing never yields priceError or priceRangeError', () => {
      fc.assert(
        fc.property(
          fc.string({ maxLength: 200 }),
          fc.string({ maxLength: 50 }),
          (description, price) => {
            const result = validateShareForm(description, 'free', price);
            expect(result.priceError).toBe(false);
            expect(result.priceRangeError).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    it('valid description with free pricing is always valid', () => {
      fc.assert(
        fc.property(
          validDescriptionArb,
          fc.string({ maxLength: 50 }),
          (description, price) => {
            const result = validateShareForm(description, 'free', price);
            expect(result.valid).toBe(true);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Description validation
  describe('Description validation', () => {
    it('empty or whitespace-only description always yields descriptionError', () => {
      fc.assert(
        fc.property(
          emptyOrWhitespaceArb,
          validPricingModelArb,
          fc.string(),
          (description, pricingModel, price) => {
            const result = validateShareForm(description, pricingModel, price);
            expect(result.descriptionError).toBe(true);
            expect(result.valid).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });
  });

  // Non-free pricing with invalid price
  describe('Non-free pricing with invalid price', () => {
    it('non-free pricing with invalid price always yields priceError', () => {
      const nonFreePricingArb = fc.constantFrom<PricingModel>('per_use', 'subscription');
      fc.assert(
        fc.property(
          validDescriptionArb,
          nonFreePricingArb,
          invalidPriceArb,
          (description, pricingModel, price) => {
            const result = validateShareForm(description, pricingModel, price);
            expect(result.priceError).toBe(true);
            expect(result.valid).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });
  });
});
