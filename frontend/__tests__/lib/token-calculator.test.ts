import { describe, it, expect } from 'vitest';
import {
  calculateCacheHitRate,
  calculateContextUsage,
  aggregateSessionStats,
  formatTokenCount,
} from '@/lib/token-calculator';
import type { TokenDataPoint } from '@/types/token-data';

function makeDataPoint(overrides: Partial<TokenDataPoint> = {}): TokenDataPoint {
  return {
    timestamp: Date.now(),
    provider: 'anthropic',
    model: 'claude-sonnet-4-6',
    inputTokens: 100,
    outputTokens: 50,
    cacheReadTokens: 0,
    cacheCreationTokens: 0,
    totalTokens: 150,
    costUSD: 0.001,
    ...overrides,
  };
}

describe('calculateCacheHitRate', () => {
  it('returns 0 when both values are 0', () => {
    expect(calculateCacheHitRate(0, 0)).toBe(0);
  });

  it('returns 100% when all tokens are cache reads', () => {
    expect(calculateCacheHitRate(1000, 0)).toBe(100);
  });

  it('returns 0% when all tokens are cache creations', () => {
    expect(calculateCacheHitRate(0, 1000)).toBe(0);
  });

  it('returns 50% for equal reads and creations', () => {
    expect(calculateCacheHitRate(500, 500)).toBe(50);
  });

  it('calculates partial hit rate', () => {
    expect(calculateCacheHitRate(750, 250)).toBe(75);
  });
});

describe('calculateContextUsage', () => {
  it('returns correct structure', () => {
    const result = calculateContextUsage(1000, 0, 0, 'gpt-4o');
    expect(result).toHaveProperty('model', 'gpt-4o');
    expect(result).toHaveProperty('maxTokens', 128_000);
    expect(result).toHaveProperty('currentUsed', 1000);
    expect(result).toHaveProperty('remainingTokens', 127_000);
    expect(result).toHaveProperty('isWarning', false);
    expect(result).toHaveProperty('isCritical', false);
  });

  it('detects warning at >80%', () => {
    // 85% of 128K = ~108,800
    const result = calculateContextUsage(108_800, 0, 0, 'gpt-4o');
    expect(result.isWarning).toBe(true);
    expect(result.isCritical).toBe(false);
  });

  it('detects critical at >95%', () => {
    // 96% of 128K = ~122,880
    const result = calculateContextUsage(122_880, 0, 0, 'gpt-4o');
    expect(result.isCritical).toBe(true);
    expect(result.isWarning).toBe(true);
  });

  it('includes cache tokens in context usage', () => {
    const result = calculateContextUsage(1000, 500, 500, 'gpt-4o');
    expect(result.currentUsed).toBe(2000);
  });

  it('calculates utilization percentage correctly', () => {
    // 64K out of 128K = 50%
    const result = calculateContextUsage(64_000, 0, 0, 'gpt-4o');
    expect(result.utilizationPercent).toBeCloseTo(50, 1);
  });

  it('uses model limit for context', () => {
    const resultSonnet = calculateContextUsage(1000, 0, 0, 'claude-sonnet-4-6');
    expect(resultSonnet.maxTokens).toBe(200_000);
  });
});

describe('aggregateSessionStats', () => {
  it('returns zero stats for empty array', () => {
    const stats = aggregateSessionStats([]);
    expect(stats.totalRequests).toBe(0);
    expect(stats.totalInputTokens).toBe(0);
    expect(stats.totalCostUSD).toBe(0);
    expect(stats.cacheHitRate).toBe(0);
  });

  it('aggregates token counts correctly', () => {
    const points = [
      makeDataPoint({ inputTokens: 100, outputTokens: 50, cacheReadTokens: 10, cacheCreationTokens: 5, costUSD: 0.01 }),
      makeDataPoint({ inputTokens: 200, outputTokens: 100, cacheReadTokens: 20, cacheCreationTokens: 10, costUSD: 0.02 }),
    ];
    const stats = aggregateSessionStats(points);
    expect(stats.totalRequests).toBe(2);
    expect(stats.totalInputTokens).toBe(300);
    expect(stats.totalOutputTokens).toBe(150);
    expect(stats.totalCacheReadTokens).toBe(30);
    expect(stats.totalCacheCreationTokens).toBe(15);
    expect(stats.totalCostUSD).toBeCloseTo(0.03, 5);
  });

  it('calculates cache hit rate from aggregated totals', () => {
    const points = [
      makeDataPoint({ cacheReadTokens: 300, cacheCreationTokens: 100 }),
      makeDataPoint({ cacheReadTokens: 100, cacheCreationTokens: 100 }),
    ];
    const stats = aggregateSessionStats(points);
    // total read = 400, total creation = 200, rate = 400/600 * 100
    expect(stats.cacheHitRate).toBeCloseTo((400 / 600) * 100, 1);
  });
});

describe('formatTokenCount', () => {
  it('formats small numbers as-is', () => {
    expect(formatTokenCount(0)).toBe('0');
    expect(formatTokenCount(500)).toBe('500');
    expect(formatTokenCount(999)).toBe('999');
  });

  it('formats thousands with K suffix', () => {
    expect(formatTokenCount(1_000)).toBe('1.0K');
    expect(formatTokenCount(1_500)).toBe('1.5K');
    expect(formatTokenCount(128_000)).toBe('128.0K');
    expect(formatTokenCount(999_999)).toBe('1000.0K');
  });

  it('formats millions with M suffix', () => {
    expect(formatTokenCount(1_000_000)).toBe('1.00M');
    expect(formatTokenCount(2_500_000)).toBe('2.50M');
  });
});
