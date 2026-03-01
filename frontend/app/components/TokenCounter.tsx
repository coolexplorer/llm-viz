'use client';

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
}

function StatCard({ label, value, color, bgColor, borderColor }: StatCardProps) {
  return (
    <div className={`rounded-xl ${bgColor} border ${borderColor} p-4 flex flex-col gap-1`}>
      <span className="text-xs font-medium text-slate-400 uppercase tracking-wide">{label}</span>
      <span className={`text-2xl font-bold font-mono ${color}`}>{value}</span>
    </div>
  );
}

export default function TokenCounter({ latest, totalTokens }: Props) {
  const inputTokens = latest?.inputTokens ?? 0;
  const outputTokens = latest?.outputTokens ?? 0;
  const cacheReadTokens = latest?.cacheReadTokens ?? 0;
  const cacheCreationTokens = latest?.cacheCreationTokens ?? 0;
  const requestTotal = inputTokens + outputTokens + cacheReadTokens + cacheCreationTokens;

  return (
    <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold text-white">Token Counter</h2>
        <span className="text-xs text-slate-500">Last request</span>
      </div>

      <div className="grid grid-cols-2 gap-3">
        <StatCard
          label="Input"
          value={formatTokenCount(inputTokens)}
          color="text-teal-400"
          bgColor="bg-teal-500/10"
          borderColor="border-teal-500/20"
        />
        <StatCard
          label="Output"
          value={formatTokenCount(outputTokens)}
          color="text-indigo-400"
          bgColor="bg-indigo-500/10"
          borderColor="border-indigo-500/20"
        />
        <StatCard
          label="Cache Read"
          value={formatTokenCount(cacheReadTokens)}
          color="text-amber-400"
          bgColor="bg-amber-500/10"
          borderColor="border-amber-500/20"
        />
        <StatCard
          label="Cache Write"
          value={formatTokenCount(cacheCreationTokens)}
          color="text-orange-400"
          bgColor="bg-orange-500/10"
          borderColor="border-orange-500/20"
        />
      </div>

      <div className="mt-4 pt-4 border-t border-white/10 flex items-center justify-between">
        <div className="flex flex-col">
          <span className="text-xs text-slate-500">This request</span>
          <span className="text-xl font-bold text-white font-mono">
            {formatTokenCount(requestTotal)}
          </span>
        </div>
        <div className="flex flex-col text-right">
          <span className="text-xs text-slate-500">Session total</span>
          <span className="text-xl font-bold text-slate-300 font-mono">
            {formatTokenCount(totalTokens)}
          </span>
        </div>
      </div>
    </div>
  );
}
