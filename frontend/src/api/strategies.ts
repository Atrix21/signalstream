import client from './client';

export interface Strategy {
  id: string;
  name: string;
  description: string;
  query: string;
  source: string[];
  tickers: string[];
  similarity_threshold: number;
  is_active: boolean;
  created_at: string;
}

export const getStrategies = async (): Promise<Strategy[]> => {
  const { data } = await client.get('/strategies');
  return data;
};

export const createStrategy = async (strategy: Partial<Strategy>) => {
  const { data } = await client.post('/strategies', strategy);
  return data;
};

export const deleteStrategy = async (id: string) => {
  const { data } = await client.delete(`/strategies?id=${id}`);
  return data;
};

export const toggleStrategy = async (id: string) => {
  const { data } = await client.patch(`/strategies/toggle?id=${id}`);
  return data;
};
