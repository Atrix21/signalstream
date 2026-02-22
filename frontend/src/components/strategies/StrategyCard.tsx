import { Card, Title, Text, Badge, Button, Flex } from '@tremor/react';
import type { Strategy } from '../../api/strategies';
import { TrashIcon, PlayIcon, PauseIcon } from '@heroicons/react/24/outline';
import { format } from 'date-fns';

interface StrategyCardProps {
  strategy: Strategy;
  onDelete: (id: string) => void;
  onToggle: (id: string) => void;
  isDeleting?: boolean;
  isToggling?: boolean;
}

export const StrategyCard = ({ 
  strategy, 
  onDelete, 
  onToggle, 
  isDeleting, 
  isToggling 
}: StrategyCardProps) => {
  return (
    <Card className="hover:shadow-tremor-card-action dark:hover:shadow-dark-tremor-card-action transition-shadow">
      <Flex alignItems="start" justifyContent="between" className="mb-4">
        <div>
          <Title>{strategy.name}</Title>
          <Text className="mt-1">{strategy.description}</Text>
        </div>
        <Badge color={strategy.is_active ? 'emerald' : 'slate'}>
          {strategy.is_active ? 'Active' : 'Paused'}
        </Badge>
      </Flex>

      <div className="space-y-3 mb-6">
        <div>
          <Text className="font-medium text-xs uppercase tracking-wide text-tremor-content-subtle">Query</Text>
          <Text className="font-mono text-sm bg-tremor-background-subtle dark:bg-dark-tremor-background-subtle p-2 rounded mt-1">
            {strategy.query}
          </Text>
        </div>
        
        <Flex justifyContent="start" className="gap-8">
          <div>
            <Text className="font-medium text-xs uppercase tracking-wide text-tremor-content-subtle">Sources</Text>
            <div className="flex gap-2 mt-1 flex-wrap">
              {strategy.source.length > 0 ? (
                strategy.source.map(source => (
                  <Badge key={source} size="xs" color="blue">{source}</Badge>
                ))
              ) : (
                <Text className="text-sm">All</Text>
              )}
            </div>
          </div>
          
          <div>
            <Text className="font-medium text-xs uppercase tracking-wide text-tremor-content-subtle">Tickers</Text>
            <div className="flex gap-2 mt-1 flex-wrap">
              {strategy.tickers.length > 0 ? (
                strategy.tickers.map(ticker => (
                  <Badge key={ticker} size="xs" color="violet">{ticker}</Badge>
                ))
              ) : (
                <Text className="text-sm">All</Text>
              )}
            </div>
          </div>

          <div>
            <Text className="font-medium text-xs uppercase tracking-wide text-tremor-content-subtle">Threshold</Text>
            <Text className="font-mono text-sm mt-1">{strategy.similarity_threshold}</Text>
          </div>
        </Flex>
      </div>

      <Flex className="border-t border-tremor-border dark:border-dark-tremor-border pt-4 mt-4">
        <Text className="text-xs text-tremor-content-subtle">
          Created {format(new Date(strategy.created_at), 'MMM d, yyyy')}
        </Text>
        <div className="flex gap-2">
          <Button
            size="xs"
            variant="secondary"
            icon={strategy.is_active ? PauseIcon : PlayIcon}
            onClick={() => onToggle(strategy.id)}
            loading={isToggling}
          >
            {strategy.is_active ? 'Pause' : 'Resume'}
          </Button>
          <Button
            size="xs"
            variant="secondary"
            color="rose"
            icon={TrashIcon}
            onClick={() => onDelete(strategy.id)}
            loading={isDeleting}
          >
            Delete
          </Button>
        </div>
      </Flex>
    </Card>
  );
};
