import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import ContextGauge from '@/app/components/ContextGauge';
import type { ContextWindowStatus } from '@/types/token-data';

function makeStatus(overrides: Partial<ContextWindowStatus> = {}): ContextWindowStatus {
  return {
    model: 'gpt-4o',
    maxTokens: 128_000,
    currentUsed: 10_000,
    utilizationPercent: 7.8,
    remainingTokens: 118_000,
    isWarning: false,
    isCritical: false,
    ...overrides,
  };
}

describe('ContextGauge', () => {
  it('renders 0.0% when status is null', () => {
    render(<ContextGauge status={null} />);
    expect(screen.getByText('0.0%')).toBeInTheDocument();
    expect(screen.getByText('Context Window')).toBeInTheDocument();
  });

  it('displays utilization percentage', () => {
    const status = makeStatus({ utilizationPercent: 55.5 });
    render(<ContextGauge status={status} />);
    expect(screen.getByText('55.5%')).toBeInTheDocument();
  });

  it('caps percentage at 100%', () => {
    const status = makeStatus({ utilizationPercent: 150 });
    render(<ContextGauge status={status} />);
    expect(screen.getByText('100.0%')).toBeInTheDocument();
  });

  it('shows model name', () => {
    const status = makeStatus({ model: 'claude-sonnet-4-6' });
    render(<ContextGauge status={status} />);
    expect(screen.getByText('claude-sonnet-4-6')).toBeInTheDocument();
  });

  it('shows token usage stats', () => {
    const status = makeStatus({
      currentUsed: 10_000,
      remainingTokens: 118_000,
      maxTokens: 128_000,
    });
    render(<ContextGauge status={status} />);
    expect(screen.getByText('10.0K')).toBeInTheDocument();
    expect(screen.getByText('118.0K')).toBeInTheDocument();
    expect(screen.getByText('128.0K')).toBeInTheDocument();
  });

  it('shows warning message when isWarning is true', () => {
    const status = makeStatus({ isWarning: true, isCritical: false });
    render(<ContextGauge status={status} />);
    expect(screen.getByText(/Warning.*High context usage/i)).toBeInTheDocument();
  });

  it('shows critical message when isCritical is true', () => {
    const status = makeStatus({ isWarning: true, isCritical: true });
    render(<ContextGauge status={status} />);
    expect(screen.getByText(/Critical.*Context window nearly full/i)).toBeInTheDocument();
    expect(screen.queryByText(/Warning.*High context usage/i)).not.toBeInTheDocument();
  });

  it('shows no warning when status is normal', () => {
    const status = makeStatus({ isWarning: false, isCritical: false });
    render(<ContextGauge status={status} />);
    expect(screen.queryByText(/Warning/i)).not.toBeInTheDocument();
    expect(screen.queryByText(/Critical/i)).not.toBeInTheDocument();
  });

  it('renders "used" label', () => {
    render(<ContextGauge status={null} />);
    expect(screen.getByText('used')).toBeInTheDocument();
  });

  it('shows section labels', () => {
    const status = makeStatus();
    render(<ContextGauge status={status} />);
    expect(screen.getByText('Used')).toBeInTheDocument();
    expect(screen.getByText('Remaining')).toBeInTheDocument();
    expect(screen.getByText('Max capacity')).toBeInTheDocument();
  });
});
