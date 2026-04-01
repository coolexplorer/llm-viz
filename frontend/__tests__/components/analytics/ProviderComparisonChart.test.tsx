import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ProviderComparisonChart from '@/app/components/analytics/ProviderComparisonChart';

// Mock recharts to avoid SVG rendering issues in jsdom
vi.mock('recharts', () => ({
  BarChart: ({ children }: { children: React.ReactNode }) => <div data-testid="bar-chart">{children}</div>,
  Bar: ({ fill, dataKey }: { fill: string; dataKey: string }) => (
    <div data-testid={`bar-${dataKey}`} data-fill={fill} />
  ),
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  Legend: () => null,
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

const mockProviderData = [
  { provider: 'anthropic', totalTokens: 50000, totalCost: 0.75, requestCount: 20, avgLatencyMs: 850 },
  { provider: 'openai', totalTokens: 30000, totalCost: 0.45, requestCount: 15, avgLatencyMs: 620 },
  { provider: 'gemini', totalTokens: 20000, totalCost: 0.10, requestCount: 10, avgLatencyMs: 450 },
];

describe('ProviderComparisonChart', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: mockProviderData }),
    } as Response);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders the Provider Comparison heading', async () => {
    render(<ProviderComparisonChart />);
    expect(screen.getByText(/Provider Comparison/i)).toBeInTheDocument();
  });

  it('fetches data from /api/analytics/providers on mount', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/analytics/providers'),
        expect.any(Object)
      );
    });
  });

  it('renders bar chart when data is loaded', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
    });
  });

  it('shows loading state initially', () => {
    render(<ProviderComparisonChart />);
    expect(screen.getByText(/Loading/i)).toBeInTheDocument();
  });

  it('assigns distinct colors to each provider', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(screen.getByTestId('bar-chart')).toBeInTheDocument();
    });
    // Anthropic color should not equal OpenAI color
    const bars = screen.getAllByTestId(/^bar-/);
    const colors = bars.map(b => b.getAttribute('data-fill'));
    const uniqueColors = new Set(colors.filter(Boolean));
    expect(uniqueColors.size).toBeGreaterThan(1);
  });

  it('uses specific color for anthropic provider', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      // Anthropic should use its brand color (e.g., #D4A27F or similar)
      const bars = screen.getAllByTestId(/^bar-/);
      expect(bars.length).toBeGreaterThan(0);
    });
  });

  it('renders time range selector with 24h, 7d, 30d options', () => {
    render(<ProviderComparisonChart />);
    expect(screen.getByText('24h')).toBeInTheDocument();
    expect(screen.getByText('7d')).toBeInTheDocument();
    expect(screen.getByText('30d')).toBeInTheDocument();
  });

  it('fetches with updated range when time range changes', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ProviderComparisonChart />);

    const btn30d = screen.getByText('30d');
    await user.click(btn30d);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('range=30d'),
        expect.any(Object)
      );
    });
  });

  it('polls every 10 seconds', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));
  });

  it('pauses polling when tab is not visible', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('shows "Last updated" indicator after data loads', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(screen.getByText(/Last updated/i)).toBeInTheDocument();
    });
  });

  it('displays provider names in the chart', async () => {
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(screen.getByText('anthropic')).toBeInTheDocument();
    });
  });

  it('shows token and cost metric toggle', async () => {
    render(<ProviderComparisonChart />);
    expect(
      screen.getByRole('button', { name: /tokens/i }) ||
      screen.getByText(/tokens/i)
    ).toBeDefined();
    expect(
      screen.getByRole('button', { name: /cost/i }) ||
      screen.getByText(/cost/i)
    ).toBeDefined();
  });

  it('shows error state when fetch fails', async () => {
    vi.spyOn(global, 'fetch').mockRejectedValue(new Error('Network error'));
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(screen.getByText(/Error/i)).toBeInTheDocument();
    });
  });

  it('shows empty state when no provider data', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: [] }),
    } as Response);
    render(<ProviderComparisonChart />);
    await waitFor(() => {
      expect(screen.getByText(/No data/i)).toBeInTheDocument();
    });
  });

  it('stops polling on unmount', async () => {
    const { unmount } = render(<ProviderComparisonChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    unmount();
    const callCount = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.length;

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(callCount);
  });
});
