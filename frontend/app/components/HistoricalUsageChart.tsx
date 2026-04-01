'use client';

import { useState, useEffect } from 'react';
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
import type { UsageHistoryResponse } from '@/types/usage-history';
import { formatTokenCount } from '@/lib/token-calculator';
import { LoadingSkeleton } from './LoadingSkeleton';
import { EmptyState } from './EmptyState';

interface Props {
  provider: string;
  apiKey: string;
}

type TimeRange = '7d' | '30d' | '90d';

export default function HistoricalUsageChart({ provider, apiKey }: Props) {
  const [timeRange, setTimeRange] = useState<TimeRange>('30d');
  const [data, setData] = useState<UsageHistoryResponse | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!apiKey) return;

    const fetchHistory = async () => {
      setIsLoading(true);
      setError(null);

      try {
        const endDate = new Date();
        const startDate = new Date();
        
        switch (timeRange) {
          case '7d':
            startDate.setDate(endDate.getDate() - 7);
            break;
          case '30d':
            startDate.setDate(endDate.getDate() - 30);
            break;
          case '90d':
            startDate.setDate(endDate.getDate() - 90);
            break;
        }

        const apiUrl = typeof window !== 'undefined'
          ? 'http://localhost:8080'
          : (process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080');

        const url = new URL(`${apiUrl}/api/usage/history`);
        url.searchParams.set('provider', provider);
        url.searchParams.set('api_key', apiKey);
        url.searchParams.set('start_date', startDate.toISOString().split('T')[0]);
        url.searchParams.set('end_date', endDate.toISOString().split('T')[0]);

        const res = await fetch(url.toString());
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}: ${await res.text()}`);
        }

        const history: UsageHistoryResponse = await res.json();
        setData(history);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch usage history');
      } finally {
        setIsLoading(false);
      }
    };

    fetchHistory();
  }, [provider, apiKey, timeRange]);

  const chartData = data?.data_points.map((point) => ({
    date: new Date(point.date).toLocaleDateString(),
    input: point.input_tokens,
    output: point.output_tokens,
    cost: point.cost_usd,
  })) ?? [];

  return (
    <div className="rounded-2xl glass-card-hover p-6 animate-fade-in-up stagger-7">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">
          Historical Usage
        </h2>
        <div className="flex items-center gap-2">
          {['7d', '30d', '90d'].map((range) => (
            <button
              key={range}
              onClick={() => setTimeRange(range as TimeRange)}
              className={`px-3 py-1 rounded-lg text-xs font-medium transition-all duration-300 ${
                timeRange === range
                  ? 'bg-teal-500/20 border border-teal-500/40 text-teal-400'
                  : 'bg-white/5 border border-white/10 text-slate-400 hover:bg-white/10'
              }`}
            >
              {range === '7d' && 'Last 7 Days'}
              {range === '30d' && 'Last 30 Days'}
              {range === '90d' && 'Last 90 Days'}
            </button>
          ))}
        </div>
      </div>

      {isLoading && <LoadingSkeleton />}

      {error && (
        <div className="rounded-xl bg-red-500/10 border border-red-500/20 px-4 py-3 text-sm text-red-400">
          {error}
        </div>
      )}

      {!isLoading && !error && chartData.length === 0 && (
        <EmptyState
          icon={
            <svg fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M3 13.125C3 12.504 3.504 12 4.125 12h2.25c.621 0 1.125.504 1.125 1.125v6.75C7.5 20.496 6.996 21 6.375 21h-2.25A1.125 1.125 0 013 19.875v-6.75zM9.75 8.625c0-.621.504-1.125 1.125-1.125h2.25c.621 0 1.125.504 1.125 1.125v11.25c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V8.625zM16.5 4.125c0-.621.504-1.125 1.125-1.125h2.25C20.496 3 21 3.504 21 4.125v15.75c0 .621-.504 1.125-1.125 1.125h-2.25a1.125 1.125 0 01-1.125-1.125V4.125z" />
            </svg>
          }
          title="No Historical Data"
          description="No usage data found for the selected time range"
        />
      )}

      {!isLoading && !error && chartData.length > 0 && (
        <>
          <div className="h-64" role="img" aria-label="Historical usage chart showing daily token consumption">
            <ResponsiveContainer width="100%" height="100%">
              <AreaChart data={chartData} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
                <defs>
                  <linearGradient id="inputHistGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#0D9488" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="#0D9488" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="outputHistGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#4F46E5" stopOpacity={0.5} />
                    <stop offset="95%" stopColor="#4F46E5" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.04)" />
                <XAxis
                  dataKey="date"
                  tick={{ fill: '#64748b', fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                />
                <YAxis
                  tick={{ fill: '#64748b', fontSize: 10 }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(value: number) => formatTokenCount(value)}
                  width={60}
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
                  formatter={(value: number | undefined) => [formatTokenCount(value ?? 0), '']}
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
                  fill="url(#inputHistGradient)"
                  isAnimationActive={false}
                />
                <Area
                  type="monotone"
                  dataKey="output"
                  stroke="#4F46E5"
                  strokeWidth={2}
                  fill="url(#outputHistGradient)"
                  isAnimationActive={false}
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>

          {data && (
            <div className="mt-4 grid grid-cols-2 gap-4">
              <div className="rounded-xl bg-teal-500/5 border border-teal-500/20 p-3">
                <span className="text-xs text-slate-400 block mb-1">Total Tokens</span>
                <span className="text-lg font-bold text-teal-400 font-mono tabular-nums">
                  {formatTokenCount(chartData.reduce((sum, d) => sum + d.input + d.output, 0))}
                </span>
              </div>
              <div className="rounded-xl bg-indigo-500/5 border border-indigo-500/20 p-3">
                <span className="text-xs text-slate-400 block mb-1">Total Cost</span>
                <span className="text-lg font-bold text-indigo-400 font-mono tabular-nums">
                  ${data.total_cost.toFixed(2)}
                </span>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
