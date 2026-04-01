import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import CumulativeUsageChart from '@/app/components/analytics/CumulativeUsageChart';

// Mock recharts to avoid SVG rendering issues in jsdom
vi.mock('recharts', () => ({
  AreaChart: ({ children }: { children: React.ReactNode }) => <div data-testid="area-chart">{children}</div>,
  Area: () => null,
  XAxis: () => null,
  YAxis: () => null,
  CartesianGrid: () => null,
  Tooltip: () => null,
  Legend: () => null,
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}));

const mockCumulativeData = [
  { date: '2026-02-01', totalTokens: 10000, totalCost: 0.15, requestCount: 5 },
  { date: '2026-02-02', totalTokens: 25000, totalCost: 0.37, requestCount: 12 },
  { date: '2026-02-03', totalTokens: 45000, totalCost: 0.68, requestCount: 20 },
];

describe('CumulativeUsageChart', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: mockCumulativeData }),
    } as Response);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders the Cumulative Usage heading', async () => {
    render(<CumulativeUsageChart />);
    expect(screen.getByText(/Cumulative Usage/i)).toBeInTheDocument();
  });

  it('fetches data from /api/analytics/cumulative on mount', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/analytics/cumulative'),
        expect.any(Object)
      );
    });
  });

  it('renders area chart when data is loaded', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => {
      expect(screen.getByTestId('area-chart')).toBeInTheDocument();
    });
  });

  it('shows loading state initially', () => {
    render(<CumulativeUsageChart />);
    expect(screen.getByText(/Loading/i)).toBeInTheDocument();
  });

  it('polls every 10 seconds', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(3));
  });

  it('pauses polling when tab is not visible', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    // Hide tab
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    // Should not have polled again
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('resumes polling when tab becomes visible again', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    // Hide tab
    Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(1);

    // Show tab again
    Object.defineProperty(document, 'visibilityState', { value: 'visible', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    // Should immediately re-fetch on visibility restore
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));
  });

  it('shows "Last updated" indicator', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => {
      expect(screen.getByText(/Last updated/i)).toBeInTheDocument();
    });
  });

  it('renders time range selector with 24h, 7d, 30d options', () => {
    render(<CumulativeUsageChart />);
    expect(screen.getByText('24h')).toBeInTheDocument();
    expect(screen.getByText('7d')).toBeInTheDocument();
    expect(screen.getByText('30d')).toBeInTheDocument();
  });

  it('changes time range when selector is clicked', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<CumulativeUsageChart />);

    const btn7d = screen.getByText('7d');
    await user.click(btn7d);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('range=7d'),
        expect.any(Object)
      );
    });
  });

  it('defaults to 24h time range', async () => {
    render(<CumulativeUsageChart />);
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('range=24h'),
        expect.any(Object)
      );
    });
  });

  it('shows error state when fetch fails', async () => {
    vi.spyOn(global, 'fetch').mockRejectedValue(new Error('Network error'));
    render(<CumulativeUsageChart />);
    await waitFor(() => {
      expect(screen.getByText(/Error/i)).toBeInTheDocument();
    });
  });

  it('shows empty state when data array is empty', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: [] }),
    } as Response);
    render(<CumulativeUsageChart />);
    await waitFor(() => {
      expect(screen.getByText(/No data/i)).toBeInTheDocument();
    });
  });

  it('stops polling on unmount', async () => {
    const { unmount } = render(<CumulativeUsageChart />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    unmount();
    const callCount = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.length;

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(callCount);
  });
});
