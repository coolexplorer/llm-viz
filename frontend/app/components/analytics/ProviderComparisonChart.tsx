'use client';

import { useState, useEffect, useLayoutEffect, useRef, useCallback, useMemo } from 'react';
import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
  ResponsiveContainer,
} from 'recharts';

interface ProviderData {
  provider: string;
  totalTokens: number;
  totalCost: number;
  requestCount: number;
  avgLatencyMs: number;
}

const RANGES = ['24h', '7d', '30d'];

const PROVIDER_COLORS: Record<string, string> = {
  anthropic: '#0D9488',
  openai: '#4F46E5',
  gemini: '#F59E0B',
  mistral: '#10B981',
  groq: '#8B5CF6',
  openrouter: '#EC4899',
};

type Metric = 'tokens' | 'cost' | 'requests';

export default function ProviderComparisonChart() {
  const [data, setData] = useState<ProviderData[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [range, setRange] = useState('24h');
  const [metric, setMetric] = useState<Metric>('tokens');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const isVisibleRef = useRef(true);
  const mountedRef = useRef(true);

  const fetchData = useCallback(async () => {
    try {
      const res = await fetch(`/api/analytics/providers?range=${range}`, { cache: 'no-store' });
      if (!mountedRef.current) return;
      if (!res.ok) throw new Error('Failed to fetch');
      const json = (await res.json()) as { data: ProviderData[] };
      if (!mountedRef.current) return;
      setData(json.data);
      setLastUpdated(new Date());
      setError(null);
    } catch (err) {
      if (!mountedRef.current) return;
      setError(err instanceof Error ? err.message : 'Error fetching data');
    } finally {
      if (mountedRef.current) setLoading(false);
    }
  }, [range]);

  useLayoutEffect(() => {
    setLoading(true);
    void fetchData();
  }, [fetchData]);

  useEffect(() => {
    mountedRef.current = true;

    intervalRef.current = setInterval(() => {
      if (isVisibleRef.current) void fetchData();
    }, 10000);

    const handleVisibility = () => {
      isVisibleRef.current = document.visibilityState === 'visible';
      if (isVisibleRef.current) void fetchData();
    };

    document.addEventListener('visibilitychange', handleVisibility);

    return () => {
      mountedRef.current = false;
      if (intervalRef.current) clearInterval(intervalRef.current);
      document.removeEventListener('visibilitychange', handleVisibility);
    };
  }, [fetchData]);

  const providers = useMemo(() => data?.map((d) => d.provider) ?? [], [data]);

  const chartData = useMemo(() => {
    if (!data || data.length === 0) return [];
    const entry: Record<string, number> = {};
    data.forEach((d) => {
      entry[d.provider] =
        metric === 'cost'
          ? d.totalCost
          : metric === 'requests'
            ? d.requestCount
            : d.totalTokens;
    });
    return [entry];
  }, [data, metric]);

  const secondsAgo = lastUpdated ? Math.floor((Date.now() - lastUpdated.getTime()) / 1000) : 0;

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Provider Comparison</h2>
        <div className="flex gap-1">
          {RANGES.map((r) => (
            <button
              key={r}
              onClick={() => setRange(r)}
              className={`text-xs px-3 py-1 rounded-full transition-all duration-300 ${
                range === r
                  ? 'bg-teal-500/20 border border-teal-500/30 text-teal-400'
                  : 'text-slate-400 hover:text-white hover:bg-white/5'
              }`}
            >
              {r}
            </button>
          ))}
        </div>
      </div>

      <div className="flex gap-1.5 mb-3">
        {(['tokens', 'cost', 'requests'] as Metric[]).map((m) => (
          <button
            key={m}
            onClick={() => setMetric(m)}
            className={`text-xs px-3 py-1 rounded-full transition-all duration-300 capitalize ${
              metric === m
                ? 'bg-indigo-500/20 border border-indigo-500/30 text-indigo-400'
                : 'text-slate-400 hover:text-white hover:bg-white/5'
            }`}
          >
            {m}
          </button>
        ))}
      </div>

      {lastUpdated && (
        <p className="text-xs text-slate-500 mb-3">Last updated: {secondsAgo} seconds ago</p>
      )}

      {loading && (
        <div className="h-64 flex items-center justify-center text-slate-400">Loading...</div>
      )}

      {!loading && error && (
        <div className="h-64 flex items-center justify-center text-red-400">Error: {error}</div>
      )}

      {!loading && !error && data && data.length === 0 && (
        <div className="h-64 flex items-center justify-center text-slate-400">No data available</div>
      )}

      {!loading && !error && data && data.length > 0 && (
        <>
          <div className="flex gap-3 mb-2">
            {providers.map((p) => (
              <span key={p} className="text-xs" style={{ color: PROVIDER_COLORS[p] ?? '#888' }}>
                {p}
              </span>
            ))}
          </div>
          <div className="h-52">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={chartData} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
                <XAxis tick={{ fill: '#94a3b8', fontSize: 10 }} />
                <YAxis tick={{ fill: '#94a3b8', fontSize: 10 }} />
                <Tooltip />
                <Legend />
                {providers.map((p) => (
                  <Bar
                    key={p}
                    dataKey={p}
                    fill={PROVIDER_COLORS[p] ?? '#888'}
                    isAnimationActive={false}
                  />
                ))}
              </BarChart>
            </ResponsiveContainer>
          </div>
        </>
      )}
    </div>
  );
}
