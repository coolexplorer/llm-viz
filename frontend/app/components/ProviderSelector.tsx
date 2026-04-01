'use client';

import { useState, useEffect } from 'react';
import type { Provider } from '@/types/token-data';
import { PROVIDER_MODELS } from '@/lib/model-limits';
import type { ApiKey } from '@/types/api-key';

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
  const [apiKey, setApiKey] = useState<string>('');
  const [showApiKey, setShowApiKey] = useState(false);

  // API Key Management state
  const [savedKeys, setSavedKeys] = useState<ApiKey[]>([]);
  const [showKeyManager, setShowKeyManager] = useState(false);
  const [newKeyInput, setNewKeyInput] = useState('');
  const [keyError, setKeyError] = useState<string | null>(null);

  // Notify parent + save to localStorage whenever settings change
  useEffect(() => {
    const settings: ProviderSettings = { provider, model, apiKey };
    onChange(settings);
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ provider, model }));
  }, [provider, model, apiKey, onChange]);

  // Fetch saved keys on mount
  useEffect(() => {
    fetch('http://localhost:8080/api/keys', { method: 'GET' })
      .then((r) => r.json())
      .then((data: { keys?: ApiKey[] }) => setSavedKeys(data.keys ?? []))
      .catch(() => {
        // silently ignore fetch errors on mount
      });
  }, []);

  const handleProviderChange = (p: Provider) => {
    setProvider(p);
    setModel(PROVIDER_MODELS[p][0]);
    setApiKey('');
  };

  const handleSaveKey = async () => {
    setKeyError(null);
    if (!newKeyInput.startsWith('sk-')) {
      setKeyError('Invalid API key format (must start with "sk-")');
      return;
    }
    try {
      const keyName = `${provider.charAt(0).toUpperCase() + provider.slice(1)} API Key`;
      const res = await fetch('http://localhost:8080/api/keys', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ provider, name: keyName, key: newKeyInput }),
      });
      if (!res.ok) {
        const errData = await res.json() as { error?: string };
        setKeyError(errData.error ?? 'Error saving key');
        return;
      }
      const saved = await res.json() as { id: string; maskedKey: string; label?: string };
      const newKey: ApiKey = {
        id: saved.id,
        provider,
        maskedKey: saved.maskedKey,
        label: saved.label ?? `${saved.maskedKey} (Active)`,
      };
      setSavedKeys((prev) => [...prev, newKey]);
      setNewKeyInput('');
    } catch {
      setKeyError('Failed to save key. Network error.');
    }
  };

  const handleDeleteKey = async (id: string) => {
    setKeyError(null);
    try {
      const res = await fetch(`http://localhost:8080/api/keys/${id}`, { method: 'DELETE' });
      if (!res.ok) {
        setKeyError('Delete failed');
        return;
      }
      setSavedKeys((prev) => prev.filter((k) => k.id !== id));
    } catch {
      setKeyError('Delete failed');
    }
  };

  const models = PROVIDER_MODELS[provider] ?? [];
  const filteredKeys = savedKeys.filter((k) => k.provider === provider);

  return (
    <div className="rounded-2xl glass-card-hover p-6">
      <h2 className="text-xl font-heading font-semibold text-white mb-4 tracking-tight">
        Provider Settings
      </h2>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* Provider */}
        <div className="flex flex-col gap-1.5">
          <label className="text-xs font-medium text-slate-400 uppercase tracking-wide">
            Provider
          </label>
          <select
            value={provider}
            onChange={(e) => handleProviderChange(e.target.value as Provider)}
            className="w-full rounded-xl bg-slate-800/50 border border-slate-700/50 text-white px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent transition-all duration-300"
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
            className="w-full rounded-xl bg-slate-800/50 border border-slate-700/50 text-white px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent transition-all duration-300"
          >
            {models.map((m) => (
              <option key={m} value={m}>
                {m}
              </option>
            ))}
          </select>
        </div>
      </div>

      {/* Main API Key Input */}
      <div className="mt-4 flex flex-col gap-1.5">
        <label className="text-xs font-medium text-slate-400 uppercase tracking-wide">
          API Key
        </label>
        <div className="flex gap-2">
          <input
            type={showApiKey ? 'text' : 'password'}
            value={apiKey}
            onChange={(e) => setApiKey(e.target.value)}
            placeholder={provider === 'anthropic' ? 'sk-ant-...' : 'sk-...'}
            className="flex-1 rounded-xl bg-slate-800/50 border border-slate-700/50 text-white px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent placeholder:text-slate-500 transition-all duration-300"
          />
          <button
            type="button"
            onClick={() => setShowApiKey((v) => !v)}
            aria-label={showApiKey ? 'Hide Key' : 'Show Key'}
            className="px-3 py-2 rounded-xl bg-white/5 border border-white/10 hover:bg-white/10 text-slate-300 text-sm transition-all duration-300"
          >
            {showApiKey ? 'Hide' : 'Show'}
          </button>
        </div>
        {!apiKey && (
          <p className="text-xs text-amber-400">
            Enter your API key to get started
          </p>
        )}
      </div>

      {/* API Key Management Section */}
      <div className="mt-5 pt-4 border-t border-white/10">
        <div className="flex items-center justify-between mb-3">
          <span className="text-sm font-medium text-slate-300">API Key Management</span>
          <button
            type="button"
            onClick={() => setShowKeyManager((v) => !v)}
            aria-label="Manage API Keys"
            className="inline-flex items-center gap-1 px-2 py-1 rounded-lg bg-teal-500/10 border border-teal-500/20 text-teal-400 text-xs font-medium hover:bg-teal-500/20 transition-all duration-300"
          >
            {showKeyManager ? 'Cancel' : 'Add API Key'}
          </button>
        </div>

        {/* Saved keys (simplified: 1 key per provider, no radio buttons) */}
        {filteredKeys.length > 0 && (
          <div className="flex flex-col gap-2 mb-3">
            {filteredKeys.map((key) => (
              <div
                key={key.id}
                className="flex items-center justify-between px-3 py-2 rounded-xl bg-teal-500/5 border border-teal-500/20"
              >
                <span className="text-sm text-slate-300 font-mono">
                  {key.label ?? `${key.maskedKey} (Active)`}
                </span>
                <button
                  type="button"
                  onClick={() => handleDeleteKey(key.id)}
                  aria-label="Delete"
                  className="px-2 py-1 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 hover:bg-red-500/20 text-xs transition-all duration-300"
                >
                  Delete
                </button>
              </div>
            ))}
          </div>
        )}

        {/* Add new key form (collapsed by default) */}
        {showKeyManager && (
          <div className="flex flex-col gap-3 rounded-xl bg-teal-500/5 border border-teal-500/20 p-4">
            <input
              type="password"
              value={newKeyInput}
              onChange={(e) => setNewKeyInput(e.target.value)}
              placeholder="paste your api key"
              className="w-full rounded-xl bg-slate-800/50 border border-slate-700/50 text-white px-4 py-3 text-sm focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent placeholder:text-slate-500 transition-all duration-300"
            />
            <button
              type="button"
              onClick={handleSaveKey}
              disabled={!newKeyInput}
              aria-label="Save Key"
              className="w-full px-4 py-2 rounded-xl bg-gradient-to-r from-teal-500 to-teal-600 hover:from-teal-400 hover:to-teal-500 text-white font-semibold text-sm shadow-md hover:shadow-[var(--shadow-glow-teal)] disabled:opacity-50 disabled:cursor-not-allowed transition-all duration-300 active:scale-95"
            >
              Save Key
            </button>
          </div>
        )}

        {/* Error message */}
        {keyError && (
          <p className="text-xs text-red-400 mt-2">{keyError}</p>
        )}
      </div>
    </div>
  );
}
