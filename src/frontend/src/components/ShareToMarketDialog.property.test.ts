/**
 * Feature: marketplace-sn-auth-billing, Property 11: 分享表单验证正确性
 *
 * For any pricing_model and form parameter combination, validateShareForm returns
 * valid=true when parameters are complete and valid, and valid=false with specific
 * error fields when parameters are missing or invalid.
 *
 * Validates: Requirements 9.3, 9.4, 9.5
 *
 * Uses fast-check for property-based testing with minimum 100 iterations.
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';
import { validateShareForm } from './ShareToMarketDialog';

type PricingModel = 'free' | 'per_use' | 'subscription';

const pricingModelArb = fc.constantFrom<PricingModel>('free', 'per_use', 'subscription');
const validDescriptionArb = fc.string({ minLength: 1, maxLength: 200 }).filter(s => s.trim().length > 0);
const emptyDescriptionArb = fc.constantFrom('', ' ', '\t', '\n', '   ');

describe('Property 11: ShareToMarketDialog form validation correctness', () => {
  it('valid description + valid pricing params => valid=true, no errors', () => {
    fc.assert(
      fc.property(
        validDescriptionArb,
        pricingModelArb,
        fc.integer({ min: 1, max: 100 }),
        fc.integer({ min: 100, max: 1000 }),
        (description, model, perUsePrice, subPrice) => {
          let price: string;
          switch (model) {
            case 'free': price = ''; break;
            case 'per_use': price = String(perUsePrice); break;
            case 'subscription': price = String(subPrice); break;
          }
          const result = validateShareForm(description, model, price);
          expect(result.valid).toBe(true);
          expect(result.descriptionError).toBe(false);
          expect(result.priceError).toBe(false);
          expect(result.priceRangeError).toBe(false);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('empty description => valid=false with descriptionError=true', () => {
    fc.assert(
      fc.property(
        emptyDescriptionArb,
        pricingModelArb,
        fc.string({ maxLength: 20 }),
        (description, model, price) => {
          const result = validateShareForm(description, model, price);
          expect(result.valid).toBe(false);
          expect(result.descriptionError).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('per_use with price outside [1,100] => valid=false', () => {
    const outOfRangeArb = fc.oneof(
      fc.integer({ min: 101, max: 50000 }),
      fc.integer({ min: -1000, max: 0 }),
    );
    fc.assert(
      fc.property(
        validDescriptionArb,
        outOfRangeArb,
        (description, price) => {
          const result = validateShareForm(description, 'per_use', String(price));
          expect(result.valid).toBe(false);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('subscription with price outside [100,1000] => valid=false', () => {
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

  it('non-free model with invalid price string => priceError=true', () => {
    const nonFreeArb = fc.constantFrom<PricingModel>('per_use', 'subscription');
    const invalidPriceArb = fc.constantFrom('', 'abc', 'NaN', ' ', '0', '-5');
    fc.assert(
      fc.property(
        validDescriptionArb,
        nonFreeArb,
        invalidPriceArb,
        (description, model, price) => {
          const result = validateShareForm(description, model, price);
          expect(result.priceError).toBe(true);
          expect(result.valid).toBe(false);
        }
      ),
      { numRuns: 100 }
    );
  });

  it('free model ignores price entirely', () => {
    fc.assert(
      fc.property(
        validDescriptionArb,
        fc.string({ maxLength: 50 }),
        (description, price) => {
          const result = validateShareForm(description, 'free', price);
          expect(result.priceError).toBe(false);
          expect(result.priceRangeError).toBe(false);
          expect(result.valid).toBe(true);
        }
      ),
      { numRuns: 100 }
    );
  });
});
