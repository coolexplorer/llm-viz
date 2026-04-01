import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import HourlyHeatmap from '@/app/components/analytics/HourlyHeatmap';

// Mock recharts - heatmap may use custom SVG or Recharts
vi.mock('recharts', () => ({
  ResponsiveContainer: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  Tooltip: () => null,
}));

const mockHeatmapData = [
  { hour: 0, dayOfWeek: 0, requestCount: 2, totalTokens: 500 },
  { hour: 9, dayOfWeek: 1, requestCount: 15, totalTokens: 37500 },
  { hour: 14, dayOfWeek: 2, requestCount: 25, totalTokens: 62500 },
  { hour: 17, dayOfWeek: 4, requestCount: 30, totalTokens: 75000 },
  { hour: 23, dayOfWeek: 6, requestCount: 5, totalTokens: 12500 },
];

describe('HourlyHeatmap', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: mockHeatmapData }),
    } as Response);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders the Hourly Heatmap heading', async () => {
    render(<HourlyHeatmap />);
    expect(screen.getByText(/Hourly/i)).toBeInTheDocument();
  });

  it('fetches data from /api/analytics/heatmap on mount', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/analytics/heatmap'),
        expect.any(Object)
      );
    });
  });

  it('shows loading state initially', () => {
    render(<HourlyHeatmap />);
    expect(screen.getByText(/Loading/i)).toBeInTheDocument();
  });

  it('renders the heatmap grid after loading', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      expect(screen.getByTestId('heatmap-grid')).toBeInTheDocument();
    });
  });

  it('renders 24 hour labels', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      // Hours 0-23 should appear as labels
      expect(screen.getByText('0')).toBeInTheDocument();
      expect(screen.getByText('23')).toBeInTheDocument();
    });
  });

  it('renders 7 day-of-week labels', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      // At least Mon and Sun or similar day abbreviations
      const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
      const found = days.filter(d => screen.queryByText(d));
      expect(found.length).toBeGreaterThan(0);
    });
  });

  it('applies higher color intensity to cells with more requests', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      const cells = screen.getAllByTestId('heatmap-cell');
      // Find cell with highest request count - should have different color
      const intensities = cells.map(cell => cell.getAttribute('data-intensity') || cell.style.opacity);
      const maxIntensity = Math.max(...intensities.map(Number));
      const minIntensity = Math.min(...intensities.map(Number).filter(v => !isNaN(v)));
      expect(maxIntensity).toBeGreaterThan(minIntensity);
    });
  });

  it('renders cells for all hour/day combinations', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      const cells = screen.getAllByTestId('heatmap-cell');
      // 24 hours * 7 days = 168 cells
      expect(cells.length).toBe(168);
    });
  });

  it('polls every 10 seconds', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));
  });

  it('pauses polling when tab is not visible', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('resumes polling when tab becomes visible', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    act(() => {
      vi.advanceTimersByTime(10000);
    });

    Object.defineProperty(document, 'visibilityState', { value: 'visible', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));
  });

  it('shows "Last updated" indicator after data loads', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      expect(screen.getByText(/Last updated/i)).toBeInTheDocument();
    });
  });

  it('renders time range selector with 7d and 30d options', () => {
    render(<HourlyHeatmap />);
    expect(screen.getByText('7d')).toBeInTheDocument();
    expect(screen.getByText('30d')).toBeInTheDocument();
  });

  it('fetches with updated range when time range changes', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<HourlyHeatmap />);

    const btn30d = screen.getByText('30d');
    await user.click(btn30d);

    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('range=30d'),
        expect.any(Object)
      );
    });
  });

  it('shows a color legend/scale', async () => {
    render(<HourlyHeatmap />);
    await waitFor(() => {
      // Legend should show low/high markers or "Less"/"More" labels
      expect(
        screen.queryByText(/Less/i) ||
        screen.queryByText(/More/i) ||
        screen.queryByTestId('heatmap-legend')
      ).not.toBeNull();
    });
  });

  it('shows error state when fetch fails', async () => {
    vi.spyOn(global, 'fetch').mockRejectedValue(new Error('Network error'));
    render(<HourlyHeatmap />);
    await waitFor(() => {
      expect(screen.getByText(/Error/i)).toBeInTheDocument();
    });
  });

  it('shows empty state when no heatmap data', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: [] }),
    } as Response);
    render(<HourlyHeatmap />);
    await waitFor(() => {
      expect(screen.getByText(/No data/i)).toBeInTheDocument();
    });
  });

  it('stops polling on unmount', async () => {
    const { unmount } = render(<HourlyHeatmap />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    unmount();
    const callCount = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.length;

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(callCount);
  });
});
