import { useState } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { ChatPanel } from './components/ChatPanel';
import { useChatSocket } from './useChatSocket';
import { loginCustomer } from './api';

export default function App() {
  // Token lives only in component state — session-scoped, never persisted.
  const [token, setToken] = useState<string | null>(null);
  const [customerId, setCustomerId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);

  const { items, status, isWaiting, sendMessage } = useChatSocket(token);

  const handleLogin = async (id: string) => {
    setLoading(true);
    setLoginError(null);
    try {
      const nextToken = await loginCustomer(id);
      setToken(nextToken);
      setCustomerId(id);
    } catch (err) {
      setLoginError(err instanceof Error ? err.message : 'Sign in failed');
    } finally {
      setLoading(false);
    }
  };

  const handleSignOut = () => {
    setToken(null);
    setCustomerId(null);
    setLoginError(null);
  };

  if (!token || !customerId) {
    return <LoginScreen loading={loading} error={loginError} onLogin={handleLogin} />;
  }

  return (
    <div className="app-shell">
      <header className="account-header">
        <div>
          <span className="eyebrow">Your account</span>
          <h1>{customerId}</h1>
        </div>
        <button type="button" className="secondary" onClick={handleSignOut}>
          Sign out
        </button>
      </header>

      
      <div className="promo-banner">
        🎉 New: the Unlimited plan now includes free roaming in 12 countries.
      </div>
     

      <ChatPanel items={items} status={status} isWaiting={isWaiting} onSend={sendMessage} />
    </div>
  );
}
