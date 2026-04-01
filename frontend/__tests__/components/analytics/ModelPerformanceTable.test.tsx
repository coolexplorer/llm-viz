import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor, act, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import ModelPerformanceTable from '@/app/components/analytics/ModelPerformanceTable';

const mockModelData = [
  {
    model: 'claude-sonnet-4-6',
    provider: 'anthropic',
    requestCount: 50,
    totalTokens: 125000,
    avgTokensPerRequest: 2500,
    totalCostUSD: 1.875,
    avgCostPerRequest: 0.0375,
    avgLatencyMs: 920,
    cacheHitRate: 65,
  },
  {
    model: 'gpt-4o',
    provider: 'openai',
    requestCount: 30,
    totalTokens: 60000,
    avgTokensPerRequest: 2000,
    totalCostUSD: 0.9,
    avgCostPerRequest: 0.03,
    avgLatencyMs: 650,
    cacheHitRate: 30,
  },
  {
    model: 'gemini-2.0-flash',
    provider: 'gemini',
    requestCount: 20,
    totalTokens: 80000,
    avgTokensPerRequest: 4000,
    totalCostUSD: 0.4,
    avgCostPerRequest: 0.02,
    avgLatencyMs: 400,
    cacheHitRate: 15,
  },
];

describe('ModelPerformanceTable', () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: mockModelData }),
    } as Response);
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.restoreAllMocks();
  });

  it('renders Model Performance heading', async () => {
    render(<ModelPerformanceTable />);
    expect(screen.getByText(/Model Performance/i)).toBeInTheDocument();
  });

  it('fetches data from /api/analytics/models on mount', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(global.fetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/analytics/models'),
        expect.any(Object)
      );
    });
  });

  it('shows loading state initially', () => {
    render(<ModelPerformanceTable />);
    expect(screen.getByText(/Loading/i)).toBeInTheDocument();
  });

  it('renders table headers after loading', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByText(/Model/i)).toBeInTheDocument();
      expect(screen.getByText(/Requests/i)).toBeInTheDocument();
      expect(screen.getByText(/Tokens/i)).toBeInTheDocument();
      expect(screen.getByText(/Cost/i)).toBeInTheDocument();
    });
  });

  it('renders all model rows', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByText('claude-sonnet-4-6')).toBeInTheDocument();
      expect(screen.getByText('gpt-4o')).toBeInTheDocument();
      expect(screen.getByText('gemini-2.0-flash')).toBeInTheDocument();
    });
  });

  it('sorts by requests ascending when Requests header is clicked', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ModelPerformanceTable />);

    await waitFor(() => {
      expect(screen.getByText('claude-sonnet-4-6')).toBeInTheDocument();
    });

    const requestsHeader = screen.getByRole('columnheader', { name: /Requests/i });
    await user.click(requestsHeader);

    const rows = screen.getAllByRole('row');
    // First data row (after header) should have fewest requests (20 = gemini)
    expect(within(rows[1]).getByText('gemini-2.0-flash')).toBeInTheDocument();
  });

  it('sorts by requests descending on second click', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ModelPerformanceTable />);

    await waitFor(() => {
      expect(screen.getByText('claude-sonnet-4-6')).toBeInTheDocument();
    });

    const requestsHeader = screen.getByRole('columnheader', { name: /Requests/i });
    await user.click(requestsHeader);
    await user.click(requestsHeader);

    const rows = screen.getAllByRole('row');
    // First data row should have most requests (50 = claude)
    expect(within(rows[1]).getByText('claude-sonnet-4-6')).toBeInTheDocument();
  });

  it('sorts by cost when Cost header is clicked', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ModelPerformanceTable />);

    await waitFor(() => {
      expect(screen.getByText('gpt-4o')).toBeInTheDocument();
    });

    const costHeader = screen.getByRole('columnheader', { name: /Cost/i });
    await user.click(costHeader);

    const rows = screen.getAllByRole('row');
    // Ascending by totalCostUSD: gemini (0.4) first
    expect(within(rows[1]).getByText('gemini-2.0-flash')).toBeInTheDocument();
  });

  it('shows sort direction indicator on active column', async () => {
    const user = userEvent.setup({ advanceTimers: vi.advanceTimersByTime });
    render(<ModelPerformanceTable />);

    await waitFor(() => {
      expect(screen.getByText('claude-sonnet-4-6')).toBeInTheDocument();
    });

    const requestsHeader = screen.getByRole('columnheader', { name: /Requests/i });
    await user.click(requestsHeader);

    // Sort indicator (↑ or ↓ or aria-sort) should be present
    expect(
      requestsHeader.getAttribute('aria-sort') === 'ascending' ||
      requestsHeader.querySelector('[data-sort]') !== null ||
      requestsHeader.textContent?.match(/[↑↓▲▼]/)
    ).toBeTruthy();
  });

  it('displays latency column header', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByRole('columnheader', { name: /Latency/i })).toBeInTheDocument();
    });
  });

  it('displays cache hit rate column header', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByRole('columnheader', { name: /Cache/i })).toBeInTheDocument();
    });
  });

  it('polls every 10 seconds', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(2));
  });

  it('pauses polling when tab is not visible', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    Object.defineProperty(document, 'visibilityState', { value: 'hidden', writable: true });
    document.dispatchEvent(new Event('visibilitychange'));

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(1);
  });

  it('shows "Last updated" indicator after data loads', async () => {
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByText(/Last updated/i)).toBeInTheDocument();
    });
  });

  it('renders time range selector', () => {
    render(<ModelPerformanceTable />);
    expect(screen.getByText('24h')).toBeInTheDocument();
    expect(screen.getByText('7d')).toBeInTheDocument();
    expect(screen.getByText('30d')).toBeInTheDocument();
  });

  it('shows error state when fetch fails', async () => {
    vi.spyOn(global, 'fetch').mockRejectedValue(new Error('Network error'));
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByText(/Error/i)).toBeInTheDocument();
    });
  });

  it('shows empty state when no model data', async () => {
    vi.spyOn(global, 'fetch').mockResolvedValue({
      ok: true,
      json: async () => ({ data: [] }),
    } as Response);
    render(<ModelPerformanceTable />);
    await waitFor(() => {
      expect(screen.getByText(/No data/i)).toBeInTheDocument();
    });
  });

  it('stops polling on unmount', async () => {
    const { unmount } = render(<ModelPerformanceTable />);
    await waitFor(() => expect(global.fetch).toHaveBeenCalledTimes(1));

    unmount();
    const callCount = (global.fetch as ReturnType<typeof vi.fn>).mock.calls.length;

    act(() => {
      vi.advanceTimersByTime(10000);
    });
    expect(global.fetch).toHaveBeenCalledTimes(callCount);
  });
});
