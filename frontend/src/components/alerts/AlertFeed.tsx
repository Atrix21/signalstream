import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getAlerts, markAlertRead } from '../../api/alerts';
import type { Alert } from '../../api/alerts';
import { Card, Text, Title, Badge, Button, Flex } from '@tremor/react';
import { EyeIcon, ArrowTopRightOnSquareIcon } from '@heroicons/react/24/outline';
import { format } from 'date-fns';

export const AlertCard = ({ alert, onRead }: { alert: Alert, onRead: (id: string) => void }) => {
  return (
    <Card 
      className={`transition-all ${alert.is_read ? 'opacity-70' : 'border-l-4 border-l-tremor-brand dark:border-l-dark-tremor-brand'}`}
    >
      <Flex justifyContent="between" alignItems="start">
        <div className="flex-1">
          <Flex justifyContent="start" className="gap-2 mb-1">
            <Badge size="xs" color={alert.similarity_score > 0.8 ? 'red' : 'orange'}>
              Match: {(alert.similarity_score * 100).toFixed(0)}%
            </Badge>
            <Text className="text-xs text-tremor-content-subtle">
              {format(new Date(alert.created_at), 'MMM d, h:mm a')}
            </Text>
          </Flex>
          <Title className="text-base truncate pr-4">{alert.title}</Title>
          <Text className="mt-2 text-sm line-clamp-2">{alert.content}</Text>
        </div>
        <div className="flex flex-col gap-2 ml-4">
          {!alert.is_read && (
            <Button 
              size="xs" 
              variant="secondary" 
              icon={EyeIcon}
              onClick={() => onRead(alert.id)}
            >
              Read
            </Button>
          )}
          <a href={alert.url} target="_blank" rel="noopener noreferrer">
            <Button size="xs" variant="light" icon={ArrowTopRightOnSquareIcon}>
              View
            </Button>
          </a>
        </div>
      </Flex>
    </Card>
  );
};

export const AlertFeed = () => {
  const queryClient = useQueryClient();
  const { data: alerts, isLoading } = useQuery({ 
    queryKey: ['alerts'], 
    queryFn: () => getAlerts(),
    refetchInterval: 10000 // Poll every 10s since SSE is pending
  });

  const readMutation = useMutation({
    mutationFn: markAlertRead,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['alerts'] }),
  });

  if (isLoading) return <Text>Loading alerts...</Text>;

  return (
    <div className="space-y-4">
      {alerts?.map((alert) => (
        <AlertCard 
          key={alert.id} 
          alert={alert} 
          onRead={(id) => readMutation.mutate(id)} 
        />
      ))}
      {alerts?.length === 0 && (
        <div className="text-center py-10">
          <Text>No alerts yet. Create a strategy to start monitoring.</Text>
        </div>
      )}
    </div>
  );
};
