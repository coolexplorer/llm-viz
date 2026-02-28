'use client';

import { useState, useCallback, useRef } from 'react';
import ProviderSelector, { type ProviderSettings } from './components/ProviderSelector';
import TokenCounter from './components/TokenCounter';
import ContextGauge from './components/ContextGauge';
import CostTracker from './components/CostTracker';
import CacheChart from './components/CacheChart';
import UsageTimeline from './components/UsageTimeline';
import ChatInput from './components/ChatInput';
import { useTokenStream } from '@/hooks/useTokenStream';
import { calculateContextUsage } from '@/lib/token-calculator';
import { calculateCost } from '@/lib/cost-calculator';
import type { TokenDataPoint, ContextWindowStatus } from '@/types/token-data';

export default function Dashboard() {
  const { tokens, isConnected, error, sessionStats, clearData, addDataPoint } =
    useTokenStream('/api/stream');
  const [settings, setSettings] = useState<ProviderSettings | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [requestError, setRequestError] = useState<string | null>(null);
  const [contextStatus, setContextStatus] = useState<ContextWindowStatus | null>(null);
  const sessionCacheSavingsRef = useRef(0);

  const handleSettingsChange = useCallback((s: ProviderSettings) => {
    setSettings(s);
  }, []);

  const handleSendMessage = useCallback(
    async (message: string) => {
      if (!settings?.apiKey) return;

      setIsLoading(true);
      setRequestError(null);

      try {
        const res = await fetch('/api/proxy', {
          method: 'POST',
          headers: { 'content-type': 'application/json' },
          body: JSON.stringify({
            provider: settings.provider,
            model: settings.model,
            apiKey: settings.apiKey,
            messages: [{ role: 'user', content: message }],
            maxTokens: 512,
          }),
        });

        if (!res.ok) {
          const err = (await res.json()) as { error?: string };
          throw new Error(err.error ?? 'Request failed');
        }

        const data = (await res.json()) as Record<string, unknown>;
        const model = settings.model;
        let dataPoint: TokenDataPoint | null = null;

        if (settings.provider === 'anthropic') {
          const usage = data.usage as {
            input_tokens: number;
            output_tokens: number;
            cache_creation_input_tokens?: number;
            cache_read_input_tokens?: number;
          };
          const cost = calculateCost(
            model,
            usage.input_tokens,
            usage.output_tokens,
            usage.cache_creation_input_tokens ?? 0,
            usage.cache_read_input_tokens ?? 0,
          );
          sessionCacheSavingsRef.current += cost.cacheSavings;
          dataPoint = {
            timestamp: Date.now(),
            provider: 'anthropic',
            model,
            inputTokens: usage.input_tokens,
            outputTokens: usage.output_tokens,
            cacheReadTokens: usage.cache_read_input_tokens ?? 0,
            cacheCreationTokens: usage.cache_creation_input_tokens ?? 0,
            totalTokens:
              usage.input_tokens +
              usage.output_tokens +
              (usage.cache_read_input_tokens ?? 0) +
              (usage.cache_creation_input_tokens ?? 0),
            costUSD: cost.totalCost,
          };
        } else {
          const usage = data.usage as {
            prompt_tokens: number;
            completion_tokens: number;
            prompt_tokens_details?: { cached_tokens?: number };
          };
          const cacheRead = usage.prompt_tokens_details?.cached_tokens ?? 0;
          const inputTokens = usage.prompt_tokens - cacheRead;
          const cost = calculateCost(model, inputTokens, usage.completion_tokens, 0, cacheRead);
          sessionCacheSavingsRef.current += cost.cacheSavings;
          dataPoint = {
            timestamp: Date.now(),
            provider: 'openai',
            model,
            inputTokens,
            outputTokens: usage.completion_tokens,
            cacheReadTokens: cacheRead,
            cacheCreationTokens: 0,
            totalTokens: usage.prompt_tokens + usage.completion_tokens,
            costUSD: cost.totalCost,
          };
        }

        if (dataPoint) {
          if (!isConnected) {
            addDataPoint(dataPoint);
          }
          setContextStatus(
            calculateContextUsage(
              dataPoint.inputTokens,
              dataPoint.cacheCreationTokens,
              dataPoint.cacheReadTokens,
              model,
            ),
          );
        }
      } catch (err) {
        setRequestError(err instanceof Error ? err.message : 'Unknown error');
      } finally {
        setIsLoading(false);
      }
    },
    [settings, isConnected, addDataPoint],
  );

  const latest = tokens[tokens.length - 1] ?? null;
  const supportsCache = settings?.provider === 'anthropic';

  return (
    <main className="min-h-screen bg-slate-950 text-white">
      {/* Header */}
      <header className="border-b border-white/10 bg-slate-950/80 backdrop-blur-sm sticky top-0 z-10">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-teal-500 to-indigo-600 flex items-center justify-center">
              <svg className="w-4 h-4 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
            </div>
            <span className="text-lg font-bold tracking-tight">llm-viz</span>
            <span className="text-xs text-slate-500 hidden sm:inline">Real-time token dashboard</span>
          </div>

          <div className="flex items-center gap-3">
            <div className="flex items-center gap-1.5">
              <div
                className={`w-2 h-2 rounded-full ${isConnected ? 'bg-teal-400 animate-pulse' : 'bg-slate-600'}`}
              />
              <span className="text-xs text-slate-500 hidden sm:inline">
                {isConnected ? 'Live' : 'Offline'}
              </span>
            </div>
            <button
              onClick={clearData}
              className="text-xs text-slate-500 hover:text-slate-300 transition-colors"
            >
              Clear data
            </button>
          </div>
        </div>
      </header>

      {/* Main content */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 space-y-4">
        {error && (
          <div className="rounded-xl bg-red-500/10 border border-red-500/20 px-4 py-3 text-sm text-red-400">
            {error}
          </div>
        )}
        {requestError && (
          <div className="rounded-xl bg-red-500/10 border border-red-500/20 px-4 py-3 text-sm text-red-400">
            {requestError}
          </div>
        )}

        <ProviderSelector onChange={handleSettingsChange} />

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <TokenCounter
            latest={latest}
            totalTokens={
              sessionStats.totalInputTokens +
              sessionStats.totalOutputTokens +
              sessionStats.totalCacheReadTokens +
              sessionStats.totalCacheCreationTokens
            }
          />
          <ContextGauge status={contextStatus} />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <CostTracker
            latest={latest}
            sessionTotalCost={sessionStats.totalCostUSD}
            sessionCacheSavings={sessionCacheSavingsRef.current}
          />
          <CacheChart sessionStats={sessionStats} supportsCache={supportsCache} />
        </div>

        <UsageTimeline data={tokens} />

        <ChatInput
          onSubmit={handleSendMessage}
          isLoading={isLoading}
          disabled={!settings?.apiKey}
        />
      </div>
    </main>
  );
}
