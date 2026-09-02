import { useEffect, useState } from 'react';
import { getReportDetail, listReports, updateReport } from '../api';
import type { Report, ReportDetail, ReportStatus } from '../types';
import { StatusBadge } from './StatusBadge';

const STATUSES: ReportStatus[] = ['open', 'in_progress', 'resolved'];

interface IncidentsTabProps {
  actorId: string;
  /** Set by a parent (e.g. "view this customer's report" from the
   * Customers tab) to force-open a specific report. Consumed once. */
  openReportId?: string | null;
  onOpenReportConsumed?: () => void;
}

function ReportRow({ report, onOpen }: { report: Report; onOpen: (id: string) => void }) {
  return (
    <li className="report-mini-list-item" onClick={() => onOpen(report.id)}>
      <span className="mono">{report.id}</span>
      <span>{report.category}</span>
      <StatusBadge status={report.status} />
    </li>
  );
}

export function IncidentsTab({ actorId, openReportId, onOpenReportConsumed }: IncidentsTabProps) {
  const [statusFilter, setStatusFilter] = useState('');
  const [categoryFilter, setCategoryFilter] = useState('');
  const [reports, setReports] = useState<Report[]>([]);
  const [listLoading, setListLoading] = useState(false);
  const [listError, setListError] = useState<string | null>(null);

  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<ReportDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailError, setDetailError] = useState<string | null>(null);

  const [statusDraft, setStatusDraft] = useState<ReportStatus>('open');
  const [notesDraft, setNotesDraft] = useState('');
  const [saving, setSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const loadReports = async () => {
    setListLoading(true);
    setListError(null);
    try {
      setReports(
        await listReports(actorId, {
          status: statusFilter || undefined,
          category: categoryFilter.trim() || undefined,
        }),
      );
    } catch (err) {
      setListError(err instanceof Error ? err.message : 'Failed to load incidents');
    } finally {
      setListLoading(false);
    }
  };

  useEffect(() => {
    loadReports();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const openReport = async (id: string) => {
    setSelectedId(id);
    setDetail(null);
    setSaveError(null);
    setDetailLoading(true);
    setDetailError(null);
    try {
      const d = await getReportDetail(actorId, id);
      setDetail(d);
      setStatusDraft(d.report.status);
      setNotesDraft(d.report.resolutionNotes ?? '');
    } catch (err) {
      setDetailError(err instanceof Error ? err.message : 'Failed to load incident');
    } finally {
      setDetailLoading(false);
    }
  };

  // A customer-tab click ("view this report") force-opens a report once.
  useEffect(() => {
    if (openReportId) {
      openReport(openReportId);
      onOpenReportConsumed?.();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [openReportId]);

  const handleSave = async () => {
    if (!selectedId) return;
    setSaving(true);
    setSaveError(null);
    try {
      const updated = await updateReport(actorId, selectedId, {
        status: statusDraft,
        resolutionNotes: notesDraft,
      });
      setDetail((prev) => (prev ? { ...prev, report: updated } : prev));
      setReports((prev) => prev.map((r) => (r.id === updated.id ? updated : r)));
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save changes');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="dashboard-split">
      <div className="dashboard-list">
        <form
          className="list-toolbar"
          onSubmit={(event) => {
            event.preventDefault();
            loadReports();
          }}
        >
          <select value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
            <option value="">All statuses</option>
            {STATUSES.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          <input
            type="text"
            placeholder="Category"
            value={categoryFilter}
            onChange={(event) => setCategoryFilter(event.target.value)}
          />
          <button type="submit" className="secondary">
            Filter
          </button>
          <button type="button" className="secondary" onClick={loadReports}>
            Refresh
          </button>
        </form>

        {listError && <p className="error-text">{listError}</p>}
        {listLoading && <p className="muted-hint">Loading…</p>}

        <table className="data-table">
          <thead>
            <tr>
              <th>Id</th>
              <th>Customer</th>
              <th>Category</th>
              <th>Status</th>
              <th>Description</th>
            </tr>
          </thead>
          <tbody>
            {reports.map((r) => (
              <tr key={r.id} className={r.id === selectedId ? 'selected' : ''} onClick={() => openReport(r.id)}>
                <td className="mono">{r.id}</td>
                <td className="mono">{r.customerId}</td>
                <td>{r.category}</td>
                <td>
                  <StatusBadge status={r.status} />
                </td>
                <td className="truncate">{r.description}</td>
              </tr>
            ))}
            {!listLoading && reports.length === 0 && (
              <tr>
                <td colSpan={5} className="muted-hint">
                  No incidents match these filters.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <div className="dashboard-detail">
        {!selectedId && <p className="muted-hint">Select an incident to view details.</p>}
        {detailLoading && <p className="muted-hint">Loading incident…</p>}
        {detailError && <p className="error-text">{detailError}</p>}

        {detail && (
          <div className="detail-card">
            <div className="detail-header">
              <div>
                <h2 className="mono">{detail.report.id}</h2>
                <p className="muted-hint">
                  {detail.report.customerId} · {detail.report.category} · filed{' '}
                  {new Date(detail.report.createdAt).toLocaleString()}
                </p>
              </div>
              <StatusBadge status={detail.report.status} />
            </div>

            <p>{detail.report.description}</p>

            <section className="edit-form">
              <h3>Update</h3>
              <label>
                Status
                <select value={statusDraft} onChange={(event) => setStatusDraft(event.target.value as ReportStatus)}>
                  {STATUSES.map((s) => (
                    <option key={s} value={s}>
                      {s}
                    </option>
                  ))}
                </select>
              </label>
              <label>
                Resolution notes
                <textarea
                  rows={3}
                  value={notesDraft}
                  onChange={(event) => setNotesDraft(event.target.value)}
                  placeholder="What was done to resolve this?"
                />
              </label>
              {saveError && <p className="error-text">{saveError}</p>}
              <button type="button" onClick={handleSave} disabled={saving}>
                {saving ? 'Saving…' : 'Save'}
              </button>
            </section>

            <section>
              <h3>Other reports for {detail.report.customerId}</h3>
              {detail.relatedByCustomer.length === 0 && <p className="muted-hint">None.</p>}
              <ul className="report-mini-list">
                {detail.relatedByCustomer.map((r) => (
                  <ReportRow key={r.id} report={r} onOpen={openReport} />
                ))}
              </ul>
            </section>

            <section>
              <h3>Similar reports ({detail.report.category})</h3>
              {detail.relatedByCategory.length === 0 && <p className="muted-hint">None.</p>}
              <ul className="report-mini-list">
                {detail.relatedByCategory.map((r) => (
                  <ReportRow key={r.id} report={r} onOpen={openReport} />
                ))}
              </ul>
            </section>
          </div>
        )}
      </div>
    </div>
  );
}
