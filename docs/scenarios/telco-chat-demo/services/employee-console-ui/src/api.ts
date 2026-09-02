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
// Neither service requires auth — there's no employee-console login token
// to attach here. `X-Actor-Id` is sent anyway: both services log it on
// every request (see their READMEs), so an employee's browsing/edits are
// at least attributable in server logs even without a queryable audit
// trail (that only exists for actions routed through chat-gateway).

async function fetchJSON<T>(url: string, actorId: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: {
      ...(init?.body ? { 'Content-Type': 'application/json' } : {}),
      'X-Actor-Role': 'employee',
      'X-Actor-Id': actorId,
      ...init?.headers,
    },
  });
  if (!res.ok) {
    throw new Error(`Request to ${url} failed (HTTP ${res.status})`);
  }
  return (await res.json()) as T;
}

export async function listCustomers(actorId: string, search?: string): Promise<CustomerSummary[]> {
  const url = new URL('/customers', config.subscriptionServiceUrl);
  if (search) url.searchParams.set('search', search);
  return fetchJSON<CustomerSummary[]>(url.toString(), actorId);
}

// Composed client-side from three calls, one per backend concern — same
// composition chat-agent's get_customer_account tool does server-side,
// just done here since the dashboard talks to both services directly.
export async function getCustomerDetail(actorId: string, customerId: string): Promise<CustomerDetail> {
  const subBase = config.subscriptionServiceUrl;
  const netBase = config.networkOpsServiceUrl;

  const [profile, subscription, usageHistory, reports] = await Promise.all([
    fetchJSON(`${subBase}/customers/${customerId}`, actorId),
    fetchJSON(`${subBase}/customers/${customerId}/subscription`, actorId),
    fetchJSON(`${netBase}/customers/${customerId}/usage/history?days=7`, actorId),
    fetchJSON<Report[]>(
      `${netBase}/reports?${new URLSearchParams({ customerId }).toString()}`,
      actorId,
    ),
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

export async function listReports(actorId: string, filters: ReportFilters = {}): Promise<Report[]> {
  const url = new URL('/reports', config.networkOpsServiceUrl);
  if (filters.status) url.searchParams.set('status', filters.status);
  if (filters.category) url.searchParams.set('category', filters.category);
  if (filters.customerId) url.searchParams.set('customerId', filters.customerId);
  return fetchJSON<Report[]>(url.toString(), actorId);
}

// Composed client-side: the report itself, plus this customer's other
// reports and other customers' reports in the same category — the
// "related incidents" panel.
export async function getReportDetail(actorId: string, reportId: string): Promise<ReportDetail> {
  const netBase = config.networkOpsServiceUrl;
  const report = await fetchJSON<Report>(`${netBase}/reports/${reportId}`, actorId);

  const [relatedByCustomer, relatedByCategory] = await Promise.all([
    fetchJSON<Report[]>(
      `${netBase}/reports?${new URLSearchParams({ customerId: report.customerId, excludeId: reportId }).toString()}`,
      actorId,
    ),
    fetchJSON<Report[]>(
      `${netBase}/reports?${new URLSearchParams({ category: report.category, excludeId: reportId }).toString()}`,
      actorId,
    ),
  ]);

  return { report, relatedByCustomer, relatedByCategory };
}

export async function updateReport(
  actorId: string,
  reportId: string,
  body: { status?: string; resolutionNotes?: string },
): Promise<Report> {
  return fetchJSON<Report>(`${config.networkOpsServiceUrl}/reports/${reportId}`, actorId, {
    method: 'PATCH',
    body: JSON.stringify(body),
  });
}
