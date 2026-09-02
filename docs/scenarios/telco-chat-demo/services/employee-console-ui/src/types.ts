// Server -> client WebSocket events, one JSON object per text frame.
export type ServerEvent =
  | { type: 'token'; content: string }
  | {
      type: 'tool_call';
      name: string;
      args: Record<string, unknown>;
      audit: boolean;
      targetCustomerId?: string;
    }
  | { type: 'tool_result'; name: string; result: unknown }
  | { type: 'done' }
  | { type: 'error'; message: string };

// Client -> server WebSocket message.
export interface ClientChatMessage {
  type: 'message';
  conversationId: string | null;
  content: string;
  targetCustomerId?: string;
}

// Local transcript model rendered by the chat panel. Distinct from
// ServerEvent: a single "assistant" item accumulates many "token" events.
export type TranscriptItem =
  | { id: string; kind: 'user'; text: string }
  | { id: string; kind: 'assistant'; text: string; streaming: boolean }
  | {
      id: string;
      kind: 'tool_call';
      name: string;
      args: Record<string, unknown>;
      audit: boolean;
    }
  | { id: string; kind: 'tool_result'; name: string; result: unknown }
  | { id: string; kind: 'error'; text: string }
  | { id: string; kind: 'divider'; text: string };

export type ConnectionStatus = 'connecting' | 'open' | 'closed' | 'error';

// ---------- Employee console data model (customers / reports) ----------

export interface CustomerSummary {
  id: string;
  name: string;
  msisdn: string;
  email: string;
}

export interface Plan {
  id: string;
  name: string;
  dataGb: number | null;
  priceCents: number;
}

export interface Subscription {
  customerId: string;
  plan: Plan;
}

export interface UsageEntry {
  customerId: string;
  date: string;
  browsingMb: number;
  streamingMb: number;
  socialMb: number;
  otherMb: number;
  totalMb: number;
}

export type ReportStatus = 'open' | 'in_progress' | 'resolved';

export interface Report {
  id: string;
  customerId: string;
  category: string;
  description: string;
  status: ReportStatus;
  resolutionNotes: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface CustomerDetail {
  profile: CustomerSummary;
  subscription: Subscription;
  usageHistory: UsageEntry[];
  reports: Report[];
}

export interface ReportDetail {
  report: Report;
  relatedByCustomer: Report[];
  relatedByCategory: Report[];
}
