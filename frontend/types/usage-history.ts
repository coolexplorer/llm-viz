export interface UsageHistoryPoint {
  date: string;
  input_tokens: number;
  output_tokens: number;
  total_tokens: number;
  cost_usd: number;
  request_count?: number;
}

export interface UsageHistoryResponse {
  provider: string;
  start_date: string;
  end_date: string;
  data_points: UsageHistoryPoint[];
  total_cost: number;
}
