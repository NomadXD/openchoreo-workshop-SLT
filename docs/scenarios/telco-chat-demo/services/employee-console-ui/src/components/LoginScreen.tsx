import { useState } from 'react';
import type { FormEvent } from 'react';

interface Props {
  loading: boolean;
  error: string | null;
  onLogin: (agentId: string) => void;
}

export function LoginScreen({ loading, error, onLogin }: Props) {
  const [agentId, setAgentId] = useState('');

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const id = agentId.trim();
    if (id) onLogin(id);
  };

  return (
    <div className="auth-screen">
      <form className="auth-card" onSubmit={handleSubmit}>
        <h1>Employee Console</h1>
        <p className="subtitle">Sign in with your agent id to assist customers.</p>

        <label htmlFor="agentId">Agent id</label>
        <input
          id="agentId"
          type="text"
          placeholder="agent-007"
          value={agentId}
          onChange={(event) => setAgentId(event.target.value)}
          autoFocus
        />

        {error && <p className="error-text">{error}</p>}

        <button type="submit" disabled={loading || !agentId.trim()}>
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  );
}
