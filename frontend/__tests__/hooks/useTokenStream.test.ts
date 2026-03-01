import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act } from '@testing-library/react';
import { useTokenStream } from '@/hooks/useTokenStream';
import type { TokenDataPoint } from '@/types/token-data';

// Mock EventSource
class MockEventSource {
  static instances: MockEventSource[] = [];
  url: string;
  onopen: ((event: Event) => void) | null = null;
  onmessage: ((event: MessageEvent) => void) | null = null;
  onerror: ((event: Event) => void) | null = null;
  private _closed = false;

  constructor(url: string) {
    this.url = url;
    MockEventSource.instances.push(this);
  }

  close() {
    this._closed = true;
  }

  get closed() {
    return this._closed;
  }

  simulateOpen() {
    this.onopen?.(new Event('open'));
  }

  simulateMessage(data: unknown) {
    const event = new MessageEvent('message', { data: JSON.stringify(data) });
    this.onmessage?.(event);
  }

  simulateMalformedMessage() {
    const event = new MessageEvent('message', { data: 'not-json{{{' });
    this.onmessage?.(event);
  }

  simulateError() {
    this.onerror?.(new Event('error'));
  }
}

function makeDataPoint(overrides: Partial<TokenDataPoint> = {}): TokenDataPoint {
  return {
    timestamp: Date.now(),
    provider: 'anthropic',
    model: 'claude-sonnet-4-6',
    inputTokens: 100,
    outputTokens: 50,
    cacheReadTokens: 0,
    cacheCreationTokens: 0,
    totalTokens: 150,
    costUSD: 0.001,
    ...overrides,
  };
}

beforeEach(() => {
  MockEventSource.instances = [];
  vi.stubGlobal('EventSource', MockEventSource);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

describe('useTokenStream', () => {
  it('initializes with default state', () => {
    const { result } = renderHook(() => useTokenStream());
    expect(result.current.tokens).toEqual([]);
    expect(result.current.isConnected).toBe(false);
    expect(result.current.error).toBeNull();
    expect(result.current.sessionStats.totalRequests).toBe(0);
  });

  it('sets isConnected to true on open', () => {
    const { result } = renderHook(() => useTokenStream());
    act(() => {
      MockEventSource.instances[0].simulateOpen();
    });
    expect(result.current.isConnected).toBe(true);
    expect(result.current.error).toBeNull();
  });

  it('adds data point on message', () => {
    const { result } = renderHook(() => useTokenStream());
    const point = makeDataPoint();
    act(() => {
      MockEventSource.instances[0].simulateMessage(point);
    });
    expect(result.current.tokens).toHaveLength(1);
    expect(result.current.tokens[0].inputTokens).toBe(100);
  });

  it('updates sessionStats when data arrives', () => {
    const { result } = renderHook(() => useTokenStream());
    const point = makeDataPoint({ inputTokens: 200, outputTokens: 100, costUSD: 0.005 });
    act(() => {
      MockEventSource.instances[0].simulateMessage(point);
    });
    expect(result.current.sessionStats.totalRequests).toBe(1);
    expect(result.current.sessionStats.totalInputTokens).toBe(200);
    expect(result.current.sessionStats.totalCostUSD).toBeCloseTo(0.005, 5);
  });

  it('accumulates multiple data points', () => {
    const { result } = renderHook(() => useTokenStream());
    act(() => {
      MockEventSource.instances[0].simulateMessage(makeDataPoint());
      MockEventSource.instances[0].simulateMessage(makeDataPoint());
      MockEventSource.instances[0].simulateMessage(makeDataPoint());
    });
    expect(result.current.tokens).toHaveLength(3);
    expect(result.current.sessionStats.totalRequests).toBe(3);
  });

  it('sets error and disconnects on SSE error', () => {
    const { result } = renderHook(() => useTokenStream());
    act(() => {
      MockEventSource.instances[0].simulateOpen();
    });
    expect(result.current.isConnected).toBe(true);
    act(() => {
      MockEventSource.instances[0].simulateError();
    });
    expect(result.current.isConnected).toBe(false);
    expect(result.current.error).toContain('SSE connection lost');
  });

  it('ignores malformed JSON messages', () => {
    const { result } = renderHook(() => useTokenStream());
    act(() => {
      MockEventSource.instances[0].simulateMalformedMessage();
    });
    expect(result.current.tokens).toHaveLength(0);
  });

  it('clears data with clearData()', () => {
    const { result } = renderHook(() => useTokenStream());
    act(() => {
      MockEventSource.instances[0].simulateMessage(makeDataPoint());
      MockEventSource.instances[0].simulateMessage(makeDataPoint());
    });
    expect(result.current.tokens).toHaveLength(2);
    act(() => {
      result.current.clearData();
    });
    expect(result.current.tokens).toHaveLength(0);
    expect(result.current.sessionStats.totalRequests).toBe(0);
  });

  it('adds data point via addDataPoint()', () => {
    const { result } = renderHook(() => useTokenStream());
    const point = makeDataPoint({ inputTokens: 500 });
    act(() => {
      result.current.addDataPoint(point);
    });
    expect(result.current.tokens).toHaveLength(1);
    expect(result.current.tokens[0].inputTokens).toBe(500);
    expect(result.current.sessionStats.totalRequests).toBe(1);
  });

  it('closes EventSource on unmount', () => {
    const { unmount } = renderHook(() => useTokenStream());
    const es = MockEventSource.instances[0];
    unmount();
    expect(es.closed).toBe(true);
  });

  it('caps data points at 100', () => {
    const { result } = renderHook(() => useTokenStream());
    act(() => {
      for (let i = 0; i < 110; i++) {
        MockEventSource.instances[0].simulateMessage(makeDataPoint({ inputTokens: i }));
      }
    });
    expect(result.current.tokens).toHaveLength(100);
    // Most recent should be the last ones added
    expect(result.current.tokens[99].inputTokens).toBe(109);
  });

  it('uses custom streamUrl', () => {
    renderHook(() => useTokenStream('/api/custom-stream'));
    expect(MockEventSource.instances[0].url).toBe('/api/custom-stream');
  });
});
