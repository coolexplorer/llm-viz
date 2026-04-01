'use client';

import { useState, useEffect } from 'react';
import type { ContextWindowStatus } from '@/types/token-data';
import { formatTokenCount } from '@/lib/token-calculator';

interface Props {
  status: ContextWindowStatus | null;
}

export default function ContextGauge({ status }: Props) {
  const [utilizationChanged, setUtilizationChanged] = useState(false);

  const pct = status ? Math.min(100, status.utilizationPercent) : 0;
  const isWarning = status?.isWarning ?? false;
  const isCritical = status?.isCritical ?? false;

  useEffect(() => {
    const startTimer = setTimeout(() => setUtilizationChanged(true), 0);
    const endTimer = setTimeout(() => setUtilizationChanged(false), 1000);
    return () => {
      clearTimeout(startTimer);
      clearTimeout(endTimer);
    };
  }, [pct]);

  const barColor = isCritical
    ? 'bg-red-500'
    : isWarning
    ? 'bg-amber-500'
    : 'bg-gradient-to-r from-teal-500 to-teal-400';

  const arcColor = isCritical ? '#ef4444' : isWarning ? '#f59e0b' : undefined;
  const arcGradientId = isCritical ? undefined : isWarning ? undefined : 'tealGradient';

  const pctColor = isCritical
    ? 'text-red-400'
    : isWarning
    ? 'text-amber-400'
    : 'text-teal-400';

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Context Window</h2>
        {status && (
          <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-slate-400 font-mono">
            {status.model}
          </span>
        )}
      </div>

      {/* Circular gauge */}
      <div className="flex flex-col items-center mb-4">
        <div
          className={`relative w-32 h-32 ${utilizationChanged ? 'animate-pulse-glow' : ''}`}
          role="img"
          aria-label={`Context window utilization: ${pct.toFixed(1)}% used`}
        >
          <svg className="w-32 h-32 -rotate-90" viewBox="0 0 128 128">
            <defs>
              <linearGradient id="tealGradient" x1="0%" y1="0%" x2="100%" y2="0%">
                <stop offset="0%" stopColor="#0D9488" />
                <stop offset="100%" stopColor="#14B8A6" />
              </linearGradient>
              <filter id="gaugeGlow">
                <feGaussianBlur stdDeviation="3" result="coloredBlur"/>
                <feMerge>
                  <feMergeNode in="coloredBlur"/>
                  <feMergeNode in="SourceGraphic"/>
                </feMerge>
              </filter>
            </defs>
            {/* Background track */}
            <circle
              cx="64"
              cy="64"
              r="52"
              fill="none"
              stroke="rgb(30 41 59)"
              strokeWidth="12"
            />
            {/* Progress arc */}
            <circle
              cx="64"
              cy="64"
              r="52"
              fill="none"
              stroke={arcColor ?? `url(#${arcGradientId})`}
              strokeWidth="12"
              strokeLinecap="round"
              strokeDasharray={`${2 * Math.PI * 52}`}
              strokeDashoffset={`${2 * Math.PI * 52 * (1 - pct / 100)}`}
              className="transition-all duration-500"
              filter="url(#gaugeGlow)"
            />
          </svg>
          <div className="absolute inset-0 flex flex-col items-center justify-center">
            <span className={`text-2xl font-bold font-mono tabular-nums ${pctColor}`}>
              {pct.toFixed(1)}%
            </span>
            <span className="text-xs text-slate-500">used</span>
          </div>
        </div>
      </div>

      {/* Stats */}
      <div className="space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Used</span>
          <span className="text-white font-mono tabular-nums">
            {formatTokenCount(status?.currentUsed ?? 0)}
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Remaining</span>
          <span className="text-white font-mono tabular-nums">
            {formatTokenCount(status?.remainingTokens ?? 0)}
          </span>
        </div>
        <div className="flex justify-between text-sm">
          <span className="text-slate-400">Max capacity</span>
          <span className="text-white font-mono tabular-nums">
            {formatTokenCount(status?.maxTokens ?? 0)}
          </span>
        </div>
      </div>

      {/* Progress bar */}
      <div className="mt-4 h-2 rounded-full bg-slate-800 overflow-hidden">
        <div
          className={`h-full rounded-full bar-grow ${barColor}`}
          style={{ width: `${pct}%` }}
        />
      </div>

      {/* Warnings */}
      {isCritical && (
        <p className="mt-3 text-xs text-red-400 font-medium">
          Critical: Context window nearly full (&gt;95%)
        </p>
      )}
      {isWarning && !isCritical && (
        <p className="mt-3 text-xs text-amber-400">
          Warning: High context usage (&gt;80%)
        </p>
      )}
    </div>
  );
}
