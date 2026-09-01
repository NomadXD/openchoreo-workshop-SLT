import { useCallback, useEffect, useRef, useState } from 'react';
import { config } from './config';
import type { ConnectionStatus, ServerEvent, TranscriptItem } from './types';

function newId(): string {
  return crypto.randomUUID();
}

export function useChatSocket(token: string | null) {
  const [items, setItems] = useState<TranscriptItem[]>([]);
  const [status, setStatus] = useState<ConnectionStatus>('connecting');
  const [isWaiting, setIsWaiting] = useState(false);

  const wsRef = useRef<WebSocket | null>(null);
  // One conversationId for the life of the WS session, per contract: the
  // gateway is never asked which id to use, so we mint it once on open and
  // send it with every message.
  const conversationIdRef = useRef<string | null>(null);
  const currentAssistantIdRef = useRef<string | null>(null);

  useEffect(() => {
    if (!token) {
      setItems([]);
      setStatus('connecting');
      return;
    }

    conversationIdRef.current = newId();
    currentAssistantIdRef.current = null;
    setItems([]);
    setStatus('connecting');
    setIsWaiting(false);

    const url = `${config.chatGatewayWsUrl}/ws/chat?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(url);
    wsRef.current = ws;

    ws.onopen = () => setStatus('open');
    ws.onclose = () => setStatus('closed');
    ws.onerror = () => setStatus('error');

    ws.onmessage = (event) => {
      let payload: ServerEvent;
      try {
        payload = JSON.parse(event.data as string) as ServerEvent;
      } catch {
        return;
      }

      switch (payload.type) {
        case 'token': {
          setIsWaiting(false);
          setItems((prev) => {
            const activeId = currentAssistantIdRef.current;
            if (activeId) {
              return prev.map((item) =>
                item.id === activeId && item.kind === 'assistant'
                  ? { ...item, text: item.text + payload.content }
                  : item,
              );
            }
            const id = newId();
            currentAssistantIdRef.current = id;
            return [...prev, { id, kind: 'assistant', text: payload.content, streaming: true }];
          });
          break;
        }
        case 'tool_call': {
          setItems((prev) => [
            ...prev,
            { id: newId(), kind: 'tool_call', name: payload.name, args: payload.args, audit: payload.audit },
          ]);
          break;
        }
        case 'tool_result': {
          setItems((prev) => [
            ...prev,
            { id: newId(), kind: 'tool_result', name: payload.name, result: payload.result },
          ]);
          break;
        }
        case 'done': {
          setIsWaiting(false);
          const activeId = currentAssistantIdRef.current;
          if (activeId) {
            setItems((prev) =>
              prev.map((item) => (item.id === activeId && item.kind === 'assistant' ? { ...item, streaming: false } : item)),
            );
          }
          currentAssistantIdRef.current = null;
          break;
        }
        case 'error': {
          setIsWaiting(false);
          currentAssistantIdRef.current = null;
          setItems((prev) => [...prev, { id: newId(), kind: 'error', text: payload.message }]);
          break;
        }
      }
    };

    return () => {
      ws.close();
      wsRef.current = null;
    };
  }, [token]);

  const sendMessage = useCallback((content: string) => {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;

    setItems((prev) => [...prev, { id: newId(), kind: 'user', text: content }]);
    setIsWaiting(true);

    ws.send(
      JSON.stringify({
        type: 'message',
        conversationId: conversationIdRef.current,
        content,
      }),
    );
  }, []);

  return { items, status, isWaiting, sendMessage };
}
