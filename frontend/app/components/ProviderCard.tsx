'use client';

import { useState, useEffect, useMemo, useCallback } from 'react';
import type { Provider, TokenDataPoint } from '@/types/token-data';
import type { ApiKey } from '@/types/api-key';
import { PROVIDER_MODELS } from '@/lib/model-limits';
import { formatTokenCount, calculateContextUsage, calculateCacheHitRate } from '@/lib/token-calculator';
import ChatInput from './ChatInput';

interface ProviderCardProps {
  provider: Provider;
  onRemove?: () => void;
}

interface ProviderConfig {
  name: string;
  icon: string;
  primaryColor: string;
  secondaryColor: string;
}

const PROVIDER_CONFIGS: Record<Provider, ProviderConfig> = {
  anthropic: {
    name: 'ANTHROPIC',
    icon: '🤖',
    primaryColor: 'var(--neon-cyan)',
    secondaryColor: 'var(--neon-blue)',
  },
  openai: {
    name: 'OPENAI',
    icon: '🌐',
    primaryColor: 'var(--neon-magenta)',
    secondaryColor: 'var(--neon-pink)',
  },
};

export default function ProviderCard({ provider, onRemove }: ProviderCardProps) {
  const [model, setModel] = useState<string>(PROVIDER_MODELS[provider][0]);
  const [apiKey, setApiKey] = useState<string>('');
  const [savedKey, setSavedKey] = useState<ApiKey | null>(null);
  const [showKeyInput, setShowKeyInput] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Realtime data (from SSE or manual API calls)
  const [tokens, setTokens] = useState<TokenDataPoint[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [sessionId] = useState(() => `${provider}-${Math.random().toString(36).substring(7)}`);

  // Historical data
  interface HistoricalDataPoint {
    input_tokens: number;
    output_tokens: number;
    date?: string;
  }
  const [historicalData, setHistoricalData] = useState<HistoricalDataPoint[]>([]);

  const config = PROVIDER_CONFIGS[provider];
  const models = PROVIDER_MODELS[provider] ?? [];

  // Fetch saved API key on mount
  useEffect(() => {
    fetch('http://localhost:8080/api/keys')
      .then((r) => r.json())
      .then((data: { keys?: ApiKey[] }) => {
        const key = data.keys?.find((k) => k.provider === provider);
        if (key) {
          setSavedKey(key);
          // Auto-set apiKey if saved
          // Note: Backend doesn't return raw key, so we can't auto-fill
        }
      })
      .catch(() => {
        // Silently ignore
      });
  }, [provider]);

  // Fetch historical data
  useEffect(() => {
    if (!apiKey) return;

    const fetchHistory = async () => {
      try {
        const endDate = new Date();
        const startDate = new Date();
        startDate.setDate(endDate.getDate() - 30);

        const url = new URL('http://localhost:8080/api/usage/history');
        url.searchParams.set('provider', provider);
        url.searchParams.set('api_key', apiKey);
        url.searchParams.set('start_date', startDate.toISOString().split('T')[0]);
        url.searchParams.set('end_date', endDate.toISOString().split('T')[0]);

        const res = await fetch(url.toString());
        if (res.ok) {
          const data = await res.json();
          setHistoricalData(data.data_points ?? []);
        }
      } catch {
        // Silently ignore
      }
    };

    fetchHistory();
  }, [provider, apiKey]);

  // Setup SSE connection when API key is available
  useEffect(() => {
    if (!apiKey && !savedKey) return;

    const sseUrl = `http://localhost:8080/api/sse?session_id=${sessionId}`;
    console.log(`[${provider}] Connecting to SSE:`, sseUrl);
    const eventSource = new EventSource(sseUrl);

    eventSource.onopen = () => {
      setIsConnected(true);
      setError(null);
    };

    eventSource.addEventListener('usage', (event) => {
      console.log(`[${provider}] SSE event received:`, event.data);
      try {
        const raw = JSON.parse(event.data);
        console.log(`[${provider}] Raw data:`, raw);

        // Map backend format to frontend TokenDataPoint
        const mapped: TokenDataPoint = {
          inputTokens: raw.usage?.usage?.input_tokens || 0,
          outputTokens: raw.usage?.usage?.output_tokens || 0,
          cacheReadTokens: raw.usage?.usage?.cache_read_input_tokens || 0,
          cacheCreationTokens: raw.usage?.usage?.cache_creation_input_tokens || 0,
          costUSD: raw.usage?.cost_usd || 0,
          timestamp: raw.usage?.timestamp || new Date().toISOString(),
        };

        console.log(`[${provider}] Mapped token data:`, mapped);
        setTokens((prev) => [...prev.slice(-99), mapped]);
      } catch (err) {
        console.error(`[${provider}] SSE parse error:`, err, event.data);
      }
    });

    eventSource.onerror = () => {
      setIsConnected(false);
      setError('Connection lost');
    };

    return () => {
      eventSource.close();
      setIsConnected(false);
    };
  }, [apiKey, savedKey, sessionId]);

  const handleSaveKey = async () => {
    if (!apiKey.startsWith('sk-')) {
      setError('Invalid API key format');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const res = await fetch('http://localhost:8080/api/keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          provider,
          name: `${config.name} Key`,
          key: apiKey,
        }),
      });

      if (!res.ok) {
        const errData = (await res.json()) as { error?: string };
        setError(errData.error ?? 'Failed to save key');
        return;
      }

      const saved = (await res.json()) as ApiKey;
      setSavedKey(saved);
      setShowKeyInput(false);
    } catch {
      setError('Network error');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteKey = async (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    if (!savedKey) {
      setError('No key to delete');
      return;
    }

    // Confirmation dialog
    const confirmMessage = `Delete ${config.name} API key?\n\nKey: ${savedKey.maskedKey}\n\nThis action cannot be undone.`;
    if (!window.confirm(confirmMessage)) {
      return;
    }

    try {
      const res = await fetch(`http://localhost:8080/api/keys/${savedKey.id}`, {
        method: 'DELETE',
      });

      if (res.ok) {
        setSavedKey(null);
        setApiKey('');
        setShowKeyInput(false);
      } else {
        const errorText = await res.text();
        setError(`Delete failed: ${errorText}`);
      }
    } catch {
      setError('Delete failed: network error');
    }
  };

  const handleSendMessage = useCallback(async (message: string) => {
    const keyToUse = apiKey || savedKey?.id;
    if (!keyToUse) {
      setError('No API key available');
      return;
    }

    setIsLoading(true);
    setError(null);

    const requestBody = {
      provider,
      model,
      messages: [{ role: 'user', content: message }],
      max_tokens: 1024,
      session_id: sessionId,
      api_key: keyToUse,
    };

    console.log(`[${provider}] Sending request:`, requestBody);

    try {
      const res = await fetch('http://localhost:8080/api/complete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
      });

      console.log(`[${provider}] Response status:`, res.status);

      if (!res.ok) {
        const errorText = await res.text();
        console.error(`[${provider}] Error response:`, errorText);
        setError(errorText || 'Request failed');
        return;
      }

      // Response will be broadcast via SSE, so we don't need to handle it here
      console.log(`[${provider}] Request completed successfully`);
    } catch {
      setError('Network error');
    } finally {
      setIsLoading(false);
    }
  }, [provider, model, apiKey, savedKey, sessionId]);

  // Compute session stats
  const sessionStats = useMemo(() => {
    const totalInput = tokens.reduce((sum, t) => sum + t.inputTokens, 0);
    const totalOutput = tokens.reduce((sum, t) => sum + t.outputTokens, 0);
    const totalCost = tokens.reduce((sum, t) => sum + t.costUSD, 0);
    const totalCacheRead = tokens.reduce((sum, t) => sum + t.cacheReadTokens, 0);
    const totalCacheCreation = tokens.reduce((sum, t) => sum + (t.cacheCreationTokens || 0), 0);

    return {
      totalInput,
      totalOutput,
      totalCost,
      totalCacheRead,
      totalCacheCreation,
    };
  }, [tokens]);

  // Calculate context usage
  const contextUsage = useMemo(() => {
    if (tokens.length === 0) return { utilizationPercent: 0, currentUsed: 0, maxTokens: 200000 };

    const latest = tokens[tokens.length - 1];
    const result = calculateContextUsage(
      latest.inputTokens,
      latest.cacheCreationTokens || 0,
      latest.cacheReadTokens,
      model
    );
    console.log(`[${provider}] Context usage:`, result);
    return result;
  }, [tokens, model, provider]);

  // Calculate cache hit rate
  const cacheHitRate = useMemo(() => {
    if (sessionStats.totalCacheRead + sessionStats.totalCacheCreation === 0) {
      console.log(`[${provider}] Cache: No cache tokens`);
      return 0;
    }
    const rate = calculateCacheHitRate(sessionStats.totalCacheRead, sessionStats.totalCacheCreation);
    console.log(`[${provider}] Cache hit rate:`, rate, '%', { read: sessionStats.totalCacheRead, creation: sessionStats.totalCacheCreation });
    return rate;
  }, [sessionStats, provider]);

  const hasKey = !!savedKey;

  return (
    <div
      className="provider-card p-6 animate-fade-in-up"
      data-provider={provider}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-3">
          <span className="text-3xl">{config.icon}</span>
          <div>
            <h2
              className="text-2xl font-display font-bold tracking-wider"
              style={{ color: config.primaryColor, textShadow: `var(--text-glow-${provider === 'anthropic' ? 'cyan' : 'magenta'})` }}
            >
              {config.name}
            </h2>
            <div className="flex items-center gap-2 mt-1">
              {isConnected ? (
                <span className="text-xs text-neon-green flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-neon-green animate-neon-pulse"></span>
                  ONLINE
                </span>
              ) : (
                <span className="text-xs text-gray-500 flex items-center gap-1">
                  <span className="w-2 h-2 rounded-full bg-gray-600"></span>
                  OFFLINE
                </span>
              )}
            </div>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {/* Settings icon */}
          <button
            onClick={() => setShowKeyInput((v) => !v)}
            className="p-2 rounded-lg border border-current opacity-50 hover:opacity-100 transition-opacity"
            style={{ color: config.primaryColor }}
            aria-label="Settings"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>

          {/* Remove provider */}
          {onRemove && (
            <button
              onClick={onRemove}
              className="p-2 rounded-lg border border-red-500 text-red-500 opacity-50 hover:opacity-100 transition-opacity"
              aria-label="Remove provider"
            >
              <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          )}
        </div>
      </div>

      {/* API Key Management */}
      {!hasKey ? (
        <div className="mb-4 p-4 rounded-lg border" style={{ borderColor: config.primaryColor, background: `rgba(${provider === 'anthropic' ? 'var(--neon-cyan-rgb)' : 'var(--neon-magenta-rgb)'}, 0.05)` }}>
          <p className="text-sm mb-3 opacity-70">No API key configured. Add one to start monitoring.</p>
          <div className="flex gap-2">
            <input
              type="password"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder="sk-..."
              className="flex-1 px-3 py-2 rounded-lg bg-black/50 border text-sm focus:outline-none focus:ring-2"
              style={{ borderColor: config.primaryColor, boxShadow: `0 0 10px rgba(${provider === 'anthropic' ? 'var(--neon-cyan-rgb)' : 'var(--neon-magenta-rgb)'}, 0.2)` }}
            />
            <button
              onClick={handleSaveKey}
              disabled={isLoading || !apiKey}
              className="btn-neon px-4 py-2 rounded-lg font-display text-sm font-bold disabled:opacity-30"
              style={{ color: config.primaryColor, borderColor: config.primaryColor }}
            >
              {isLoading ? 'SAVING...' : 'SAVE'}
            </button>
          </div>
          {error && <p className="text-xs text-neon-red mt-2">{error}</p>}
        </div>
      ) : (
        <>
          {/* Model selector */}
          <div className="mb-4 flex items-center gap-2">
            <label className="text-xs opacity-50 uppercase tracking-wider">Model:</label>
            <select
              value={model}
              onChange={(e) => setModel(e.target.value)}
              className="flex-1 px-3 py-2 rounded-lg bg-black/50 border text-sm focus:outline-none focus:ring-2"
              style={{ borderColor: config.primaryColor }}
            >
              {models.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
          </div>

          {/* Key management (collapsed) */}
          {showKeyInput && savedKey && (
            <div className="mb-4 p-4 rounded-lg border relative z-10" style={{ borderColor: config.primaryColor, background: `rgba(${provider === 'anthropic' ? 'var(--neon-cyan-rgb)' : 'var(--neon-magenta-rgb)'}, 0.05)` }}>
              <div className="mb-2">
                <div className="text-xs opacity-50 uppercase tracking-wider mb-1">Saved API Key</div>
                <div className="flex items-center gap-2">
                  <span className="text-xl">{config.icon}</span>
                  <span className="text-sm font-medium">{config.name}</span>
                </div>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-sm font-mono opacity-70">{savedKey.maskedKey}</span>
                <button
                  onClick={handleDeleteKey}
                  className="relative z-20 px-3 py-1 rounded-lg border border-red-500 text-red-500 text-xs hover:bg-red-500/20 transition-colors cursor-pointer font-medium"
                  type="button"
                >
                  DELETE
                </button>
              </div>
            </div>
          )}

          {/* Token Metrics */}
          <div className="grid grid-cols-3 gap-4 mb-6">
            <div className="text-center">
              <div className="text-xs opacity-50 uppercase tracking-wider mb-1">INPUT</div>
              <div className="text-2xl font-mono tabular-nums font-bold" style={{ color: config.primaryColor }}>
                {formatTokenCount(sessionStats.totalInput)}
              </div>
            </div>
            <div className="text-center">
              <div className="text-xs opacity-50 uppercase tracking-wider mb-1">OUTPUT</div>
              <div className="text-2xl font-mono tabular-nums font-bold" style={{ color: config.secondaryColor }}>
                {formatTokenCount(sessionStats.totalOutput)}
              </div>
            </div>
            <div className="text-center">
              <div className="text-xs opacity-50 uppercase tracking-wider mb-1">COST</div>
              <div className="text-2xl font-mono tabular-nums font-bold text-neon-yellow">
                ${sessionStats.totalCost.toFixed(3)}
              </div>
            </div>
          </div>

          {/* Unified Timeline (Historical + Real-time) */}
          <div className="mb-4">
            <div className="h-32 flex items-end gap-1">
              {/* Placeholder sparkline */}
              {historicalData.length === 0 && tokens.length === 0 ? (
                <div className="flex-1 flex items-center justify-center text-xs opacity-30">
                  No data yet
                </div>
              ) : (
                <>
                  {/* Historical bars (faded) */}
                  {historicalData.slice(-20).map((point, i) => {
                    const total = point.input_tokens + point.output_tokens;
                    const max = Math.max(...historicalData.map((p) => p.input_tokens + p.output_tokens), 1);
                    const height = (total / max) * 100;
                    return (
                      <div
                        key={`hist-${i}`}
                        className="flex-1 rounded-t opacity-30"
                        style={{ height: `${height}%`, background: config.primaryColor }}
                      />
                    );
                  })}

                  {/* Real-time bars (bright) */}
                  {tokens.slice(-10).map((t, i) => {
                    const total = t.inputTokens + t.outputTokens;
                    const max = Math.max(...tokens.map((p) => p.inputTokens + p.outputTokens), 1);
                    const height = (total / max) * 100;
                    return (
                      <div
                        key={`rt-${i}`}
                        className="flex-1 rounded-t"
                        style={{ height: `${height}%`, background: config.primaryColor, boxShadow: `var(--glow-${provider === 'anthropic' ? 'cyan' : 'magenta'})` }}
                      />
                    );
                  })}
                </>
              )}
            </div>
            <div className="mt-2 flex items-center justify-between text-xs opacity-50">
              <span>Historical</span>
              <span>Real-time →</span>
            </div>
          </div>

          {/* Context & Stats */}
          <div className="grid grid-cols-2 gap-4 mb-4">
            <div className="p-3 rounded-lg border" style={{ borderColor: config.primaryColor, background: `rgba(${provider === 'anthropic' ? 'var(--neon-cyan-rgb)' : 'var(--neon-magenta-rgb)'}, 0.05)` }}>
              <div className="text-xs opacity-50 uppercase tracking-wider mb-2">Context</div>
              <div className="h-2 rounded-full bg-black/50 overflow-hidden">
                <div
                  className="h-full rounded-full transition-all duration-500"
                  style={{ width: `${Math.min(contextUsage.utilizationPercent, 100)}%`, background: config.primaryColor, boxShadow: `var(--glow-${provider === 'anthropic' ? 'cyan' : 'magenta'})` }}
                />
              </div>
              <div className="text-xs mt-1 font-mono tabular-nums" style={{ color: config.primaryColor }}>
                {contextUsage.utilizationPercent.toFixed(1)}%
              </div>
            </div>

            <div className="p-3 rounded-lg border border-neon-green bg-neon-green/5">
              <div className="text-xs opacity-50 uppercase tracking-wider mb-2">Cache</div>
              <div className="text-lg font-mono tabular-nums font-bold text-neon-green">
                {cacheHitRate.toFixed(0)}%
              </div>
              <div className="text-xs opacity-50">Hit rate</div>
            </div>
          </div>

          {/* Chat Input */}
          <ChatInput
            onSubmit={handleSendMessage}
            isLoading={isLoading}
            disabled={!hasKey}
          />
        </>
      )}
    </div>
  );
}
