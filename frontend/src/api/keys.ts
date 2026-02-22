import client from './client';

export interface APIKeyStatus {
  provider: string;
  has_key: boolean;
}

export const getAPIKeys = async (): Promise<APIKeyStatus[]> => {
  const { data } = await client.get('/keys');
  return data;
};

export const updateAPIKey = async (provider: string, key: string) => {
  const { data } = await client.post('/keys', { provider, key });
  return data;
};

export const deleteAPIKey = async (provider: string) => {
  const { data } = await client.delete(`/keys?provider=${provider}`);
  return data;
};
