import type { TokenDataPoint, SessionStats, ContextWindowStatus } from '@/types/token-data';
import { getModelLimit } from './model-limits';

export function calculateCacheHitRate(
  cacheReadTokens: number,
  cacheCreationTokens: number,
): number {
  const total = cacheReadTokens + cacheCreationTokens;
  if (total === 0) return 0;
  return (cacheReadTokens / total) * 100;
}

export function calculateContextUsage(
  inputTokens: number,
  cacheCreationTokens: number,
  cacheReadTokens: number,
  model: string,
): ContextWindowStatus {
  const maxTokens = getModelLimit(model);
  const currentUsed = inputTokens + cacheCreationTokens + cacheReadTokens;
  const utilizationPercent = (currentUsed / maxTokens) * 100;

  return {
    model,
    maxTokens,
    currentUsed,
    utilizationPercent,
    remainingTokens: maxTokens - currentUsed,
    isWarning: utilizationPercent > 80,
    isCritical: utilizationPercent > 95,
  };
}

export function aggregateSessionStats(dataPoints: TokenDataPoint[]): SessionStats {
  const totalInputTokens = dataPoints.reduce((sum, d) => sum + d.inputTokens, 0);
  const totalOutputTokens = dataPoints.reduce((sum, d) => sum + d.outputTokens, 0);
  const totalCacheReadTokens = dataPoints.reduce((sum, d) => sum + d.cacheReadTokens, 0);
  const totalCacheCreationTokens = dataPoints.reduce((sum, d) => sum + d.cacheCreationTokens, 0);

  return {
    totalRequests: dataPoints.length,
    totalInputTokens,
    totalOutputTokens,
    totalCacheReadTokens,
    totalCacheCreationTokens,
    totalCostUSD: dataPoints.reduce((sum, d) => sum + d.costUSD, 0),
    cacheHitRate: calculateCacheHitRate(totalCacheReadTokens, totalCacheCreationTokens),
  };
}

export function formatTokenCount(tokens: number): string {
  if (tokens >= 1_000_000) return `${(tokens / 1_000_000).toFixed(2)}M`;
  if (tokens >= 1_000) return `${(tokens / 1_000).toFixed(1)}K`;
  return tokens.toString();
}
