'use client';

import { useState, useEffect, useLayoutEffect, useRef, useCallback } from 'react';
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

interface CumulativeDataPoint {
  date: string;
  totalTokens: number;
  totalCost: number;
  requestCount: number;
}

const RANGES = ['24h', '7d', '30d'];

export default function CumulativeUsageChart() {
  const [data, setData] = useState<CumulativeDataPoint[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [range, setRange] = useState('24h');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const isVisibleRef = useRef(true);
  const mountedRef = useRef(true);

  const fetchData = useCallback(async () => {
    try {
      const res = await fetch(`/api/analytics/cumulative?range=${range}`, { cache: 'no-store' });
      if (!mountedRef.current) return;
      if (!res.ok) throw new Error('Failed to fetch');
      const json = (await res.json()) as { data: CumulativeDataPoint[] };
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

  // Use useLayoutEffect for immediate fetch so it fires synchronously during act()
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

  const secondsAgo = lastUpdated ? Math.floor((Date.now() - lastUpdated.getTime()) / 1000) : 0;

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Cumulative Usage</h2>
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
        <div className="h-64">
          <ResponsiveContainer width="100%" height="100%">
            <AreaChart data={data} margin={{ top: 5, right: 5, left: 0, bottom: 5 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="rgba(255,255,255,0.05)" />
              <XAxis dataKey="date" tick={{ fill: '#94a3b8', fontSize: 10 }} />
              <YAxis tick={{ fill: '#94a3b8', fontSize: 10 }} />
              <Tooltip />
              <Legend />
              <Area
                type="monotone"
                dataKey="totalTokens"
                stroke="#0D9488"
                fill="#0D9488"
                fillOpacity={0.2}
                isAnimationActive={false}
              />
              <Area
                type="monotone"
                dataKey="totalCost"
                stroke="#4F46E5"
                fill="#4F46E5"
                fillOpacity={0.2}
                isAnimationActive={false}
              />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      )}
    </div>
  );
}
