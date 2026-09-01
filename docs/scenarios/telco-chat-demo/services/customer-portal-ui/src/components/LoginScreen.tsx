import { useState } from 'react';
import type { FormEvent } from 'react';

const DEMO_CUSTOMERS = [
  { id: 'cust-001', label: 'Amara Perera' },
  { id: 'cust-002', label: 'Nadeesha Fernando' },
  { id: 'cust-003', label: 'Kasun Silva' },
  { id: 'cust-004', label: 'Ishara Jayawardena' },
];

interface Props {
  loading: boolean;
  error: string | null;
  onLogin: (customerId: string) => void;
}

export function LoginScreen({ loading, error, onLogin }: Props) {
  const [customerId, setCustomerId] = useState(DEMO_CUSTOMERS[0].id);
  const [useCustomId, setUseCustomId] = useState(false);
  const [customId, setCustomId] = useState('');

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    const id = (useCustomId ? customId : customerId).trim();
    if (id) onLogin(id);
  };

  const canSubmit = !loading && (useCustomId ? customId.trim().length > 0 : true);

  return (
    <div className="auth-screen">
      <form className="auth-card" onSubmit={handleSubmit}>
        <h1>Customer Portal</h1>
        <p className="subtitle">Sign in to chat with support.</p>

        <label htmlFor="customerId">Customer</label>
        {useCustomId ? (
          <input
            id="customerId"
            type="text"
            placeholder="cust-001"
            value={customId}
            onChange={(event) => setCustomId(event.target.value)}
            autoFocus
          />
        ) : (
          <select id="customerId" value={customerId} onChange={(event) => setCustomerId(event.target.value)}>
            {DEMO_CUSTOMERS.map((customer) => (
              <option key={customer.id} value={customer.id}>
                {customer.id} — {customer.label}
              </option>
            ))}
          </select>
        )}

        <button type="button" className="link-button" onClick={() => setUseCustomId((v) => !v)}>
          {useCustomId ? 'Choose from demo customers instead' : 'Enter a different customer id'}
        </button>

        {error && <p className="error-text">{error}</p>}

        <button type="submit" disabled={!canSubmit}>
          {loading ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
    </div>
  );
}
