export interface ApiKey {
  id: string;
  provider: 'anthropic' | 'openai';
  maskedKey: string;
  label?: string;
}
