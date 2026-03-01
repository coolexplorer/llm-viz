import { describe, it, expect } from 'vitest';
import { getModelLimit, MODEL_CONTEXT_LIMITS, PROVIDER_MODELS } from '@/lib/model-limits';

describe('getModelLimit', () => {
  it('returns 200K for claude-sonnet-4-6', () => {
    expect(getModelLimit('claude-sonnet-4-6')).toBe(200_000);
  });

  it('returns 200K for claude-opus-4-6', () => {
    expect(getModelLimit('claude-opus-4-6')).toBe(200_000);
  });

  it('returns 1M for claude-opus-4-6-1m', () => {
    expect(getModelLimit('claude-opus-4-6-1m')).toBe(1_000_000);
  });

  it('returns 128K for gpt-4o', () => {
    expect(getModelLimit('gpt-4o')).toBe(128_000);
  });

  it('returns 200K for o1', () => {
    expect(getModelLimit('o1')).toBe(200_000);
  });

  it('defaults to 128K for unknown model', () => {
    expect(getModelLimit('totally-unknown-model')).toBe(128_000);
  });

  it('returns 16385 for gpt-3.5-turbo', () => {
    expect(getModelLimit('gpt-3.5-turbo')).toBe(16_385);
  });
});

describe('MODEL_CONTEXT_LIMITS', () => {
  it('contains entries for anthropic models', () => {
    expect(MODEL_CONTEXT_LIMITS['claude-sonnet-4-6']).toBeDefined();
    expect(MODEL_CONTEXT_LIMITS['claude-haiku-4-5']).toBeDefined();
  });

  it('contains entries for openai models', () => {
    expect(MODEL_CONTEXT_LIMITS['gpt-4o']).toBeDefined();
    expect(MODEL_CONTEXT_LIMITS['o3-mini']).toBeDefined();
  });
});

describe('PROVIDER_MODELS', () => {
  it('includes openai models', () => {
    expect(PROVIDER_MODELS['openai']).toContain('gpt-4o');
    expect(PROVIDER_MODELS['openai']).toContain('gpt-4o-mini');
    expect(PROVIDER_MODELS['openai']).toContain('o1');
  });

  it('includes anthropic models', () => {
    expect(PROVIDER_MODELS['anthropic']).toContain('claude-sonnet-4-6');
    expect(PROVIDER_MODELS['anthropic']).toContain('claude-opus-4-6');
    expect(PROVIDER_MODELS['anthropic']).toContain('claude-haiku-4-5');
  });

  it('openai has at least 3 models', () => {
    expect(PROVIDER_MODELS['openai'].length).toBeGreaterThanOrEqual(3);
  });

  it('anthropic has at least 3 models', () => {
    expect(PROVIDER_MODELS['anthropic'].length).toBeGreaterThanOrEqual(3);
  });
});
