'use client';

import { useMemo } from 'react';
import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { TokenDataPoint } from '@/types/token-data';
import { EmptyState } from './EmptyState';

interface Props {
  data: TokenDataPoint[];
}

function formatTime(ts: number) {
  return new Date(ts).toLocaleTimeString('en-US', {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false,
  });
}

function formatYAxis(value: number): string {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(0)}K`;
  return value.toString();
}

export default function UsageTimeline({ data }: Props) {
  const chartData = useMemo(
    () =>
      data.map((d) => ({
        time: formatTime(d.timestamp),
        ts: d.timestamp,
        input: d.inputTokens,
        output: d.outputTokens,
        cache: d.cacheReadTokens,
      })),
    [data],
  );

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Usage Timeline</h2>
        <span className="inline-flex items-center gap-1.5 px-3 py-1 rounded-full bg-white/5 border border-white/10 text-xs text-slate-400 font-mono tabular-nums">
          Last {data.length} requests
        </span>
      </div>

      {data.length === 0 ? (
        <EmptyState
          icon={
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
            </svg>
          }
          title="No Timeline Data"
          description="Make a request to see token usage over time"
        />
      ) : (
        <div className="h-64" role="img" aria-label="Token usage timeline chart showing input, output, and cache tokens over time">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={chartData} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
              <defs>
                <linearGradient id="inputGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#0D9488" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#0D9488" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="outputGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#4F46E5" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#4F46E5" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="cacheGradient" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#F59E0B" stopOpacity={0.2} />
                  <stop offset="95%" stopColor="#F59E0B" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.04)" />
              <XAxis
                dataKey="time"
                tick={{ fill: '#64748b', fontSize: 10 }}
                tickLine={false}
                axisLine={false}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={{ fill: '#64748b', fontSize: 10 }}
                tickLine={false}
                axisLine={false}
                tickFormatter={formatYAxis}
                width={40}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: 'rgba(15, 23, 42, 0.95)',
                  backdropFilter: 'blur(8px)',
                  border: '1px solid rgba(255,255,255,0.1)',
                  borderRadius: '12px',
                  padding: '12px',
                  boxShadow: '0 8px 16px rgba(0,0,0,0.3)',
                  color: '#e2e8f0',
                  fontSize: '12px',
                }}
                labelStyle={{ color: '#94a3b8', marginBottom: '4px' }}
                formatter={(value: number | undefined, name: string | undefined) => [
                  (value ?? 0).toLocaleString(),
                  (name ?? '').charAt(0).toUpperCase() + (name ?? '').slice(1),
                ]}
              />
              <Legend
                iconType="circle"
                iconSize={8}
                formatter={(value) => (
                  <span className="text-xs text-slate-400">
                    {value.charAt(0).toUpperCase() + value.slice(1)} tokens
                  </span>
                )}
              />
              <Area
                type="monotone"
                dataKey="input"
                stroke="#0D9488"
                strokeWidth={2}
                fill="url(#inputGradient)"
                dot={false}
                activeDot={{
                  r: 6,
                  fill: '#0D9488',
                  stroke: '#fff',
                  strokeWidth: 2,
                  style: { filter: 'drop-shadow(0 2px 4px rgba(13,148,136,0.5))' },
                }}
                isAnimationActive={false}
              />
              <Area
                type="monotone"
                dataKey="output"
                stroke="#4F46E5"
                strokeWidth={2}
                fill="url(#outputGradient)"
                dot={false}
                activeDot={{
                  r: 6,
                  fill: '#4F46E5',
                  stroke: '#fff',
                  strokeWidth: 2,
                  style: { filter: 'drop-shadow(0 2px 4px rgba(79,70,229,0.5))' },
                }}
                isAnimationActive={false}
              />
              <Area
                type="monotone"
                dataKey="cache"
                stroke="#F59E0B"
                strokeWidth={2}
                fill="url(#cacheGradient)"
                dot={false}
                activeDot={{
                  r: 6,
                  fill: '#F59E0B',
                  stroke: '#fff',
                  strokeWidth: 2,
                  style: { filter: 'drop-shadow(0 2px 4px rgba(245,158,11,0.5))' },
                }}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
