import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { getStrategies, deleteStrategy, toggleStrategy } from '../api/strategies';
import { AppShell } from '../components/layout/AppShell';
import { StrategyCard } from '../components/strategies/StrategyCard';
import { StrategyForm } from '../components/strategies/StrategyForm';
import { Title, Text, Button, Grid } from '@tremor/react';
import { PlusIcon } from '@heroicons/react/24/outline';
import type { Strategy } from '../api/strategies';

export const Strategies = () => {
  const queryClient = useQueryClient();
  const [isFormOpen, setIsFormOpen] = useState(false);
  
  const { data: strategies, isLoading } = useQuery({ 
    queryKey: ['strategies'], 
    queryFn: getStrategies 
  });

  const deleteMutation = useMutation({
    mutationFn: deleteStrategy,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['strategies'] }),
  });

  const toggleMutation = useMutation({
    mutationFn: toggleStrategy,
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['strategies'] }),
  });

  const handleDelete = (id: string) => {
    if (confirm('Are you sure you want to delete this strategy?')) {
      deleteMutation.mutate(id);
    }
  };

  const handleToggle = (id: string) => {
    toggleMutation.mutate(id);
  };

  if (isLoading) return <AppShell>Loading strategies...</AppShell>;

  return (
    <AppShell>
      <div className="flex justify-between items-center mb-6">
        <div>
          <Title>Strategies</Title>
          <Text>Manage your semantic search criteria</Text>
        </div>
        <Button icon={PlusIcon} onClick={() => setIsFormOpen(true)}>
          New Strategy
        </Button>
      </div>

      <Grid numItems={1} numItemsMd={2} numItemsLg={3} className="gap-6">
        {strategies?.map((strategy: Strategy) => (
          <StrategyCard
            key={strategy.id}
            strategy={strategy}
            onDelete={handleDelete}
            onToggle={handleToggle}
            isDeleting={deleteMutation.isPending && deleteMutation.variables === strategy.id}
            isToggling={toggleMutation.isPending && toggleMutation.variables === strategy.id}
          />
        ))}
      </Grid>

      {strategies?.length === 0 && (
        <div className="text-center py-20 bg-tremor-background-subtle dark:bg-dark-tremor-background-subtle rounded-tremor-default border-dashed border-2 border-tremor-border dark:border-dark-tremor-border">
          <Text>No strategies found. Create one to start monitoring events.</Text>
        </div>
      )}

      <StrategyForm isOpen={isFormOpen} onClose={() => setIsFormOpen(false)} />
    </AppShell>
  );
};
