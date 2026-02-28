'use client';

import type { TokenDataPoint } from '@/types/token-data';
import { formatUSD, calculateCost } from '@/lib/cost-calculator';

interface Props {
  latest: TokenDataPoint | null;
  sessionTotalCost: number;
  sessionCacheSavings: number;
}

export default function CostTracker({ latest, sessionTotalCost, sessionCacheSavings }: Props) {
  const breakdown = latest
    ? calculateCost(
        latest.model,
        latest.inputTokens,
        latest.outputTokens,
        latest.cacheCreationTokens,
        latest.cacheReadTokens,
      )
    : null;

  return (
    <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Cost Tracker</h2>
        <span className="text-xs text-slate-500">USD estimates</span>
      </div>

      {/* Session total */}
      <div className="rounded-xl bg-indigo-500/10 border border-indigo-500/20 p-4 mb-4">
        <div className="flex items-baseline gap-2">
          <span className="text-3xl font-bold text-indigo-400 font-mono">
            {formatUSD(sessionTotalCost)}
          </span>
          <span className="text-sm text-slate-400">session total</span>
        </div>
        {sessionCacheSavings > 0 && (
          <p className="text-xs text-teal-400 mt-1">
            Saved {formatUSD(sessionCacheSavings)} via cache
          </p>
        )}
      </div>

      {/* Last request breakdown */}
      <h3 className="text-sm font-medium text-slate-400 mb-3">Last Request Breakdown</h3>
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Input cost</span>
          <span className="text-teal-400 font-mono">
            {breakdown ? formatUSD(breakdown.inputCost) : '--'}
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Output cost</span>
          <span className="text-indigo-400 font-mono">
            {breakdown ? formatUSD(breakdown.outputCost) : '--'}
          </span>
        </div>
        {breakdown && breakdown.cacheWriteCost > 0 && (
          <div className="flex justify-between text-sm">
            <span className="text-slate-400">Cache write cost</span>
            <span className="text-orange-400 font-mono">{formatUSD(breakdown.cacheWriteCost)}</span>
          </div>
        )}
        {breakdown && breakdown.cacheReadCost > 0 && (
          <div className="flex justify-between text-sm">
            <span className="text-slate-400">Cache read cost</span>
            <span className="text-amber-400 font-mono">{formatUSD(breakdown.cacheReadCost)}</span>
          </div>
        )}
        <div className="pt-2 border-t border-white/10 flex justify-between text-sm font-medium">
          <span className="text-white">Request total</span>
          <span className="text-white font-mono">
            {breakdown ? formatUSD(breakdown.totalCost) : '--'}
          </span>
        </div>
        {breakdown && breakdown.cacheSavings > 0 && (
          <div className="flex justify-between text-sm">
            <span className="text-teal-400">Cache savings</span>
            <span className="text-teal-400 font-mono">-{formatUSD(breakdown.cacheSavings)}</span>
          </div>
        )}
      </div>
    </div>
  );
}
