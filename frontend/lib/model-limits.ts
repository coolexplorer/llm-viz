export const MODEL_CONTEXT_LIMITS: Record<string, number> = {
  // Anthropic
  'claude-opus-4-6': 200_000,
  'claude-sonnet-4-6': 200_000,
  'claude-haiku-4-5': 200_000,
  'claude-sonnet-4-5': 200_000,
  'claude-opus-4-5': 200_000,
  // Anthropic 1M beta
  'claude-opus-4-6-1m': 1_000_000,
  'claude-sonnet-4-6-1m': 1_000_000,
  // OpenAI
  'gpt-4o': 128_000,
  'gpt-4o-mini': 128_000,
  'gpt-3.5-turbo': 16_385,
  'o1': 200_000,
  'o1-mini': 128_000,
  'o3-mini': 200_000,
};

export function getModelLimit(model: string): number {
  return MODEL_CONTEXT_LIMITS[model] ?? 128_000;
}

export const PROVIDER_MODELS: Record<string, string[]> = {
  openai: ['gpt-4o', 'gpt-4o-mini', 'gpt-3.5-turbo', 'o1', 'o1-mini', 'o3-mini'],
  anthropic: [
    'claude-opus-4-6',
    'claude-sonnet-4-6',
    'claude-haiku-4-5',
    'claude-sonnet-4-5',
    'claude-opus-4-5',
  ],
};
