import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { useAuthStore } from './store/authStore';
import { Login } from './pages/Login';
import { AppShell } from './components/layout/AppShell';
import { APIKeys } from './pages/Settings/APIKeys';
import { Strategies } from './pages/Strategies';
import { Alerts } from './pages/Alerts';
import { Title } from '@tremor/react';

const queryClient = new QueryClient();

// Dashboard Component using Layout
const Dashboard = () => {
  return (
    <AppShell>
      <div className="mb-6">
        <Title>Dashboard</Title>
      </div>
      <div className="grid gap-6">
        {/* We can reuse the alert feed here for now */}
      </div>
    </AppShell>
  );
};

const Events = () => (

  <AppShell>
    <Title>Event Explorer</Title>
    <div className="mt-4">Global Events</div>
  </AppShell>
);

const ProtectedRoute = ({ children }: { children: React.ReactNode }) => {

  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" />;
};

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route
            path="/"
            element={
              <ProtectedRoute>
                <Dashboard />
              </ProtectedRoute>
            }
          />
          <Route
            path="/strategies"
            element={
              <ProtectedRoute>
                <Strategies />
              </ProtectedRoute>
            }
          />
          <Route
            path="/alerts"
            element={
              <ProtectedRoute>
                <Alerts />
              </ProtectedRoute>
            }
          />
          <Route
            path="/events"
            element={
              <ProtectedRoute>
                <Events />
              </ProtectedRoute>
            }
          />
          <Route
            path="/settings/keys"
            element={
              <ProtectedRoute>
                <APIKeys />
              </ProtectedRoute>
            }
          />
        </Routes>
      </BrowserRouter>
    </QueryClientProvider>
  );
}

export default App;
