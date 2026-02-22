import client from './client';

export interface Alert {
  id: string;
  user_id: string;
  strategy_id: string;
  event_id: string;
  title: string;
  content: string;
  url: string;
  similarity_score: number;
  is_read: boolean;
  created_at: string;
}

export const getAlerts = async (limit = 50, offset = 0): Promise<Alert[]> => {
  const { data } = await client.get(`/alerts?limit=${limit}&offset=${offset}`);
  return data;
};

export const markAlertRead = async (id: string) => {
  const { data } = await client.patch(`/alerts/read?id=${id}`);
  return data;
};
