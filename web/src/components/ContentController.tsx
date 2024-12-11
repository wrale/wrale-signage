import React, { useEffect, useRef, useState } from 'react';
import { ContentSequence } from '../types';

interface ContentControllerProps {
  displayId: string;
  wsURL: string;
  onSequenceUpdate: (sequence: ContentSequence) => void;
  onReloadRequired: () => void;
}

export const ContentController: React.FC<ContentControllerProps> = ({
  displayId,
  wsURL,
  onSequenceUpdate,
  onReloadRequired
}) => {
  const ws = useRef<WebSocket | null>(null);
  const [currentUrl, setCurrentUrl] = useState<string>('');
  const [lastError, setLastError] = useState<string | null>(null);

  useEffect(() => {
    const connect = () => {
      const fullURL = `${wsURL}?id=${displayId}`;
      ws.current = new WebSocket(fullURL);

      ws.current.onmessage = (event) => {
        const message = JSON.parse(event.data);
        
        switch (message.type) {
          case 'SEQUENCE_UPDATE':
            if (message.sequence) {
              onSequenceUpdate(message.sequence);
            }
            break;
          case 'RELOAD':
            onReloadRequired();
            break;
        }
      };

      ws.current.onclose = () => {
        // Reconnect after delay
        setTimeout(connect, 5000);
      };

      ws.current.onerror = (error) => {
        setLastError(error.toString());
        sendStatus();
      };
    };

    connect();

    return () => {
      if (ws.current) {
        ws.current.close();
      }
    };
  }, [displayId, wsURL]);

  const sendStatus = () => {
    if (ws.current?.readyState === WebSocket.OPEN) {
      ws.current.send(JSON.stringify({
        type: 'STATUS',
        timestamp: new Date().toISOString(),
        status: {
          currentUrl,
          lastError,
          updatedAt: new Date().toISOString()
        }
      }));
    }
  };

  // Report status changes
  useEffect(() => {
    sendStatus();
  }, [currentUrl, lastError]);

  // Public interface
  return null; // Controller handles WebSocket logic only
};
