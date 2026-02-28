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
      <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
        <h2 className="text-lg font-semibold text-white mb-4">Cache Efficiency</h2>
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
      <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
        <h2 className="text-lg font-semibold text-white mb-4">Cache Efficiency</h2>
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

  return (
    <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Cache Efficiency</h2>
        <div className="text-right">
          <span className="text-2xl font-bold text-teal-400 font-mono">
            {cacheHitRate.toFixed(1)}%
          </span>
          <p className="text-xs text-slate-500">hit rate</p>
        </div>
      </div>

      <div className="h-40">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              cx="50%"
              cy="50%"
              innerRadius={40}
              outerRadius={65}
              dataKey="value"
              paddingAngle={2}
            >
              {data.map((entry, index) => (
                <Cell key={`cell-${index}`} fill={entry.color} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: '#0f172a',
                border: '1px solid rgba(255,255,255,0.1)',
                borderRadius: '8px',
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
        <div className="flex justify-between">
          <span className="text-slate-500">Cache reads</span>
          <span className="text-teal-400 font-mono">{formatTokenCount(totalCacheReadTokens)}</span>
        </div>
        <div className="flex justify-between">
          <span className="text-slate-500">Cache writes</span>
          <span className="text-indigo-400 font-mono">{formatTokenCount(totalCacheCreationTokens)}</span>
        </div>
      </div>
    </div>
  );
}
