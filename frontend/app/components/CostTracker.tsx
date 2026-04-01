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

  // Use alert color for high costs
  const requestCostHigh = breakdown && breakdown.totalCost > 0.01;
  const requestCostColor = requestCostHigh ? 'text-cost-alert' : 'text-white';

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Cost Tracker</h2>
        <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-indigo-500/10 border border-indigo-500/20 text-indigo-400 text-xs font-medium">
          USD estimates
        </span>
      </div>

      {/* Session total */}
      <div className="rounded-xl bg-indigo-500/10 border border-indigo-500/20 p-4 mb-4 relative overflow-hidden transition-all duration-300 hover:shadow-[0_0_30px_rgba(79,70,229,0.3)]">
        <div className="absolute inset-0 bg-live-gradient opacity-10" />
        <div className="relative z-10">
          <div className="flex items-baseline gap-2">
            <span className="text-3xl font-bold text-indigo-400 font-mono tabular-nums">
              {formatUSD(sessionTotalCost)}
            </span>
            <span className="text-sm text-slate-400">session total</span>
          </div>
          {sessionCacheSavings > 0 && (
            <p className="text-xs text-success-glow mt-1">
              Saved {formatUSD(sessionCacheSavings)} via cache
            </p>
          )}
        </div>
      </div>

      {/* Last request breakdown */}
      <h3 className="text-sm font-medium text-slate-400 mb-3">Last Request Breakdown</h3>
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Input cost</span>
          <span className="text-teal-400 font-mono tabular-nums">
            {breakdown ? formatUSD(breakdown.inputCost) : '--'}
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Output cost</span>
          <span className="text-indigo-400 font-mono tabular-nums">
            {breakdown ? formatUSD(breakdown.outputCost) : '--'}
          </span>
        </div>
        {breakdown && breakdown.cacheWriteCost > 0 && (
          <div className="flex justify-between text-sm">
            <span className="text-slate-400">Cache write cost</span>
            <span className="text-orange-400 font-mono tabular-nums">{formatUSD(breakdown.cacheWriteCost)}</span>
          </div>
        )}
        {breakdown && breakdown.cacheReadCost > 0 && (
          <div className="flex justify-between text-sm">
            <span className="text-slate-400">Cache read cost</span>
            <span className="text-amber-400 font-mono tabular-nums">{formatUSD(breakdown.cacheReadCost)}</span>
          </div>
        )}
        <div className="pt-2 border-t border-white/10 flex justify-between text-sm font-medium">
          <span className="text-white">Request total</span>
          <span className={`${requestCostColor} font-mono tabular-nums`}>
            {breakdown ? formatUSD(breakdown.totalCost) : '--'}
          </span>
        </div>
        {breakdown && breakdown.cacheSavings > 0 && (
          <div className="flex justify-between text-sm">
            <span className="text-teal-400">Cache savings</span>
            <span className="text-teal-400 font-mono tabular-nums">-{formatUSD(breakdown.cacheSavings)}</span>
          </div>
        )}
      </div>
    </div>
  );
}
