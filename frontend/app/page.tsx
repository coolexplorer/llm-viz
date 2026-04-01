'use client';

import { useState } from 'react';
import type { Provider } from '@/types/token-data';
import ProviderCard from './components/ProviderCard';

const AVAILABLE_PROVIDERS: Provider[] = ['anthropic', 'openai'];

export default function Dashboard() {
  // Active providers (user can add/remove)
  const [activeProviders, setActiveProviders] = useState<Provider[]>(['anthropic']);
  const [showAddMenu, setShowAddMenu] = useState(false);

  const handleAddProvider = (provider: Provider) => {
    if (!activeProviders.includes(provider)) {
      setActiveProviders([...activeProviders, provider]);
    }
    setShowAddMenu(false);
  };

  const handleRemoveProvider = (provider: Provider) => {
    setActiveProviders(activeProviders.filter((p) => p !== provider));
  };

  const availableToAdd = AVAILABLE_PROVIDERS.filter(
    (p) => !activeProviders.includes(p)
  );

  return (
    <main className="min-h-screen bg-void text-white font-body">
      {/* Header */}
      <header className="sticky top-0 z-50 border-b border-neon-cyan/20 bg-bg-dark/95 backdrop-blur-md">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-4">
            {/* Logo */}
            <div className="relative">
              <div className="w-10 h-10 rounded-lg bg-gradient-to-br from-neon-cyan to-neon-magenta flex items-center justify-center animate-neon-pulse">
                <svg className="w-6 h-6 text-black" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
              </div>
            </div>

            {/* Title */}
            <div>
              <h1
                className="text-2xl font-display font-black tracking-widest text-glitch"
                data-text="LLM-VIZ"
                style={{
                  background: 'linear-gradient(90deg, var(--neon-cyan), var(--neon-magenta))',
                  WebkitBackgroundClip: 'text',
                  WebkitTextFillColor: 'transparent',
                  backgroundClip: 'text',
                }}
              >
                LLM-VIZ
              </h1>
              <p className="text-xs opacity-50 uppercase tracking-widest">
                Neural Network Monitor
              </p>
            </div>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-3">
            {/* Add Provider */}
            {availableToAdd.length > 0 && (
              <div className="relative">
                <button
                  onClick={() => setShowAddMenu(!showAddMenu)}
                  className="btn-neon btn-cyan px-4 py-2 rounded-lg font-display text-sm font-bold uppercase tracking-wider"
                >
                  + Add Provider
                </button>

                {showAddMenu && (
                  <div className="absolute top-full right-0 mt-2 w-48 rounded-lg border-2 border-neon-cyan bg-black/95 p-2 shadow-2xl backdrop-blur-xl">
                    {availableToAdd.map((provider) => (
                      <button
                        key={provider}
                        onClick={() => handleAddProvider(provider)}
                        className="w-full text-left px-3 py-2 rounded hover:bg-neon-cyan/20 text-sm capitalize transition-colors font-medium"
                      >
                        {provider === 'anthropic' && '🤖 '}{provider === 'openai' && '🌐 '}
                        {provider}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            )}

            {/* Keyboard shortcuts */}
            <div className="hidden xl:flex items-center gap-1 text-xs opacity-50">
              <kbd>Ctrl</kbd> <span>+</span> <kbd>K</kbd>
              <span className="ml-1">Focus</span>
            </div>
          </div>
        </div>
      </header>

      {/* Main Dashboard */}
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {activeProviders.length === 0 ? (
          /* Empty State */
          <div className="flex flex-col items-center justify-center min-h-[60vh] text-center">
            <div className="mb-6">
              <div className="w-24 h-24 rounded-full border-4 border-neon-cyan/30 flex items-center justify-center mx-auto animate-neon-pulse">
                <svg className="w-12 h-12 text-neon-cyan" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
              </div>
            </div>
            <h2 className="text-3xl font-display font-bold mb-2 text-neon-cyan">
              NO PROVIDERS ACTIVE
            </h2>
            <p className="text-sm opacity-50 mb-6 max-w-md">
              Add a provider to start monitoring real-time token usage, costs, and performance metrics.
            </p>
            <button
              onClick={() => setShowAddMenu(true)}
              className="btn-neon btn-cyan px-6 py-3 rounded-lg font-display text-sm font-bold uppercase tracking-wider"
            >
              + Add Your First Provider
            </button>
          </div>
        ) : (
          /* Provider Grid */
          <div className={`grid gap-6 ${
            activeProviders.length === 1
              ? 'grid-cols-1 max-w-4xl mx-auto'
              : 'grid-cols-1 lg:grid-cols-2'
          }`}>
            {activeProviders.map((provider, index) => (
              <div
                key={provider}
                className={`stagger-${index + 1}`}
              >
                <ProviderCard
                  provider={provider}
                  onRemove={
                    activeProviders.length > 1
                      ? () => handleRemoveProvider(provider)
                      : undefined
                  }
                />
              </div>
            ))}
          </div>
        )}

        {/* Global Stats (if multiple providers) */}
        {activeProviders.length > 1 && (
          <div className="mt-8 p-6 rounded-2xl border border-neon-yellow/30 bg-neon-yellow/5 animate-fade-in-up stagger-3">
            <h3 className="text-xl font-display font-bold mb-4 uppercase tracking-wider text-neon-yellow">
              ◢ Global Statistics ◣
            </h3>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
              <div className="text-center">
                <div className="text-xs opacity-50 uppercase tracking-wider mb-1">Total Providers</div>
                <div className="text-3xl font-mono tabular-nums font-bold text-neon-yellow">
                  {activeProviders.length}
                </div>
              </div>
              <div className="text-center">
                <div className="text-xs opacity-50 uppercase tracking-wider mb-1">Total Cost</div>
                <div className="text-3xl font-mono tabular-nums font-bold text-neon-yellow">
                  $0.000
                </div>
              </div>
              <div className="text-center">
                <div className="text-xs opacity-50 uppercase tracking-wider mb-1">Uptime</div>
                <div className="text-3xl font-mono tabular-nums font-bold text-neon-green">
                  100%
                </div>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <footer className="mt-12 py-6 border-t border-white/5 text-center text-xs opacity-30">
        <p>NEURAL NETWORK MONITOR v2.0 // POWERED BY CYBERPUNK AESTHETICS</p>
      </footer>
    </main>
  );
}
