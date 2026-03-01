// Prices per 1M tokens in USD
interface ModelPricing {
  inputPer1M: number;
  outputPer1M: number;
  cacheWritePer1M?: number;
  cacheReadPer1M?: number;
}

const MODEL_PRICING: Record<string, ModelPricing> = {
  // Anthropic (https://www.anthropic.com/pricing)
  'claude-opus-4-6': {
    inputPer1M: 15,
    outputPer1M: 75,
    cacheWritePer1M: 18.75,  // 25% surcharge
    cacheReadPer1M: 1.5,     // 90% discount
  },
  'claude-sonnet-4-6': {
    inputPer1M: 3,
    outputPer1M: 15,
    cacheWritePer1M: 3.75,
    cacheReadPer1M: 0.3,
  },
  'claude-haiku-4-5': {
    inputPer1M: 0.8,
    outputPer1M: 4,
    cacheWritePer1M: 1,
    cacheReadPer1M: 0.08,
  },
  'claude-sonnet-4-5': {
    inputPer1M: 3,
    outputPer1M: 15,
    cacheWritePer1M: 3.75,
    cacheReadPer1M: 0.3,
  },
  'claude-opus-4-5': {
    inputPer1M: 15,
    outputPer1M: 75,
    cacheWritePer1M: 18.75,
    cacheReadPer1M: 1.5,
  },
  // OpenAI
  'gpt-4o': {
    inputPer1M: 2.5,
    outputPer1M: 10,
    cacheReadPer1M: 1.25,
  },
  'gpt-4o-mini': {
    inputPer1M: 0.15,
    outputPer1M: 0.6,
    cacheReadPer1M: 0.075,
  },
  'gpt-3.5-turbo': {
    inputPer1M: 0.5,
    outputPer1M: 1.5,
  },
  'o1': {
    inputPer1M: 15,
    outputPer1M: 60,
    cacheReadPer1M: 7.5,
  },
  'o1-mini': {
    inputPer1M: 1.1,
    outputPer1M: 4.4,
    cacheReadPer1M: 0.55,
  },
  'o3-mini': {
    inputPer1M: 1.1,
    outputPer1M: 4.4,
    cacheReadPer1M: 0.55,
  },
};

export interface CostBreakdown {
  inputCost: number;
  outputCost: number;
  cacheWriteCost: number;
  cacheReadCost: number;
  totalCost: number;
  cacheSavings: number;
}

export function calculateCost(
  model: string,
  inputTokens: number,
  outputTokens: number,
  cacheCreationTokens = 0,
  cacheReadTokens = 0,
): CostBreakdown {
  const pricing = MODEL_PRICING[model] ?? MODEL_PRICING['gpt-4o-mini'];

  const inputCost = (inputTokens / 1_000_000) * pricing.inputPer1M;
  const outputCost = (outputTokens / 1_000_000) * pricing.outputPer1M;
  const cacheWriteCost = cacheCreationTokens > 0 && pricing.cacheWritePer1M
    ? (cacheCreationTokens / 1_000_000) * pricing.cacheWritePer1M
    : 0;
  const cacheReadCost = cacheReadTokens > 0 && pricing.cacheReadPer1M
    ? (cacheReadTokens / 1_000_000) * pricing.cacheReadPer1M
    : 0;

  // What cacheRead would have cost without cache
  const cacheReadSavings = cacheReadTokens > 0
    ? (cacheReadTokens / 1_000_000) * (pricing.inputPer1M - (pricing.cacheReadPer1M ?? 0))
    : 0;

  return {
    inputCost,
    outputCost,
    cacheWriteCost,
    cacheReadCost,
    totalCost: inputCost + outputCost + cacheWriteCost + cacheReadCost,
    cacheSavings: Math.max(0, cacheReadSavings),
  };
}

export function formatUSD(amount: number): string {
  if (amount < 0.01) return `$${(amount * 1000).toFixed(3)}m`;
  return `$${amount.toFixed(4)}`;
}

export function getModelPricing(model: string): ModelPricing {
  return MODEL_PRICING[model] ?? MODEL_PRICING['gpt-4o-mini'];
}
