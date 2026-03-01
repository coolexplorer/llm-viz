export interface TokenDataPoint {
  timestamp: number;
  provider: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheCreationTokens: number;
  totalTokens: number;
  costUSD: number;
}

export interface ProviderConfig {
  name: string;
  models: string[];
  supportsCache: boolean;
}

export interface SessionStats {
  totalRequests: number;
  totalInputTokens: number;
  totalOutputTokens: number;
  totalCacheReadTokens: number;
  totalCacheCreationTokens: number;
  totalCostUSD: number;
  cacheHitRate: number;
}

export interface ContextWindowStatus {
  model: string;
  maxTokens: number;
  currentUsed: number;
  utilizationPercent: number;
  remainingTokens: number;
  isWarning: boolean;
  isCritical: boolean;
}

export type Provider = 'openai' | 'anthropic';

export interface CompletionRequest {
  provider: Provider;
  model: string;
  apiKey: string;
  messages: { role: 'user' | 'assistant'; content: string }[];
  maxTokens?: number;
}
