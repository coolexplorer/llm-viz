import { describe, it, expect } from 'vitest';
import { calculateCost, formatUSD, getModelPricing } from '@/lib/cost-calculator';

describe('calculateCost', () => {
  it('calculates basic input/output costs for claude-sonnet-4-6', () => {
    const result = calculateCost('claude-sonnet-4-6', 1_000_000, 1_000_000);
    expect(result.inputCost).toBeCloseTo(3, 5);
    expect(result.outputCost).toBeCloseTo(15, 5);
    expect(result.cacheWriteCost).toBe(0);
    expect(result.cacheReadCost).toBe(0);
    expect(result.totalCost).toBeCloseTo(18, 5);
    expect(result.cacheSavings).toBe(0);
  });

  it('calculates cache write cost for anthropic', () => {
    const result = calculateCost('claude-sonnet-4-6', 0, 0, 1_000_000, 0);
    expect(result.cacheWriteCost).toBeCloseTo(3.75, 5);
    expect(result.totalCost).toBeCloseTo(3.75, 5);
  });

  it('calculates cache read cost and savings for anthropic', () => {
    const result = calculateCost('claude-sonnet-4-6', 0, 0, 0, 1_000_000);
    expect(result.cacheReadCost).toBeCloseTo(0.3, 5);
    // savings = (3 - 0.3) * 1M / 1M = 2.7
    expect(result.cacheSavings).toBeCloseTo(2.7, 5);
  });

  it('calculates costs for openai gpt-4o', () => {
    const result = calculateCost('gpt-4o', 1_000_000, 1_000_000);
    expect(result.inputCost).toBeCloseTo(2.5, 5);
    expect(result.outputCost).toBeCloseTo(10, 5);
    expect(result.totalCost).toBeCloseTo(12.5, 5);
  });

  it('falls back to gpt-4o-mini pricing for unknown model', () => {
    const result = calculateCost('unknown-model', 1_000_000, 0);
    const fallback = calculateCost('gpt-4o-mini', 1_000_000, 0);
    expect(result.inputCost).toBe(fallback.inputCost);
  });

  it('handles zero tokens', () => {
    const result = calculateCost('claude-haiku-4-5', 0, 0, 0, 0);
    expect(result.inputCost).toBe(0);
    expect(result.outputCost).toBe(0);
    expect(result.totalCost).toBe(0);
    expect(result.cacheSavings).toBe(0);
  });

  it('calculates cost for claude-opus-4-6', () => {
    const result = calculateCost('claude-opus-4-6', 1_000_000, 1_000_000);
    expect(result.inputCost).toBeCloseTo(15, 5);
    expect(result.outputCost).toBeCloseTo(75, 5);
  });

  it('returns zero cache savings when cacheReadTokens is 0', () => {
    const result = calculateCost('claude-sonnet-4-6', 100, 100, 0, 0);
    expect(result.cacheSavings).toBe(0);
  });
});

describe('formatUSD', () => {
  it('formats amounts >= $0.01 with 4 decimal places', () => {
    expect(formatUSD(1.5)).toBe('$1.5000');
    expect(formatUSD(0.01)).toBe('$0.0100');
    expect(formatUSD(100)).toBe('$100.0000');
  });

  it('formats small amounts < $0.01 in millicents', () => {
    expect(formatUSD(0.001)).toBe('$1.000m');
    expect(formatUSD(0.0001)).toBe('$0.100m');
    expect(formatUSD(0.000005)).toBe('$0.005m');
  });

  it('formats zero correctly', () => {
    expect(formatUSD(0)).toBe('$0.000m');
  });
});

describe('getModelPricing', () => {
  it('returns correct pricing for claude-sonnet-4-6', () => {
    const pricing = getModelPricing('claude-sonnet-4-6');
    expect(pricing.inputPer1M).toBe(3);
    expect(pricing.outputPer1M).toBe(15);
    expect(pricing.cacheWritePer1M).toBe(3.75);
    expect(pricing.cacheReadPer1M).toBe(0.3);
  });

  it('falls back to gpt-4o-mini for unknown model', () => {
    const pricing = getModelPricing('nonexistent-model');
    const fallback = getModelPricing('gpt-4o-mini');
    expect(pricing).toEqual(fallback);
  });

  it('returns pricing for gpt-4o', () => {
    const pricing = getModelPricing('gpt-4o');
    expect(pricing.inputPer1M).toBe(2.5);
    expect(pricing.outputPer1M).toBe(10);
  });
});
