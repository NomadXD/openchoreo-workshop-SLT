import { useState } from 'react';
import { LoginScreen } from './components/LoginScreen';
import { ChatPanel } from './components/ChatPanel';
import { CustomersTab } from './components/CustomersTab';
import { IncidentsTab } from './components/IncidentsTab';
import { useChatSocket } from './useChatSocket';
import { loginEmployee } from './api';

type Tab = 'customers' | 'incidents' | 'chat';

export default function App() {
  // Token lives only in component state — session-scoped, never persisted.
  const [token, setToken] = useState<string | null>(null);
  const [agentId, setAgentId] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [loginError, setLoginError] = useState<string | null>(null);
  const [targetCustomerId, setTargetCustomerId] = useState('');

  const [activeTab, setActiveTab] = useState<Tab>('customers');
  // Set by "view this report" (Customers tab) / "similar reports" links
  // (Incidents tab itself) to force the Incidents tab to open a specific
  // report; consumed once, then cleared.
  const [jumpToReportId, setJumpToReportId] = useState<string | null>(null);

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
    setActiveTab('customers');
  };

  const openReportInIncidents = (reportId: string) => {
    setJumpToReportId(reportId);
    setActiveTab('incidents');
  };

  const chatAboutCustomer = (customerId: string) => {
    setTargetCustomerId(customerId);
    setActiveTab('chat');
  };

  if (!token || !agentId) {
    return <LoginScreen loading={loading} error={loginError} onLogin={handleLogin} />;
  }

  const trimmedTarget = targetCustomerId.trim();

  return (
    <div className="app-shell dashboard-shell">
      <header className="account-header">
        <div>
          <span className="eyebrow">Signed in as</span>
          <h1>{agentId}</h1>
        </div>
        <button type="button" className="secondary" onClick={handleSignOut}>
          Sign out
        </button>
      </header>

      <nav className="tab-bar">
        <button
          type="button"
          className={activeTab === 'customers' ? 'tab active' : 'tab'}
          onClick={() => setActiveTab('customers')}
        >
          Customers
        </button>
        <button
          type="button"
          className={activeTab === 'incidents' ? 'tab active' : 'tab'}
          onClick={() => setActiveTab('incidents')}
        >
          Incidents
        </button>
        <button
          type="button"
          className={activeTab === 'chat' ? 'tab active' : 'tab'}
          onClick={() => setActiveTab('chat')}
        >
          Chat
        </button>
      </nav>

      {activeTab === 'customers' && (
        <CustomersTab actorId={agentId} onOpenReport={openReportInIncidents} onChatAbout={chatAboutCustomer} />
      )}

      {activeTab === 'incidents' && (
        <IncidentsTab
          actorId={agentId}
          openReportId={jumpToReportId}
          onOpenReportConsumed={() => setJumpToReportId(null)}
        />
      )}

      {activeTab === 'chat' && (
        <>
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
        </>
      )}
    </div>
  );
}
