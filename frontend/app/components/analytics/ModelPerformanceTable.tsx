'use client';

import { useState, useEffect, useLayoutEffect, useRef, useCallback } from 'react';

interface ModelData {
  model: string;
  provider: string;
  requestCount: number;
  totalTokens: number;
  avgTokensPerRequest: number;
  totalCostUSD: number;
  avgCostPerRequest: number;
  avgLatencyMs: number;
  cacheHitRate: number;
}

const RANGES = ['24h', '7d', '30d'];

type SortKey = keyof ModelData | null;
type SortDir = 'asc' | 'desc';

export default function ModelPerformanceTable() {
  const [data, setData] = useState<ModelData[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [range, setRange] = useState('24h');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [sortKey, setSortKey] = useState<SortKey>(null);
  const [sortDir, setSortDir] = useState<SortDir>('asc');
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const isVisibleRef = useRef(true);
  const mountedRef = useRef(true);

  const fetchData = useCallback(async () => {
    try {
      const res = await fetch(`/api/analytics/models?range=${range}`, { cache: 'no-store' });
      if (!mountedRef.current) return;
      if (!res.ok) throw new Error('Failed to fetch');
      const json = (await res.json()) as { data: ModelData[] };
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

  const handleSort = (key: keyof ModelData) => {
    if (sortKey === key) {
      setSortDir((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortKey(key);
      setSortDir('asc');
    }
  };

  const sortedData = data
    ? [...data].sort((a, b) => {
        if (!sortKey) return 0;
        const aVal = a[sortKey] as number;
        const bVal = b[sortKey] as number;
        return sortDir === 'asc' ? aVal - bVal : bVal - aVal;
      })
    : [];

  const secondsAgo = lastUpdated ? Math.floor((Date.now() - lastUpdated.getTime()) / 1000) : 0;

  const thClass = 'px-3 py-2 text-left text-xs font-medium text-slate-400 uppercase tracking-wider';
  const sortableThClass = `${thClass} cursor-pointer hover:text-white select-none`;
  const tdClass = 'px-3 py-2 text-sm text-slate-300 whitespace-nowrap';

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Model Performance</h2>
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
        <div className="h-32 flex items-center justify-center text-slate-400">Loading...</div>
      )}

      {!loading && error && (
        <div className="h-32 flex items-center justify-center text-red-400">Error: {error}</div>
      )}

      {!loading && !error && data && data.length === 0 && (
        <div className="h-32 flex items-center justify-center text-slate-400">No data available</div>
      )}

      {!loading && !error && data && data.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-white/5">
          <table className="w-full">
            <thead>
              <tr className="border-b border-white/10 bg-white/3">
                <th className={thClass}>Name</th>
                <th className={thClass}>Provider</th>
                <th
                  className={sortableThClass}
                  onClick={() => handleSort('totalTokens')}
                  aria-sort={
                    sortKey === 'totalTokens'
                      ? sortDir === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : undefined
                  }
                >
                  Tokens {sortKey === 'totalTokens' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
                </th>
                <th
                  className={sortableThClass}
                  onClick={() => handleSort('requestCount')}
                  aria-sort={
                    sortKey === 'requestCount'
                      ? sortDir === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : undefined
                  }
                >
                  Requests {sortKey === 'requestCount' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
                </th>
                <th
                  className={sortableThClass}
                  onClick={() => handleSort('totalCostUSD')}
                  aria-sort={
                    sortKey === 'totalCostUSD'
                      ? sortDir === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : undefined
                  }
                >
                  Cost {sortKey === 'totalCostUSD' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
                </th>
                <th
                  className={sortableThClass}
                  onClick={() => handleSort('avgLatencyMs')}
                  aria-sort={
                    sortKey === 'avgLatencyMs'
                      ? sortDir === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : undefined
                  }
                >
                  Latency {sortKey === 'avgLatencyMs' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
                </th>
                <th
                  className={sortableThClass}
                  onClick={() => handleSort('cacheHitRate')}
                  aria-sort={
                    sortKey === 'cacheHitRate'
                      ? sortDir === 'asc'
                        ? 'ascending'
                        : 'descending'
                      : undefined
                  }
                >
                  Cache % {sortKey === 'cacheHitRate' ? (sortDir === 'asc' ? '↑' : '↓') : ''}
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedData.map((row) => (
                <tr key={row.model} className="border-b border-white/5 hover:bg-white/5 transition-colors duration-150">
                  <td className={`${tdClass} font-mono text-xs`}>{row.model}</td>
                  <td className={tdClass}>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full bg-teal-500/10 border border-teal-500/20 text-teal-400 text-xs">
                      {row.provider}
                    </span>
                  </td>
                  <td className={`${tdClass} font-mono tabular-nums`}>{row.totalTokens.toLocaleString()}</td>
                  <td className={`${tdClass} font-mono tabular-nums`}>{row.requestCount}</td>
                  <td className={`${tdClass} font-mono tabular-nums text-indigo-400`}>${row.totalCostUSD.toFixed(4)}</td>
                  <td className={`${tdClass} font-mono tabular-nums`}>{row.avgLatencyMs}ms</td>
                  <td className={`${tdClass} font-mono tabular-nums text-teal-400`}>{row.cacheHitRate}%</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
