'use client';

import { useState, useEffect, useLayoutEffect, useRef, useCallback } from 'react';

interface HeatmapDataPoint {
  hour: number;
  dayOfWeek: number;
  requestCount: number;
  totalTokens: number;
}

const RANGES = ['7d', '30d'];
const DAY_LABELS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

export default function HourlyHeatmap() {
  const [data, setData] = useState<HeatmapDataPoint[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [range, setRange] = useState('7d');
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const isVisibleRef = useRef(true);
  const mountedRef = useRef(true);

  const fetchData = useCallback(async () => {
    try {
      const res = await fetch(`/api/analytics/heatmap?range=${range}`, { cache: 'no-store' });
      if (!mountedRef.current) return;
      if (!res.ok) throw new Error('Failed to fetch');
      const json = (await res.json()) as { data: HeatmapDataPoint[] };
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

  const secondsAgo = lastUpdated ? Math.floor((Date.now() - lastUpdated.getTime()) / 1000) : 0;

  const buildGrid = (points: HeatmapDataPoint[]) => {
    const grid: number[][] = Array.from({ length: 7 }, () => Array(24).fill(0));
    points.forEach((p) => {
      if (p.dayOfWeek >= 0 && p.dayOfWeek < 7 && p.hour >= 0 && p.hour < 24) {
        grid[p.dayOfWeek][p.hour] = p.requestCount;
      }
    });
    return grid;
  };

  const getMaxCount = (points: HeatmapDataPoint[]) =>
    points.length > 0 ? Math.max(...points.map((p) => p.requestCount), 1) : 1;

  const getColor = (intensity: number) => {
    if (intensity === 0) return 'rgba(255,255,255,0.04)';
    const alpha = 0.15 + intensity * 0.85;
    return `rgba(13, 148, 136, ${alpha})`;
  };

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <div className="flex items-center justify-between mb-2">
        <h2 className="text-xl font-heading font-semibold text-white tracking-tight">Hourly Heatmap</h2>
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
        <div className="h-48 flex items-center justify-center text-slate-400">Loading...</div>
      )}

      {!loading && error && (
        <div className="h-48 flex items-center justify-center text-red-400">Error: {error}</div>
      )}

      {!loading && !error && data && data.length === 0 && (
        <div className="h-48 flex items-center justify-center text-slate-400">No data available</div>
      )}

      {!loading && !error && data && data.length > 0 && (
        <>
          {/* Hour labels */}
          <div className="flex mb-1 ml-8">
            {Array.from({ length: 24 }, (_, h) => (
              <div key={h} className="flex-1 text-center text-xs text-slate-500">
                {h}
              </div>
            ))}
          </div>

          {/* Grid with day labels */}
          <div data-testid="heatmap-grid">
            {(() => {
              const grid = buildGrid(data);
              const maxCount = getMaxCount(data);
              return DAY_LABELS.map((day, dayIdx) => (
                <div key={day} className="flex items-center mb-0.5">
                  <div className="w-8 text-xs text-slate-500 text-right pr-1">{day}</div>
                  {Array.from({ length: 24 }, (_, hour) => {
                    const count = grid[dayIdx][hour];
                    const intensity = count / maxCount;
                    return (
                      <div
                        key={hour}
                        data-testid="heatmap-cell"
                        data-intensity={intensity}
                        className="flex-1 h-4 rounded-sm mx-px"
                        style={{ backgroundColor: getColor(intensity) }}
                        title={`${day} ${hour}:00 — ${count} requests`}
                      />
                    );
                  })}
                </div>
              ));
            })()}
          </div>

          {/* Legend */}
          <div
            data-testid="heatmap-legend"
            className="flex items-center gap-2 mt-3 justify-end"
          >
            <span className="text-xs text-slate-500">Less</span>
            {[0, 0.25, 0.5, 0.75, 1].map((v) => (
              <div
                key={v}
                className="w-3 h-3 rounded-sm"
                style={{ backgroundColor: getColor(v) }}
              />
            ))}
            <span className="text-xs text-slate-500">More</span>
          </div>
        </>
      )}
    </div>
  );
}
