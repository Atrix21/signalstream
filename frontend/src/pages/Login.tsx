import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuthStore } from '../store/authStore';
import client from '../api/client';
import { TextInput, Button, Card, Title, Text } from '@tremor/react';

export const Login = () => {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [isRegistering, setIsRegistering] = useState(false);
  const navigate = useNavigate();
  const login = useAuthStore((state) => state.login);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');

    try {
      const endpoint = isRegistering ? '/auth/register' : '/auth/login';
      const { data } = await client.post(endpoint, { email, password });
      
      login(data.token, data.user);
      navigate('/');
    } catch (err: any) {
      setError(err.response?.data?.error || 'An error occurred');
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-tremor-background-muted dark:bg-dark-tremor-background-muted p-4">
      <Card className="max-w-md w-full">
        <Title>{isRegistering ? 'Create Account' : 'Welcome Back'}</Title>
        <Text className="mb-6">
          {isRegistering 
            ? 'Sign up to start monitoring financial events' 
            : 'Login to your SignalStream dashboard'}
        </Text>

        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">
              Email
            </label>
            <TextInput
              type="email"
              placeholder="user@example.com"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              required
            />
          </div>
          <div>
            <label className="text-tremor-default text-tremor-content dark:text-dark-tremor-content">
              Password
            </label>
            <TextInput
              type="password"
              placeholder="••••••••"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          {error && (
            <div className="text-red-500 text-sm">{error}</div>
          )}

          <Button type="submit" className="w-full">
            {isRegistering ? 'Sign Up' : 'Sign In'}
          </Button>
        </form>

        <div className="mt-4 text-center">
          <button
            type="button"
            className="text-sm text-blue-500 hover:underline"
            onClick={() => setIsRegistering(!isRegistering)}
          >
            {isRegistering 
              ? 'Already have an account? Sign in' 
              : "Don't have an account? Sign up"}
          </button>
        </div>
      </Card>
    </div>
  );
};
