import React, { forwardRef, useCallback, useEffect, useRef } from 'react';
import type { ContentItem, ContentEvent } from '../types';
import { monitorScript } from '../contentMonitor';

interface ContentFrameProps {
  content: ContentItem;
  isActive: boolean;
  onLoad: () => void;
  onError: (error: Error) => void;
  onHealthEvent: (event: ContentEvent) => void;
}

export const ContentFrame = forwardRef<HTMLIFrameElement, ContentFrameProps>(
  ({ content, isActive, onLoad, onError, onHealthEvent }, ref) => {
    const startTime = useRef(performance.now());
    const loadReported = useRef(false);

    // Handle initial frame load
    const handleLoad = useCallback(() => {
      if (!ref || !('current' in ref) || !ref.current) return;

      try {
        const frame = ref.current;
        
        // Inject monitoring script
        const script = document.createElement('script');
        script.textContent = monitorScript;
        frame.contentDocument?.head.appendChild(script);

        // Initialize frame
        frame.contentWindow?.postMessage({
          type: 'CONTENT_READY',
          config: {
            reportInterval: 5000, // Health check interval
            contentUrl: content.url,
            contentType: content.duration.type
          }
        }, '*');

        // Load reported through monitoring script
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Frame load failed');
        handleError(error);
      }
    }, [content, onLoad]);

    // Handle messages from frame
    const handleMessage = useCallback((event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      
      const { type, event: healthEvent } = event.data || {};

      if (type === 'CONTENT_EVENT') {
        // Process health event
        onHealthEvent({
          ...healthEvent,
          contentUrl: content.url, // Ensure correct URL
          timestamp: Date.now(),
          context: {
            isActive: String(isActive),
            duration: String(content.duration.value),
            durationType: content.duration.type
          }
        });

        // Signal load completion on first loaded event
        if (healthEvent.type === 'CONTENT_LOADED' && !loadReported.current) {
          loadReported.current = true;
          onLoad();
        }
      }
    }, [content, isActive, onLoad, onHealthEvent]);

    // Handle errors
    const handleError = useCallback((error: Error) => {
      onError(error);
      onHealthEvent({
        type: 'CONTENT_ERROR',
        contentUrl: content.url,
        timestamp: Date.now(),
        error: {
          code: 'LOAD_ERROR',
          message: error.message,
          details: error
        }
      });
    }, [content.url, onError, onHealthEvent]);

    // Set up message listener
    useEffect(() => {
      window.addEventListener('message', handleMessage);
      return () => window.removeEventListener('message', handleMessage);
    }, [handleMessage]);

    // Report visibility changes
    useEffect(() => {
      if (isActive) {
        onHealthEvent({
          type: 'CONTENT_VISIBLE',
          contentUrl: content.url,
          timestamp: Date.now(),
          context: {
            displayTime: String(performance.now() - startTime.current)
          }
        });
      } else {
        onHealthEvent({
          type: 'CONTENT_HIDDEN',
          contentUrl: content.url,
          timestamp: Date.now()
        });
      }
    }, [isActive, content.url, onHealthEvent]);

    // Reset tracking on content change
    useEffect(() => {
      startTime.current = performance.now();
      loadReported.current = false;
    }, [content.url]);

    return (
      <iframe
        ref={ref}
        src={content.url}
        className={`absolute inset-0 w-full h-full border-0 transition-opacity duration-500
          ${isActive ? 'opacity-100' : 'opacity-0 pointer-events-none'}`}
        onLoad={handleLoad}
        title={`Content frame for ${content.url}`}
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        sandbox="allow-scripts allow-same-origin allow-forms allow-downloads"
      />
    );
  }
);

ContentFrame.displayName = 'ContentFrame';