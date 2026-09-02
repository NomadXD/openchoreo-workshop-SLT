import { useEffect, useState } from 'react';
import { getCustomerDetail, listCustomers } from '../api';
import type { CustomerDetail, CustomerSummary } from '../types';
import { StatusBadge } from './StatusBadge';

function formatPlan(plan: CustomerDetail['subscription']['plan']): string {
  const data = plan.dataGb === null ? 'Unlimited' : `${plan.dataGb}GB`;
  const price = (plan.priceCents / 100).toLocaleString('en-LK', { minimumFractionDigits: 2 });
  return `${plan.name} — ${data} — LKR ${price}/mo`;
}

interface CustomersTabProps {
  onOpenReport: (reportId: string) => void;
  onChatAbout: (customerId: string) => void;
}

export function CustomersTab({ onOpenReport, onChatAbout }: CustomersTabProps) {
  const [search, setSearch] = useState('');
  const [customers, setCustomers] = useState<CustomerSummary[]>([]);
  const [listLoading, setListLoading] = useState(false);
  const [listError, setListError] = useState<string | null>(null);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<CustomerDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);

  const loadCustomers = async (query?: string) => {
    setListLoading(true);
    setListError(null);
    try {
      setCustomers(await listCustomers(query));
    } catch (err) {
      setListError(err instanceof Error ? err.message : 'Failed to load customers');
    } finally {
      setListLoading(false);
    }
  };

  useEffect(() => {
    loadCustomers();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openCustomer = async (id: string) => {
    setSelectedId(id);
    setDetail(null);
    setDetailLoading(true);
    setDetailError(null);
    try {
      setDetail(await getCustomerDetail(id));
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : 'Failed to load customer');
    } finally {
      setDetailLoading(false);
    }
  };

  return (
    <div className="dashboard-split">
      <div className="dashboard-list">
        <form
          className="list-toolbar"
          onSubmit={(event) => {
            event.preventDefault();
            loadCustomers(search.trim() || undefined);
          }}
        >
          <input
            type="text"
            placeholder="Search name, id, or phone"
            value={search}
            onChange={(event) => setSearch(event.target.value)}
          />
          <button type="submit" className="secondary">
            Search
          </button>
          <button type="button" className="secondary" onClick={() => loadCustomers(search.trim() || undefined)}>
            Refresh
          </button>
        </form>

        {listError && <p className="error-text">{listError}</p>}
        {listLoading && <p className="muted-hint">Loading…</p>}

        <table className="data-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Id</th>
              <th>Phone</th>
            </tr>
          </thead>
          <tbody>
            {customers.map((c) => (
              <tr
                key={c.id}
                className={c.id === selectedId ? 'selected' : ''}
                onClick={() => openCustomer(c.id)}
              >
                <td>{c.name}</td>
                <td className="mono">{c.id}</td>
                <td>{c.msisdn}</td>
              </tr>
            ))}
            {!listLoading && customers.length === 0 && (
              <tr>
                <td colSpan={3} className="muted-hint">
                  No customers found.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="dashboard-detail">
        {!selectedId && <p className="muted-hint">Select a customer to view their account.</p>}
        {detailLoading && <p className="muted-hint">Loading account…</p>}
        {detailError && <p className="error-text">{detailError}</p>}

        {detail && (
          <div className="detail-card">
            <div className="detail-header">
              <div>
                <h2>{detail.profile.name}</h2>
                <p className="muted-hint">
                  {detail.profile.id} · {detail.profile.msisdn} · {detail.profile.email}
                </p>
              </div>
              <button type="button" className="secondary" onClick={() => onChatAbout(detail.profile.id)}>
                Chat about this customer
              </button>
            </div>

            <section>
              <h3>Subscription</h3>
              <p>{formatPlan(detail.subscription.plan)}</p>
            </section>

            <section>
              <h3>Usage — last 7 days</h3>
              <table className="data-table compact">
                <thead>
                  <tr>
                    <th>Date</th>
                    <th>Browsing</th>
                    <th>Streaming</th>
                    <th>Social</th>
                    <th>Other</th>
                    <th>Total</th>
                  </tr>
                </thead>
                <tbody>
                  {detail.usageHistory.map((u) => (
                    <tr key={u.date}>
                      <td>{u.date}</td>
                      <td>{u.browsingMb} MB</td>
                      <td>{u.streamingMb} MB</td>
                      <td>{u.socialMb} MB</td>
                      <td>{u.otherMb} MB</td>
                      <td>
                        <strong>{u.totalMb} MB</strong>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </section>

            <section>
              <h3>Service reports</h3>
              {detail.reports.length === 0 && <p className="muted-hint">No reports on file.</p>}
              <ul className="report-mini-list">
                {detail.reports.map((r) => (
                  <li key={r.id} onClick={() => onOpenReport(r.id)}>
                    <span className="mono">{r.id}</span>
                    <span>{r.category}</span>
                    <StatusBadge status={r.status} />
                  </li>
                ))}
              </ul>
            </section>
          </div>
        )}
      </div>
    </div>
  );
}
