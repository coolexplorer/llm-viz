'use client';

import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';
import type { TokenDataPoint } from '@/types/token-data';

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
  const chartData = data.map((d) => ({
    time: formatTime(d.timestamp),
    ts: d.timestamp,
    input: d.inputTokens,
    output: d.outputTokens,
    cache: d.cacheReadTokens,
  }));

  return (
    <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Usage Timeline</h2>
        <span className="text-xs text-slate-500">Last {data.length} requests</span>
      </div>

      {data.length === 0 ? (
        <div className="h-64 flex items-center justify-center text-sm text-slate-500">
          No data yet. Make a request to see the timeline.
        </div>
      ) : (
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis
                dataKey="time"
                tick={{ fill: '#94a3b8', fontSize: 10 }}
                tickLine={false}
                axisLine={false}
                interval="preserveStartEnd"
              />
              <YAxis
                tick={{ fill: '#94a3b8', fontSize: 10 }}
                tickLine={false}
                axisLine={false}
                tickFormatter={formatYAxis}
                width={40}
              />
              <Tooltip
                contentStyle={{
                  backgroundColor: '#0f172a',
                  border: '1px solid rgba(255,255,255,0.1)',
                  borderRadius: '8px',
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
              <Line
                type="monotone"
                dataKey="input"
                stroke="#0D9488"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, fill: '#0D9488' }}
              />
              <Line
                type="monotone"
                dataKey="output"
                stroke="#4F46E5"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, fill: '#4F46E5' }}
              />
              <Line
                type="monotone"
                dataKey="cache"
                stroke="#F59E0B"
                strokeWidth={2}
                dot={false}
                activeDot={{ r: 4, fill: '#F59E0B' }}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
