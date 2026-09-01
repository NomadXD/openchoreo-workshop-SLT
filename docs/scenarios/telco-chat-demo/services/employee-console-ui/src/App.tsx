import { useState } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { ChatPanel } from './components/ChatPanel';
import { useChatSocket } from './useChatSocket';
import { loginEmployee } from './api';

export default function App() {
  // Token lives only in component state — session-scoped, never persisted.
  const [token, setToken] = useState<string | null>(null);
  const [agentId, setAgentId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [targetCustomerId, setTargetCustomerId] = useState('');

  const { items, status, isWaiting, sendMessage } = useChatSocket(token);

  const handleLogin = async (id: string) => {
    setLoading(true);
    setLoginError(null);
    try {
      const nextToken = await loginEmployee(id);
      setToken(nextToken);
      setAgentId(id);
    } catch (err) {
      setLoginError(err instanceof Error ? err.message : 'Sign in failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSignOut = () => {
    setToken(null);
    setAgentId(null);
    setTargetCustomerId('');
    setLoginError(null);
  };

  if (!token || !agentId) {
    return <LoginScreen loading={loading} error={loginError} onLogin={handleLogin} />;
  }

  const trimmedTarget = targetCustomerId.trim();

  return (
    <div className="app-shell">
      <header className="account-header">
        <div>
          <span className="eyebrow">Signed in as</span>
          <h1>{agentId}</h1>
        </div>
        <button type="button" className="secondary" onClick={handleSignOut}>
          Sign out
        </button>
      </header>

      <div className="assisting-bar">
        <label htmlFor="targetCustomerId">Assisting customer</label>
        <input
          id="targetCustomerId"
          type="text"
          placeholder="cust-002"
          value={targetCustomerId}
          onChange={(event) => setTargetCustomerId(event.target.value)}
        />
        {!trimmedTarget && <span className="assisting-hint">Set a customer id to enable chat</span>}
      </div>

      <ChatPanel
        items={items}
        status={status}
        isWaiting={isWaiting}
        onSend={(content) => sendMessage(content, trimmedTarget)}
        disabledHint={trimmedTarget ? null : 'Set an assisting customer first'}
      />
    </div>
  );
}
