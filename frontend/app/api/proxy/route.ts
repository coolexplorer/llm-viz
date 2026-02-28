import { NextRequest, NextResponse } from 'next/server';
import { broadcastTokenData } from '../stream/route';
import { calculateCost } from '@/lib/cost-calculator';
import type { TokenDataPoint } from '@/types/token-data';

interface OpenAIUsage {
  prompt_tokens: number;
  completion_tokens: number;
  prompt_tokens_details?: { cached_tokens?: number };
}

interface AnthropicUsage {
  input_tokens: number;
  output_tokens: number;
  cache_creation_input_tokens?: number;
  cache_read_input_tokens?: number;
}

function normalizeOpenAIUsage(
  usage: OpenAIUsage,
  model: string,
): Omit<TokenDataPoint, 'timestamp' | 'provider'> {
  const inputTokens = usage.prompt_tokens - (usage.prompt_tokens_details?.cached_tokens ?? 0);
  const cacheReadTokens = usage.prompt_tokens_details?.cached_tokens ?? 0;
  const outputTokens = usage.completion_tokens;
  const cost = calculateCost(model, inputTokens, outputTokens, 0, cacheReadTokens);

  return {
    model,
    inputTokens,
    outputTokens,
    cacheReadTokens,
    cacheCreationTokens: 0,
    totalTokens: inputTokens + outputTokens + cacheReadTokens,
    costUSD: cost.totalCost,
  };
}

function normalizeAnthropicUsage(
  usage: AnthropicUsage,
  model: string,
): Omit<TokenDataPoint, 'timestamp' | 'provider'> {
  const inputTokens = usage.input_tokens;
  const outputTokens = usage.output_tokens;
  const cacheReadTokens = usage.cache_read_input_tokens ?? 0;
  const cacheCreationTokens = usage.cache_creation_input_tokens ?? 0;
  const cost = calculateCost(model, inputTokens, outputTokens, cacheCreationTokens, cacheReadTokens);

  return {
    model,
    inputTokens,
    outputTokens,
    cacheReadTokens,
    cacheCreationTokens,
    totalTokens: inputTokens + outputTokens + cacheReadTokens + cacheCreationTokens,
    costUSD: cost.totalCost,
  };
}

export async function POST(request: NextRequest) {
  const body = await request.json() as {
    provider: 'openai' | 'anthropic';
    model: string;
    apiKey: string;
    messages: { role: string; content: string }[];
    maxTokens?: number;
  };

  const { provider, model, apiKey, messages, maxTokens = 1024 } = body;

  if (!apiKey) {
    return NextResponse.json({ error: 'API key required' }, { status: 400 });
  }

  try {
    let responseBody: Record<string, unknown>;

    if (provider === 'anthropic') {
      const res = await fetch('https://api.anthropic.com/v1/messages', {
        method: 'POST',
        headers: {
          'x-api-key': apiKey,
          'anthropic-version': '2023-06-01',
          'content-type': 'application/json',
        },
        body: JSON.stringify({
          model,
          max_tokens: maxTokens,
          messages,
        }),
      });

      if (!res.ok) {
        const err = await res.json() as { error?: { message?: string } };
        return NextResponse.json(
          { error: err.error?.message ?? 'Anthropic API error' },
          { status: res.status },
        );
      }

      responseBody = await res.json() as Record<string, unknown>;
      const usage = responseBody.usage as AnthropicUsage;
      const normalized = normalizeAnthropicUsage(usage, model);
      const dataPoint: TokenDataPoint = {
        timestamp: Date.now(),
        provider: 'anthropic',
        ...normalized,
      };
      broadcastTokenData(dataPoint);
    } else {
      const res = await fetch('https://api.openai.com/v1/chat/completions', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${apiKey}`,
          'content-type': 'application/json',
        },
        body: JSON.stringify({
          model,
          max_tokens: maxTokens,
          messages,
        }),
      });

      if (!res.ok) {
        const err = await res.json() as { error?: { message?: string } };
        return NextResponse.json(
          { error: err.error?.message ?? 'OpenAI API error' },
          { status: res.status },
        );
      }

      responseBody = await res.json() as Record<string, unknown>;
      const usage = responseBody.usage as OpenAIUsage;
      const normalized = normalizeOpenAIUsage(usage, model);
      const dataPoint: TokenDataPoint = {
        timestamp: Date.now(),
        provider: 'openai',
        ...normalized,
      };
      broadcastTokenData(dataPoint);
    }

    return NextResponse.json(responseBody);
  } catch (err) {
    const message = err instanceof Error ? err.message : 'Unknown error';
    return NextResponse.json({ error: message }, { status: 500 });
  }
}
