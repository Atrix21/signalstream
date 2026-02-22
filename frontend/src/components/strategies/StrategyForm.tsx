import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createStrategy } from '../../api/strategies';
import { Dialog, DialogPanel, Title, TextInput, Textarea, Button, NumberInput, MultiSelect, MultiSelectItem } from '@tremor/react';

interface StrategyFormProps {
  isOpen: boolean;
  onClose: () => void;
}

const SOURCES = ['Polygon.io', 'SEC EDGAR', 'NewsAPI'];

export const StrategyForm = ({ isOpen, onClose }: StrategyFormProps) => {
  const queryClient = useQueryClient();
  const [formData, setFormData] = useState({
    name: '',
    description: '',
    query: '',
    source: [] as string[],
    tickers: [] as string[],
    similarity_threshold: 0.5,
    is_active: true,
  });
  
  // Helper for tickers text input -> array
  const [tickerInput, setTickerInput] = useState('');

  const mutation = useMutation({
    mutationFn: createStrategy,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['strategies'] });
      onClose();
      // Reset form
      setFormData({
        name: '',
        description: '',
        query: '',
        source: [],
        tickers: [],
        similarity_threshold: 0.5,
        is_active: true,
      });
      setTickerInput('');
    },
  });

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const tickers = tickerInput
      .split(',')
      .map(t => t.trim().toUpperCase())
      .filter(t => t.length > 0);

    mutation.mutate({
      name: formData.name, 
      query: formData.query,
      similarity_threshold: formData.similarity_threshold,
      is_active: formData.is_active,
      source: formData.source,
      description: formData.description,
      tickers: tickers,
    });
  };

  return (
    <Dialog open={isOpen} onClose={onClose} static={true} className="z-50">
      <DialogPanel className="max-w-2xl w-full">
        <Title className="mb-4">Create New Strategy</Title>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">Name</label>
            <TextInput 
              required
              placeholder="e.g., Tech Earnings Watch" 
              value={formData.name}
              onChange={e => setFormData({...formData, name: e.target.value})}
            />
          </div>

          <div>
            <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">Description</label>
            <TextInput 
              placeholder="Brief description of this strategy" 
              value={formData.description}
              onChange={e => setFormData({...formData, description: e.target.value})}
            />
          </div>

          <div>
            <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">Semantic Query</label>
            <Textarea 
              required
              rows={3}
              placeholder="Describe the events you want to find, e.g., 'Companies announcing stock buybacks or dividend increases'"
              value={formData.query}
              onChange={e => setFormData({...formData, query: e.target.value})}
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">Sources</label>
              <MultiSelect
                value={formData.source}
                onValueChange={(val) => setFormData({...formData, source: val})}
                placeholder="Select sources..."
              >
                {SOURCES.map(source => (
                  <MultiSelectItem key={source} value={source}>{source}</MultiSelectItem>
                ))}
              </MultiSelect>
            </div>
            <div>
              <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">Similarity Threshold (0.0 - 1.0)</label>
              <NumberInput 
                min={0} 
                max={1} 
                step={0.05}
                value={formData.similarity_threshold}
                onValueChange={(val) => setFormData({...formData, similarity_threshold: val})}
              />
            </div>
          </div>

          <div>
            <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">Tickers (Comma separated)</label>
            <TextInput 
              placeholder="AAPL, MSFT, TSLA" 
              value={tickerInput}
              onChange={e => setTickerInput(e.target.value)}
            />
          </div>

          <div className="flex justify-end space-x-2 mt-6">
            <Button variant="secondary" onClick={onClose} type="button">
              Cancel
            </Button>
            <Button type="submit" loading={mutation.isPending}>
              Create Strategy
            </Button>
          </div>
        </form>
      </DialogPanel>
    </Dialog>
  );
};
