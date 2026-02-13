/**
 * Property-Based Tests for ActivationModal Email Validation Logic
 *
 * Uses fast-check to verify universal properties across randomized inputs.
 * Each property test runs a minimum of 100 iterations.
 *
 * Feature: sn-email-binding, Property 1: Invalid email rejection
 */

import fc from 'fast-check';
import { describe, it, expect } from 'vitest';

/**
 * Pure function extracted from ActivationModal.tsx email validation logic:
 *
 *   if (!activationEmail) → invalid
 *   const atIndex = activationEmail.indexOf('@');
 *   if (atIndex < 1 || atIndex >= activationEmail.length - 1) → invalid
 *   if (!activationEmail.substring(atIndex + 1).includes('.')) → invalid
 */
function isValidEmail(email: string): boolean {
  if (!email) return false;
  const atIndex = email.indexOf('@');
  if (atIndex < 1 || atIndex >= email.length - 1) return false;
  if (!email.substring(atIndex + 1).includes('.')) return false;
  return true;
}

/** Generates alphanumeric strings of given length range */
const alphanumArb = (min: number, max: number) =>
  fc.string({ minLength: min, maxLength: max }).map(s =>
    s.replace(/[^a-z0-9]/g, 'a')
  ).filter(s => s.length >= min);

/** Generates lowercase alpha strings of given length range */
const alphaArb = (min: number, max: number) =>
  fc.string({ minLength: min, maxLength: max }).map(s =>
    s.replace(/[^a-z]/g, 'a')
  ).filter(s => s.length >= min);

/**
 * Arbitrary: generates valid emails in local@domain.tld format.
 */
const validEmailArb = fc.tuple(
  alphanumArb(1, 20),
  alphanumArb(1, 10),
  alphaArb(2, 5),
).map(([local, domain, tld]) => `${local}@${domain}.${tld}`);

describe('ActivationModal Email Validation Property-Based Tests', () => {
  // ─────────────────────────────────────────────────────────────────────────
  // Feature: sn-email-binding, Property 1: Invalid email rejection
  // ─────────────────────────────────────────────────────────────────────────
  describe('Property 1: Invalid email rejection', () => {
    /**
     * **Validates: Requirements 1.3, 1.4**
     *
     * Empty strings must always be rejected.
     */
    it('rejects empty string', () => {
      expect(isValidEmail('')).toBe(false);
    });

    /**
     * **Validates: Requirements 1.3, 1.4**
     *
     * Any string without an '@' character is invalid.
     */
    it('rejects strings with no @ character', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 50 }).filter(s => !s.includes('@')),
          (email: string) => {
            expect(isValidEmail(email)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 1.3, 1.4**
     *
     * Any string where '@' is the first character (no local part) is invalid.
     */
    it('rejects strings where @ is at the start', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 30 }).filter(s => !s.includes('@')),
          (rest: string) => {
            expect(isValidEmail(`@${rest}`)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 1.3, 1.4**
     *
     * Any string where '@' is the last character (no domain part) is invalid.
     */
    it('rejects strings where @ is at the end', () => {
      fc.assert(
        fc.property(
          fc.string({ minLength: 1, maxLength: 30 }).filter(s => !s.includes('@')),
          (prefix: string) => {
            expect(isValidEmail(`${prefix}@`)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 1.3, 1.4**
     *
     * Any string with local@domain but no dot in domain part is invalid.
     */
    it('rejects emails with no dot in domain part', () => {
      fc.assert(
        fc.property(
          alphanumArb(1, 15),
          fc.string({ minLength: 1, maxLength: 15 }).map(s => s.replace(/[@.]/g, 'a')).filter(s => s.length >= 1),
          (local: string, domain: string) => {
            expect(isValidEmail(`${local}@${domain}`)).toBe(false);
          }
        ),
        { numRuns: 100 }
      );
    });

    /**
     * **Validates: Requirements 1.3, 1.4**
     *
     * Any well-formed email (local@domain.tld) must be accepted.
     */
    it('accepts valid emails in local@domain.tld format', () => {
      fc.assert(
        fc.property(validEmailArb, (email: string) => {
          expect(isValidEmail(email)).toBe(true);
        }),
        { numRuns: 100 }
      );
    });
  });
});
