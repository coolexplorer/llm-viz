import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import UsageTimeline from '@/app/components/UsageTimeline';
import type { TokenDataPoint } from '@/types/token-data';

// Mock recharts components
vi.mock('recharts', () => ({
  LineChart: ({ children }: { children: React.ReactNode }) => <div data-testid="line-chart">{children}</div>,
  Line: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  Legend: () => null,
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

function makeDataPoint(overrides: Partial<TokenDataPoint> = {}): TokenDataPoint {
  return {
    timestamp: 1700000000000,
    provider: 'anthropic',
    model: 'claude-sonnet-4-6',
    inputTokens: 100,
    outputTokens: 50,
    cacheReadTokens: 10,
    cacheCreationTokens: 5,
    totalTokens: 165,
    costUSD: 0.001,
    ...overrides,
  };
}

describe('UsageTimeline', () => {
  it('shows empty state message when data is empty', () => {
    render(<UsageTimeline data={[]} />);
    expect(screen.getByText(/No data yet/i)).toBeInTheDocument();
  });

  it('shows Usage Timeline heading', () => {
    render(<UsageTimeline data={[]} />);
    expect(screen.getByText('Usage Timeline')).toBeInTheDocument();
  });

  it('shows request count in header', () => {
    const data = [makeDataPoint(), makeDataPoint(), makeDataPoint()];
    render(<UsageTimeline data={data} />);
    expect(screen.getByText('Last 3 requests')).toBeInTheDocument();
  });

  it('renders line chart when data is present', () => {
    const data = [makeDataPoint()];
    render(<UsageTimeline data={data} />);
    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });

  it('does not show empty state when data is present', () => {
    const data = [makeDataPoint()];
    render(<UsageTimeline data={data} />);
    expect(screen.queryByText(/No data yet/i)).not.toBeInTheDocument();
  });

  it('shows "Last 0 requests" for empty data', () => {
    render(<UsageTimeline data={[]} />);
    expect(screen.getByText('Last 0 requests')).toBeInTheDocument();
  });

  it('renders with multiple data points', () => {
    const data = Array.from({ length: 10 }, (_, i) => makeDataPoint({ timestamp: 1700000000000 + i * 1000 }));
    render(<UsageTimeline data={data} />);
    expect(screen.getByText('Last 10 requests')).toBeInTheDocument();
    expect(screen.getByTestId('line-chart')).toBeInTheDocument();
  });
});
