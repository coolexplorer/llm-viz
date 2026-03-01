import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import TokenCounter from '@/app/components/TokenCounter';
import type { TokenDataPoint } from '@/types/token-data';

function makeDataPoint(overrides: Partial<TokenDataPoint> = {}): TokenDataPoint {
  return {
    timestamp: Date.now(),
    provider: 'anthropic',
    model: 'claude-sonnet-4-6',
    inputTokens: 1000,
    outputTokens: 500,
    cacheReadTokens: 200,
    cacheCreationTokens: 100,
    totalTokens: 1800,
    costUSD: 0.01,
    ...overrides,
  };
}

describe('TokenCounter', () => {
  it('renders zero state when latest is null', () => {
    render(<TokenCounter latest={null} totalTokens={0} />);
    expect(screen.getByText('Token Counter')).toBeInTheDocument();
    // All four stat cards should show 0
    const zeros = screen.getAllByText('0');
    expect(zeros.length).toBeGreaterThanOrEqual(4);
  });

  it('displays input tokens from latest data point', () => {
    const point = makeDataPoint({ inputTokens: 5000, outputTokens: 0, cacheReadTokens: 0, cacheCreationTokens: 0 });
    render(<TokenCounter latest={point} totalTokens={5000} />);
    expect(screen.getAllByText('5.0K').length).toBeGreaterThanOrEqual(1);
  });

  it('displays output tokens', () => {
    const point = makeDataPoint({ inputTokens: 0, outputTokens: 2000, cacheReadTokens: 0, cacheCreationTokens: 0 });
    render(<TokenCounter latest={point} totalTokens={2000} />);
    expect(screen.getAllByText('2.0K').length).toBeGreaterThanOrEqual(1);
  });

  it('displays cache read tokens', () => {
    const point = makeDataPoint({ inputTokens: 0, outputTokens: 0, cacheReadTokens: 1500, cacheCreationTokens: 0 });
    render(<TokenCounter latest={point} totalTokens={1500} />);
    expect(screen.getAllByText('1.5K').length).toBeGreaterThanOrEqual(1);
  });

  it('displays cache write tokens', () => {
    const point = makeDataPoint({ inputTokens: 0, outputTokens: 0, cacheReadTokens: 0, cacheCreationTokens: 800 });
    render(<TokenCounter latest={point} totalTokens={800} />);
    expect(screen.getAllByText('800').length).toBeGreaterThanOrEqual(1);
  });

  it('shows session total tokens', () => {
    const point = makeDataPoint({ inputTokens: 100, outputTokens: 50 });
    render(<TokenCounter latest={point} totalTokens={5000} />);
    expect(screen.getByText('5.0K')).toBeInTheDocument();
  });

  it('shows labels for all stat cards', () => {
    render(<TokenCounter latest={null} totalTokens={0} />);
    expect(screen.getByText('Input')).toBeInTheDocument();
    expect(screen.getByText('Output')).toBeInTheDocument();
    expect(screen.getByText('Cache Read')).toBeInTheDocument();
    expect(screen.getByText('Cache Write')).toBeInTheDocument();
  });

  it('shows "This request" and "Session total" labels', () => {
    render(<TokenCounter latest={null} totalTokens={0} />);
    expect(screen.getByText('This request')).toBeInTheDocument();
    expect(screen.getByText('Session total')).toBeInTheDocument();
  });

  it('computes request total as sum of all token types', () => {
    const point = makeDataPoint({
      inputTokens: 1000,
      outputTokens: 1000,
      cacheReadTokens: 1000,
      cacheCreationTokens: 1000,
    });
    render(<TokenCounter latest={point} totalTokens={4000} />);
    // Request total = 4000 = 4.0K
    const values = screen.getAllByText('4.0K');
    expect(values.length).toBeGreaterThanOrEqual(1);
  });

  it('formats large tokens in millions', () => {
    const point = makeDataPoint({ inputTokens: 2_000_000 });
    render(<TokenCounter latest={point} totalTokens={2_000_000} />);
    expect(screen.getAllByText('2.00M').length).toBeGreaterThanOrEqual(1);
  });
});
