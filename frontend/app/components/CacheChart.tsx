'use client';

import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import type { SessionStats } from '@/types/token-data';
import { formatTokenCount } from '@/lib/token-calculator';

interface Props {
  sessionStats: SessionStats;
  supportsCache: boolean;
}

const COLORS = {
  hit: '#0D9488',   // teal
  miss: '#4F46E5',  // indigo
  creation: '#F59E0B', // amber
};

export default function CacheChart({ sessionStats, supportsCache }: Props) {
  if (!supportsCache) {
    return (
      <div className="rounded-2xl glass-card-hover p-6">
        <h2 className="text-xl font-heading font-semibold text-white mb-4 tracking-tight">Cache Efficiency</h2>
        <p className="text-sm text-slate-500 text-center py-8">
          Cache tracking not supported for this provider.
        </p>
      </div>
    );
  }

  const { totalCacheReadTokens, totalCacheCreationTokens, cacheHitRate } = sessionStats;
  const totalCacheActivity = totalCacheReadTokens + totalCacheCreationTokens;

  if (totalCacheActivity === 0) {
    return (
      <div className="rounded-2xl glass-card-hover p-6">
        <h2 className="text-xl font-heading font-semibold text-white mb-4 tracking-tight">Cache Efficiency</h2>
        <p className="text-sm text-slate-500 text-center py-8">
          No cache activity yet. Make some requests to see cache metrics.
        </p>
      </div>
    );
  }

  const data = [
    { name: 'Cache Hit', value: totalCacheReadTokens, color: COLORS.hit },
    { name: 'Cache Miss', value: totalCacheCreationTokens, color: COLORS.miss },
  ].filter((d) => d.value > 0);

  // Use neon cyan for high cache hit rates
  const hitRateColor = cacheHitRate > 50 ? 'text-neon-cyan' : 'text-teal-400';

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Cache Efficiency</h2>
        <div className="text-right">
          <span className={`text-2xl font-bold ${hitRateColor} font-mono tabular-nums`}>
            {cacheHitRate.toFixed(1)}%
          </span>
          <p className="text-xs text-slate-500">hit rate</p>
        </div>
      </div>

      <div className="h-40">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <defs>
              <linearGradient id="hitGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#14B8A6" stopOpacity={1} />
                <stop offset="100%" stopColor="#0F766E" stopOpacity={0.85} />
              </linearGradient>
              <linearGradient id="missGradient" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="#6366F1" stopOpacity={1} />
                <stop offset="100%" stopColor="#4338CA" stopOpacity={0.85} />
              </linearGradient>
            </defs>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              innerRadius={40}
              outerRadius={65}
              dataKey="value"
              paddingAngle={2}
              isAnimationActive={false}
            >
              {data.map((entry, index) => (
                <Cell
                  key={`cell-${index}`}
                  fill={`url(#${entry.name === 'Cache Hit' ? 'hitGradient' : 'missGradient'})`}
                  opacity={0.95}
                />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'rgba(15, 23, 42, 0.95)',
                backdropFilter: 'blur(8px)',
                border: '1px solid rgba(255,255,255,0.1)',
                borderRadius: '12px',
                padding: '12px',
                boxShadow: '0 4px 6px rgba(0,0,0,0.3)',
                color: '#e2e8f0',
                fontSize: '12px',
              }}
              formatter={(value: number | undefined) => [formatTokenCount(value ?? 0), '']}
            />
            <Legend
              iconType="circle"
              iconSize={8}
              formatter={(value) => (
                <span className="text-xs text-slate-400">{value}</span>
              )}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-2 grid grid-cols-2 gap-2 text-xs">
        <div className="flex justify-between px-3 py-2 rounded-lg bg-teal-500/5 border border-teal-500/10">
          <span className="text-slate-500">Cache reads</span>
          <span className="text-teal-400 font-mono tabular-nums">{formatTokenCount(totalCacheReadTokens)}</span>
        </div>
        <div className="flex justify-between px-3 py-2 rounded-lg bg-indigo-500/5 border border-indigo-500/10">
          <span className="text-slate-500">Cache writes</span>
          <span className="text-indigo-400 font-mono tabular-nums">{formatTokenCount(totalCacheCreationTokens)}</span>
        </div>
      </div>
    </div>
  );
}
