'use client';

import { useState, useEffect, useCallback } from 'react';
import type { TokenDataPoint, SessionStats } from '@/types/token-data';
import { aggregateSessionStats } from '@/lib/token-calculator';

const MAX_DATA_POINTS = 100;

export function useTokenStream(streamUrl: string = '/api/stream') {
  const [tokens, setTokens] = useState<TokenDataPoint[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sessionStats, setSessionStats] = useState<SessionStats>({
    totalRequests: 0,
    totalInputTokens: 0,
    totalOutputTokens: 0,
    totalCacheReadTokens: 0,
    totalCacheCreationTokens: 0,
    totalCostUSD: 0,
    cacheHitRate: 0,
  });

  useEffect(() => {
    const es = new EventSource(streamUrl);

    es.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    es.onmessage = (event) => {
      try {
        const data: TokenDataPoint = JSON.parse(event.data);
        setTokens((prev) => [...prev.slice(-(MAX_DATA_POINTS - 1)), data]);
      } catch {
        // Ignore malformed events
      }
    };

    es.onerror = () => {
      setIsConnected(false);
      setError('SSE connection lost. Reconnecting...');
    };

    return () => {
      es.close();
      setIsConnected(false);
    };
  }, [streamUrl]);

  useEffect(() => {
    setSessionStats(aggregateSessionStats(tokens));
  }, [tokens]);

  const clearData = useCallback(() => {
    setTokens([]);
    setSessionStats({
      totalRequests: 0,
      totalInputTokens: 0,
      totalOutputTokens: 0,
      totalCacheReadTokens: 0,
      totalCacheCreationTokens: 0,
      totalCostUSD: 0,
      cacheHitRate: 0,
    });
  }, []);

  const addDataPoint = useCallback((point: TokenDataPoint) => {
    setTokens((prev) => [...prev.slice(-(MAX_DATA_POINTS - 1)), point]);
  }, []);

  return { tokens, isConnected, error, sessionStats, clearData, addDataPoint };
}
