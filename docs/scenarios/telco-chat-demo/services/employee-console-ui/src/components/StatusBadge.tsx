import type { ReportStatus } from '../types';

const LABELS: Record<ReportStatus, string> = {
  open: 'Open',
  in_progress: 'In progress',
  resolved: 'Resolved',
};

export function StatusBadge({ status }: { status: ReportStatus }) {
  return <span className={`status-pill status-${status}`}>{LABELS[status] ?? status}</span>;
}
