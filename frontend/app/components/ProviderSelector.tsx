'use client';

import { useState, useEffect } from 'react';
import type { Provider } from '@/types/token-data';
import { PROVIDER_MODELS } from '@/lib/model-limits';

export interface ProviderSettings {
  provider: Provider;
  model: string;
  apiKey: string;
}

interface Props {
  onChange: (settings: ProviderSettings) => void;
}

const STORAGE_KEY = 'llm-viz-provider-settings';

function readSavedSettings(): { provider: Provider; model: string } {
  if (typeof window === 'undefined') return { provider: 'anthropic', model: 'claude-sonnet-4-6' };
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const parsed = JSON.parse(saved) as { provider?: Provider; model?: string };
      return {
        provider: parsed.provider ?? 'anthropic',
        model: parsed.model ?? 'claude-sonnet-4-6',
      };
    }
  } catch {
    // ignore
  }
  return { provider: 'anthropic', model: 'claude-sonnet-4-6' };
}

export default function ProviderSelector({ onChange }: Props) {
  const [provider, setProvider] = useState<Provider>(() => readSavedSettings().provider);
  const [model, setModel] = useState<string>(() => readSavedSettings().model);
  const [apiKey, setApiKey] = useState('');
  const [showKey, setShowKey] = useState(false);

  // Notify parent + save to localStorage whenever settings change
  useEffect(() => {
    const settings: ProviderSettings = { provider, model, apiKey };
    onChange(settings);
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ provider, model }));
    // Note: apiKey intentionally NOT persisted to localStorage for security
  }, [provider, model, apiKey, onChange]);

  const handleProviderChange = (p: Provider) => {
    setProvider(p);
    setModel(PROVIDER_MODELS[p][0]);
  };

  const models = PROVIDER_MODELS[provider] ?? [];

  return (
    <div className="rounded-2xl bg-white/5 border border-white/10 p-6 backdrop-blur-sm">
      <h2 className="text-lg font-semibold text-white mb-4">Provider Settings</h2>

      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {/* Provider */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-slate-400 uppercase tracking-wide">
            Provider
          </label>
          <select
            value={provider}
            onChange={(e) => handleProviderChange(e.target.value as Provider)}
            className="rounded-lg bg-slate-800 border border-slate-700 text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500"
          >
            <option value="anthropic">Anthropic</option>
            <option value="openai">OpenAI</option>
          </select>
        </div>

        {/* Model */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-slate-400 uppercase tracking-wide">
            Model
          </label>
          <select
            value={model}
            onChange={(e) => setModel(e.target.value)}
            className="rounded-lg bg-slate-800 border border-slate-700 text-white px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500"
          >
            {models.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>

        {/* API Key */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-slate-400 uppercase tracking-wide">
            API Key <span className="text-amber-400">(browser only)</span>
          </label>
          <div className="relative">
            <input
              type={showKey ? 'text' : 'password'}
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              placeholder={provider === 'anthropic' ? 'sk-ant-...' : 'sk-...'}
              className="w-full rounded-lg bg-slate-800 border border-slate-700 text-white px-3 py-2 pr-10 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500 placeholder:text-slate-600"
            />
            <button
              type="button"
              onClick={() => setShowKey(!showKey)}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition-colors"
              aria-label={showKey ? 'Hide key' : 'Show key'}
            >
              {showKey ? (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                </svg>
              )}
            </button>
          </div>
        </div>
      </div>

      {!apiKey && (
        <p className="mt-3 text-xs text-amber-400/80">
          Enter your API key to start tracking token usage. Keys are never sent to our servers.
        </p>
      )}
    </div>
  );
}
