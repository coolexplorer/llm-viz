import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import CostTracker from '@/app/components/CostTracker';
import type { TokenDataPoint } from '@/types/token-data';

function makeDataPoint(overrides: Partial<TokenDataPoint> = {}): TokenDataPoint {
  return {
    timestamp: Date.now(),
    provider: 'anthropic',
    model: 'claude-sonnet-4-6',
    inputTokens: 1000,
    outputTokens: 500,
    cacheReadTokens: 0,
    cacheCreationTokens: 0,
    totalTokens: 1500,
    costUSD: 0.001,
    ...overrides,
  };
}

describe('CostTracker', () => {
  it('renders Cost Tracker heading', () => {
    render(<CostTracker latest={null} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.getByText('Cost Tracker')).toBeInTheDocument();
  });

  it('shows session total cost', () => {
    render(<CostTracker latest={null} sessionTotalCost={1.5} sessionCacheSavings={0} />);
    expect(screen.getByText('$1.5000')).toBeInTheDocument();
  });

  it('shows -- for breakdown when latest is null', () => {
    render(<CostTracker latest={null} sessionTotalCost={0} sessionCacheSavings={0} />);
    const dashes = screen.getAllByText('--');
    expect(dashes.length).toBeGreaterThanOrEqual(1);
  });

  it('shows cache savings when > 0', () => {
    render(<CostTracker latest={null} sessionTotalCost={0} sessionCacheSavings={0.005} />);
    expect(screen.getByText(/Saved.*via cache/i)).toBeInTheDocument();
  });

  it('does not show cache savings when 0', () => {
    render(<CostTracker latest={null} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.queryByText(/Saved.*via cache/i)).not.toBeInTheDocument();
  });

  it('shows request breakdown when latest is provided', () => {
    const point = makeDataPoint({ inputTokens: 1_000_000, outputTokens: 1_000_000 });
    render(<CostTracker latest={point} sessionTotalCost={18} sessionCacheSavings={0} />);
    // input cost = 3, output cost = 15
    expect(screen.getByText('$3.0000')).toBeInTheDocument();
    expect(screen.getByText('$15.0000')).toBeInTheDocument();
  });

  it('shows cache write cost when cacheCreationTokens > 0', () => {
    const point = makeDataPoint({ cacheCreationTokens: 1_000_000 });
    render(<CostTracker latest={point} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.getByText('Cache write cost')).toBeInTheDocument();
  });

  it('shows cache read cost when cacheReadTokens > 0', () => {
    const point = makeDataPoint({ cacheReadTokens: 1_000_000 });
    render(<CostTracker latest={point} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.getByText('Cache read cost')).toBeInTheDocument();
  });

  it('does not show cache write cost when 0', () => {
    const point = makeDataPoint({ cacheCreationTokens: 0 });
    render(<CostTracker latest={point} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.queryByText('Cache write cost')).not.toBeInTheDocument();
  });

  it('shows request total in breakdown', () => {
    const point = makeDataPoint();
    render(<CostTracker latest={point} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.getByText('Request total')).toBeInTheDocument();
  });

  it('shows USD estimates label', () => {
    render(<CostTracker latest={null} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.getByText('USD estimates')).toBeInTheDocument();
  });

  it('shows session total label', () => {
    render(<CostTracker latest={null} sessionTotalCost={0} sessionCacheSavings={0} />);
    expect(screen.getByText('session total')).toBeInTheDocument();
  });
});
