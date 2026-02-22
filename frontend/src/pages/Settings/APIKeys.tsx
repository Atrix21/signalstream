import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getAPIKeys, updateAPIKey, deleteAPIKey } from '../../api/keys';
import { AppShell } from '../../components/layout/AppShell';
import { Card, Title, Text, Button, TextInput, Badge } from '@tremor/react';
import { TrashIcon, CheckCircleIcon, XCircleIcon } from '@heroicons/react/24/outline';

const PROVIDERS = [
  { id: 'polygon', name: 'Polygon.io', description: 'Financial news and market data' },
  { id: 'openai', name: 'OpenAI', description: 'AI embeddings and analysis' },
];

export const APIKeys = () => {
  const queryClient = useQueryClient();
  const { data: keys, isLoading } = useQuery({ queryKey: ['api-keys'], queryFn: getAPIKeys });
  
  const [inputKeys, setInputKeys] = useState<Record<string, string>>({});

  const updateMutation = useMutation({
    mutationFn: ({ provider, key }: { provider: string; key: string }) => 
      updateAPIKey(provider, key),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
      setInputKeys({});
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAPIKey,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['api-keys'] });
    },
  });

  const handleSave = (provider: string) => {
    const key = inputKeys[provider];
    if (key) {
      updateMutation.mutate({ provider, key });
    }
  };

  const handleDelete = (provider: string) => {
    if (confirm('Are you sure you want to delete this API key?')) {
      deleteMutation.mutate(provider);
    }
  };

  if (isLoading) return <AppShell>Loading...</AppShell>;

  return (
    <AppShell>
      <div className="mb-6">
        <Title>API Keys</Title>
        <Text>Manage external service connections</Text>
      </div>

      <div className="grid gap-6">
        {PROVIDERS.map((provider) => {
          const keyStatus = keys?.find((k) => k.provider === provider.id);
          const hasKey = keyStatus?.has_key;

          return (
            <Card key={provider.id}>
              <div className="flex justify-between items-start mb-4">
                <div>
                  <Title>{provider.name}</Title>
                  <Text>{provider.description}</Text>
                </div>
                <Badge 
                  icon={hasKey ? CheckCircleIcon : XCircleIcon}
                  color={hasKey ? 'emerald' : 'rose'}
                >
                  {hasKey ? 'Connected' : 'Not Connected'}
                </Badge>
              </div>

              <div className="flex gap-4 items-end">
                <div className="flex-1">
                  <label className="text-sm text-tremor-content-default dark:text-dark-tremor-content-default mb-1 block">
                    {hasKey ? 'Update API Key' : 'Enter API Key'}
                  </label>
                  <TextInput
                    type="password"
                    placeholder="sk-..."
                    value={inputKeys[provider.id] || ''}
                    onChange={(e) => setInputKeys({ ...inputKeys, [provider.id]: e.target.value })}
                  />
                </div>
                <Button 
                  onClick={() => handleSave(provider.id)}
                  loading={updateMutation.isPending}
                  disabled={!inputKeys[provider.id]}
                >
                  Save
                </Button>
                {hasKey && (
                  <Button 
                    variant="secondary" 
                    color="rose"
                    icon={TrashIcon}
                    onClick={() => handleDelete(provider.id)}
                    loading={deleteMutation.isPending}
                  >
                    Delete
                  </Button>
                )}
              </div>
            </Card>
          );
        })}
      </div>
    </AppShell>
  );
};
