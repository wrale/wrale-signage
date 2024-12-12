import React, { forwardRef, useCallback, useEffect } from 'react';
import type { ContentItem, ContentEvent } from '../types';

interface ContentFrameProps {
  content: ContentItem;
  isActive: boolean;
  onLoad: () => void;
  onError: (error: Error) => void;
  onHealthEvent: (event: ContentEvent) => void;
}

export const ContentFrame = forwardRef<HTMLIFrameElement, ContentFrameProps>(
  ({ content, isActive, onLoad, onError, onHealthEvent }, ref) => {
    const handleLoad = useCallback(() => {
      if (!ref || !('current' in ref) || !ref.current) return;

      try {
        const frame = ref.current;
        const loadTime = performance.now();

        // Report content loaded
        onHealthEvent({
          type: 'CONTENT_LOADED',
          contentUrl: content.url,
          timestamp: Date.now(),
          metrics: {
            loadTime,
            renderTime: loadTime, // Initial estimate
            resourceStats: {
              // These could be enhanced with more detailed stats
              imageCount: 0,
              scriptCount: 0,
              totalBytes: 0
            }
          }
        });

        // Signal frame is ready
        frame.contentWindow?.postMessage({ type: 'CONTENT_READY' }, '*');
        onLoad();
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Frame load failed');
        onError(error);
      }
    }, [content.url, onLoad, onError, onHealthEvent]);

    const handleMessage = useCallback((event: MessageEvent) => {
      if (event.origin !== window.location.origin) return;
      
      if (event.data?.type === 'VIDEO_LOADED') {
        onLoad();
      } else if (event.data?.type === 'ERROR') {
        onError(new Error(event.data.message));
      } else if (event.data?.type === 'CONTENT_EVENT') {
        onHealthEvent(event.data.event);
      }
    }, [onLoad, onError, onHealthEvent]);

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
          timestamp: Date.now()
        });
      } else {
        onHealthEvent({
          type: 'CONTENT_HIDDEN',
          contentUrl: content.url,
          timestamp: Date.now()
        });
      }
    }, [isActive, content.url, onHealthEvent]);

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
