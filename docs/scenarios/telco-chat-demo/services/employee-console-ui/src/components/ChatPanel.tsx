import { useEffect, useRef, useState } from 'react';
import type { FormEvent } from 'react';
import type { ConnectionStatus, TranscriptItem } from '../types';

interface Props {
  items: TranscriptItem[];
  status: ConnectionStatus;
  isWaiting: boolean;
  onSend: (content: string) => void;
  /** When set, sending is disabled and this hint replaces the input placeholder. */
  disabledHint?: string | null;
}

export function ChatPanel({ items, status, isWaiting, onSend, disabledHint }: Props) {
  const [draft, setDraft] = useState('');
  const bottomRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [items, isWaiting]);

  const blocked = status !== 'open' || !!disabledHint;
  const canSend = !blocked && draft.trim().length > 0;

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    if (!canSend) return;
    onSend(draft.trim());
    setDraft('');
  };

  return (
    <div className="chat-panel">
      <div className="transcript">
        {items.length === 0 && !isWaiting && <p className="empty-hint">No messages yet — say hello.</p>}
        {items.map((item) => (
          <TranscriptRow key={item.id} item={item} />
        ))}
        {isWaiting && (
          <div className="bubble assistant typing" aria-label="Assistant is typing">
            <span className="dot" />
            <span className="dot" />
            <span className="dot" />
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <form className="composer" onSubmit={handleSubmit}>
        <input
          type="text"
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          disabled={blocked}
          placeholder={
            status === 'connecting'
              ? 'Connecting…'
              : status !== 'open'
                ? 'Not connected'
                : (disabledHint ?? 'Type a message')
          }
        />
        <button type="submit" disabled={!canSend}>
          Send
        </button>
      </form>

      {status !== 'open' && (
        <p className="connection-hint">
          {status === 'connecting' && 'Connecting to chat…'}
          {status === 'closed' && 'Connection closed.'}
          {status === 'error' && 'Connection error — check the chat gateway URL.'}
        </p>
      )}
    </div>
  );
}

function TranscriptRow({ item }: { item: TranscriptItem }) {
  switch (item.kind) {
    case 'user':
      return <div className="bubble user">{item.text}</div>;
    case 'assistant':
      return (
        <div className="bubble assistant">
          {item.text}
          {item.streaming && <span className="cursor" />}
        </div>
      );
    case 'tool_call':
      return (
        <div className="chip tool-call" title={JSON.stringify(item.args, null, 2)}>
          🔧 {item.name}
          {item.audit && <span className="audit-badge">audited</span>}
        </div>
      );
    case 'tool_result':
      return (
        <div className="chip tool-result" title={JSON.stringify(item.result, null, 2)}>
          ✓ {item.name} result
        </div>
      );
    case 'error':
      return <div className="bubble error">⚠ {item.text}</div>;
    case 'divider':
      return <div className="divider">{item.text}</div>;
    default:
      return null;
  }
}
