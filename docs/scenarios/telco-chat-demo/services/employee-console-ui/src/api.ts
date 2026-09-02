import { config } from './config';
import type { CustomerDetail, CustomerSummary, Report, ReportDetail } from './types';

interface LoginResponse {
  token: string;
}

export async function loginEmployee(agentId: string): Promise<string> {
  const res = await fetch(`${config.chatGatewayHttpUrl}/api/auth/employee/login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentId }),
  });

  if (!res.ok) {
    throw new Error(`Sign in failed (HTTP ${res.status})`);
  }

  const data = (await res.json()) as LoginResponse;
  if (!data.token) {
    throw new Error('Sign in response did not include a token');
  }
  return data.token;
}

// ---------- Dashboard data (subscription-service / network-ops-service) ----------
//
// Both services sit behind a platform-level Gateway API CORS filter that
// mistranslates `allowHeaders: ['*']` into an empty allow-list on the
// preflight response (a kgateway limitation, not something either service
// controls) — any request carrying a custom header or an explicit
// `Content-Type` fails preflight no matter what the app itself sends.
// Deliberately CORS-simple requests as the workaround: no custom headers
// (there's no employee-console login token these services would check
// anyway, and no X-Actor-* attribution — see the top-level demo README's
// note on the audit-trail gap this creates), and no manual Content-Type,
// so the browser applies its own safelisted default (`text/plain`) even
// on `updateReport`'s JSON body — the Go handlers decode the raw bytes
// regardless of what Content-Type says.

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, init);
  if (!res.ok) {
    throw new Error(`Request to ${url} failed (HTTP ${res.status})`);
  }
  return (await res.json()) as T;
}

export async function listCustomers(search?: string): Promise<CustomerSummary[]> {
  const url = new URL('/customers', config.subscriptionServiceUrl);
  if (search) url.searchParams.set('search', search);
  return fetchJSON<CustomerSummary[]>(url.toString());
}

// Composed client-side from three calls, one per backend concern — same
// composition chat-agent's get_customer_account tool does server-side,
// just done here since the dashboard talks to both services directly.
export async function getCustomerDetail(customerId: string): Promise<CustomerDetail> {
  const subBase = config.subscriptionServiceUrl;
  const netBase = config.networkOpsServiceUrl;

  const [profile, subscription, usageHistory, reports] = await Promise.all([
    fetchJSON(`${subBase}/customers/${customerId}`),
    fetchJSON(`${subBase}/customers/${customerId}/subscription`),
    fetchJSON(`${netBase}/customers/${customerId}/usage/history?days=7`),
    fetchJSON<Report[]>(`${netBase}/reports?${new URLSearchParams({ customerId }).toString()}`),
  ]);

  return {
    profile: profile as CustomerSummary,
    subscription: subscription as CustomerDetail['subscription'],
    usageHistory: usageHistory as CustomerDetail['usageHistory'],
    reports,
  };
}

export interface ReportFilters {
  status?: string;
  category?: string;
  customerId?: string;
}

export async function listReports(filters: ReportFilters = {}): Promise<Report[]> {
  const url = new URL('/reports', config.networkOpsServiceUrl);
  if (filters.status) url.searchParams.set('status', filters.status);
  if (filters.category) url.searchParams.set('category', filters.category);
  if (filters.customerId) url.searchParams.set('customerId', filters.customerId);
  return fetchJSON<Report[]>(url.toString());
}

// Composed client-side: the report itself, plus this customer's other
// reports and other customers' reports in the same category — the
// "related incidents" panel.
export async function getReportDetail(reportId: string): Promise<ReportDetail> {
  const netBase = config.networkOpsServiceUrl;
  const report = await fetchJSON<Report>(`${netBase}/reports/${reportId}`);

  const [relatedByCustomer, relatedByCategory] = await Promise.all([
    fetchJSON<Report[]>(
      `${netBase}/reports?${new URLSearchParams({ customerId: report.customerId, excludeId: reportId }).toString()}`,
    ),
    fetchJSON<Report[]>(
      `${netBase}/reports?${new URLSearchParams({ category: report.category, excludeId: reportId }).toString()}`,
    ),
  ]);

  return { report, relatedByCustomer, relatedByCategory };
}

export async function updateReport(
  reportId: string,
  body: { status?: string; resolutionNotes?: string },
): Promise<Report> {
  return fetchJSON<Report>(`${config.networkOpsServiceUrl}/reports/${reportId}`, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}
