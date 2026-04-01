'use client';

import { useState, useEffect } from 'react';
import type { TokenDataPoint } from '@/types/token-data';
import { formatTokenCount } from '@/lib/token-calculator';

interface Props {
  latest: TokenDataPoint | null;
  totalTokens: number;
}

interface StatCardProps {
  label: string;
  value: string;
  color: string;
  bgColor: string;
  borderColor: string;
  animated?: boolean;
}

function StatCard({ label, value, color, bgColor, borderColor, animated }: StatCardProps) {
  return (
    <div className={`rounded-xl ${bgColor} border ${borderColor} p-4 flex flex-col gap-1 transition-all duration-300 hover:scale-[1.02]`}>
      <span className="text-xs font-medium text-slate-400 uppercase tracking-wide">{label}</span>
      <span className={`text-2xl font-bold font-mono tabular-nums ${color} ${animated ? 'animate-count-up' : ''}`}>
        {value}
      </span>
    </div>
  );
}

export default function TokenCounter({ latest, totalTokens }: Props) {
  const [isUpdating, setIsUpdating] = useState(false);

  useEffect(() => {
    if (!latest) return;
    const startTimer = setTimeout(() => setIsUpdating(true), 0);
    const endTimer = setTimeout(() => setIsUpdating(false), 300);
    return () => {
      clearTimeout(startTimer);
      clearTimeout(endTimer);
    };
  }, [latest]);

  const inputTokens = latest?.inputTokens ?? 0;
  const outputTokens = latest?.outputTokens ?? 0;
  const cacheReadTokens = latest?.cacheReadTokens ?? 0;
  const cacheCreationTokens = latest?.cacheCreationTokens ?? 0;
  const requestTotal = inputTokens + outputTokens + cacheReadTokens + cacheCreationTokens;

  // Use neon glow for cache read (success state)
  const cacheReadColor = cacheReadTokens > 0 ? 'text-success-glow' : 'text-amber-400';

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      {/* ARIA live region for screen readers */}
      <div role="status" aria-live="polite" aria-atomic="true" className="sr-only">
        {latest && `New token data: ${inputTokens} input, ${outputTokens} output, ${cacheReadTokens} cache read tokens`}
      </div>

      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Token Counter</h2>
        <span className="inline-flex items-center px-2.5 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-slate-400">
          Last request
        </span>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <StatCard
          label="Input"
          value={formatTokenCount(inputTokens)}
          color="text-teal-400"
          bgColor="bg-teal-500/10"
          borderColor="border-teal-500/20"
          animated={isUpdating}
        />
        <StatCard
          label="Output"
          value={formatTokenCount(outputTokens)}
          color="text-indigo-400"
          bgColor="bg-indigo-500/10"
          borderColor="border-indigo-500/20"
          animated={isUpdating}
        />
        <StatCard
          label="Cache Read"
          value={formatTokenCount(cacheReadTokens)}
          color={cacheReadColor}
          bgColor="bg-amber-500/10"
          borderColor="border-amber-500/20"
          animated={isUpdating}
        />
        <StatCard
          label="Cache Write"
          value={formatTokenCount(cacheCreationTokens)}
          color="text-orange-400"
          bgColor="bg-orange-500/10"
          borderColor="border-orange-500/20"
          animated={isUpdating}
        />
      </div>

      <div className="mt-4 pt-4 border-t border-white/10 flex items-center justify-between">
        <div className="flex flex-col">
          <span className="text-xs text-slate-500">This request</span>
          <span className="text-xl font-bold text-white font-mono tabular-nums">
            {formatTokenCount(requestTotal)}
          </span>
        </div>
        <div className="flex flex-col text-right">
          <span className="text-xs text-slate-500">Session total</span>
          <span className="text-xl font-bold text-slate-300 font-mono tabular-nums">
            {formatTokenCount(totalTokens)}
          </span>
        </div>
      </div>
    </div>
  );
}
