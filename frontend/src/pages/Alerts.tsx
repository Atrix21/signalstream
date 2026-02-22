import { AppShell } from '../components/layout/AppShell';
import { AlertFeed } from '../components/alerts/AlertFeed';
import { Title, Text } from '@tremor/react';

export const Alerts = () => {
  return (
    <AppShell>
      <div className="mb-6">
        <Title>Alerts</Title>
        <Text>Real-time notifications from your strategies</Text>
      </div>
      <AlertFeed />
    </AppShell>
  );
};
