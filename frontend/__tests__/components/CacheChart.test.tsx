import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import CacheChart from '@/app/components/CacheChart';
import type { SessionStats } from '@/types/token-data';

// Mock recharts to avoid SVG rendering issues in jsdom
vi.mock('recharts', () => ({
  PieChart: ({ children }: { children: React.ReactNode }) => <div data-testid="pie-chart">{children}</div>,
  Pie: () => <div data-testid="pie" />,
  Cell: () => null,
  Tooltip: () => null,
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Legend: () => null,
}));

function makeSessionStats(overrides: Partial<SessionStats> = {}): SessionStats {
  return {
    totalRequests: 10,
    totalInputTokens: 10_000,
    totalOutputTokens: 5_000,
    totalCacheReadTokens: 3_000,
    totalCacheCreationTokens: 1_000,
    totalCostUSD: 0.1,
    cacheHitRate: 75,
    ...overrides,
  };
}

describe('CacheChart', () => {
  it('shows unsupported message when supportsCache is false', () => {
    render(<CacheChart sessionStats={makeSessionStats()} supportsCache={false} />);
    expect(screen.getByText(/Cache tracking not supported/i)).toBeInTheDocument();
  });

  it('shows no cache activity message when all cache tokens are 0', () => {
    const stats = makeSessionStats({ totalCacheReadTokens: 0, totalCacheCreationTokens: 0 });
    render(<CacheChart sessionStats={stats} supportsCache={true} />);
    expect(screen.getByText(/No cache activity yet/i)).toBeInTheDocument();
  });

  it('shows cache hit rate when cache activity exists', () => {
    const stats = makeSessionStats({ cacheHitRate: 75 });
    render(<CacheChart sessionStats={stats} supportsCache={true} />);
    expect(screen.getByText('75.0%')).toBeInTheDocument();
  });

  it('shows "hit rate" label', () => {
    const stats = makeSessionStats();
    render(<CacheChart sessionStats={stats} supportsCache={true} />);
    expect(screen.getByText('hit rate')).toBeInTheDocument();
  });

  it('shows cache reads count', () => {
    const stats = makeSessionStats({ totalCacheReadTokens: 3_000 });
    render(<CacheChart sessionStats={stats} supportsCache={true} />);
    expect(screen.getByText('3.0K')).toBeInTheDocument();
  });

  it('shows cache writes count', () => {
    const stats = makeSessionStats({ totalCacheCreationTokens: 1_000 });
    render(<CacheChart sessionStats={stats} supportsCache={true} />);
    expect(screen.getByText('1.0K')).toBeInTheDocument();
  });

  it('renders Cache Efficiency heading', () => {
    render(<CacheChart sessionStats={makeSessionStats()} supportsCache={true} />);
    expect(screen.getByText('Cache Efficiency')).toBeInTheDocument();
  });

  it('renders pie chart when cache activity is present', () => {
    const stats = makeSessionStats();
    render(<CacheChart sessionStats={stats} supportsCache={true} />);
    expect(screen.getByTestId('pie-chart')).toBeInTheDocument();
  });

  it('shows Cache Efficiency heading even when unsupported', () => {
    render(<CacheChart sessionStats={makeSessionStats()} supportsCache={false} />);
    expect(screen.getByText('Cache Efficiency')).toBeInTheDocument();
  });

  it('shows "Cache reads" and "Cache writes" labels', () => {
    render(<CacheChart sessionStats={makeSessionStats()} supportsCache={true} />);
    expect(screen.getByText('Cache reads')).toBeInTheDocument();
    expect(screen.getByText('Cache writes')).toBeInTheDocument();
  });
});
