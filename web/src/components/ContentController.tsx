import React, { useEffect, useRef, useState } from 'react';
import { ContentFrame } from './ContentFrame';
import { TransitionOverlay } from './TransitionOverlay';
import { DisplayControl } from '../services/websocket';
import type { ContentSequence, DisplayConfig, ContentItem, ContentEvent, DisplayStatus } from '../types';

interface ContentControllerProps {
  config: DisplayConfig;
  wsURL: string;
}

export const ContentController: React.FC<ContentControllerProps> = ({
  config,
  wsURL
}) => {
  // Display state
  const [sequence, setSequence] = useState<ContentSequence | null>(null);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [isTransitioning, setIsTransitioning] = useState(false);
  const [status, setStatus] = useState<DisplayStatus>({
    displayId: config.displayId,
    currentUrl: '',
    currentSequenceVersion: '',
    state: 'LOADING',
    updatedAt: new Date().toISOString()
  });

  // Refs
  const primaryFrameRef = useRef<HTMLIFrameElement>(null);
  const secondaryFrameRef = useRef<HTMLIFrameElement>(null);
  const controlRef = useRef<DisplayControl | null>(null);
  const contentTimerRef = useRef<number>();

  // Initialize WebSocket control
  useEffect(() => {
    controlRef.current = new DisplayControl({
      url: wsURL,
      displayId: config.displayId,
      config
    });

    // Handle sequence updates
    controlRef.current.on('sequence', (message) => {
      if (message.sequence) {
        setSequence(message.sequence);
        setCurrentIndex(0);
        scheduleNextContent(message.sequence.items[0]);
      }
    });

    // Handle errors
    controlRef.current.on('error', (error) => {
      setStatus(prev => ({
        ...prev,
        state: 'ERROR',
        lastError: error,
        updatedAt: new Date().toISOString()
      }));
    });

    // Connect
    controlRef.current.connect().catch(error => {
      console.error('Failed to connect:', error);
      setStatus(prev => ({
        ...prev,
        state: 'ERROR',
        lastError: {
          code: 'CONNECTION_ERROR',
          message: 'Failed to connect to control service'
        },
        updatedAt: new Date().toISOString()
      }));
    });

    return () => {
      controlRef.current?.dispose();
      if (contentTimerRef.current) {
        window.clearTimeout(contentTimerRef.current);
      }
    };
  }, [config, wsURL]);

  // Schedule next content change
  const scheduleNextContent = (content: ContentItem) => {
    if (contentTimerRef.current) {
      window.clearTimeout(contentTimerRef.current);
    }

    const duration = typeof content.duration === 'object' 
      ? content.duration.value ?? 10000 // Default 10s
      : content.duration;

    contentTimerRef.current = window.setTimeout(() => {
      if (sequence) {
        const nextIndex = (currentIndex + 1) % sequence.items.length;
        handleContentTransition(nextIndex);
      }
    }, duration);
  };

  // Handle content transitions
  const handleContentTransition = async (nextIndex: number) => {
    if (!sequence) return;

    try {
      setIsTransitioning(true);
      const nextContent = sequence.items[nextIndex];

      // Wait for transition duration
      await new Promise(resolve => 
        setTimeout(resolve, nextContent.transition.duration)
      );

      setCurrentIndex(nextIndex);
      scheduleNextContent(nextContent);

      setStatus(prev => ({
        ...prev,
        currentUrl: nextContent.url,
        state: 'ACTIVE',
        updatedAt: new Date().toISOString()
      }));

    } catch (err) {
      console.error('Transition failed:', err);
      
      setStatus(prev => ({
        ...prev,
        state: 'ERROR',
        lastError: {
          code: 'TRANSITION_ERROR',
          message: 'Failed to transition to next content'
        },
        updatedAt: new Date().toISOString()
      }));

    } finally {
      setIsTransitioning(false);
    }
  };

  // Handle content load errors
  const handleContentError = (error: Error) => {
    console.error('Content error:', error);
    
    setStatus(prev => ({
      ...prev,
      state: 'ERROR',
      lastError: {
        code: 'CONTENT_ERROR',
        message: error.message
      },
      updatedAt: new Date().toISOString()
    }));

    // Try to recover by moving to next item
    if (sequence) {
      const nextIndex = (currentIndex + 1) % sequence.items.length;
      handleContentTransition(nextIndex);
    }
  };

  // Report health events to control service
  const handleHealthEvent = (event: ContentEvent) => {
    if (controlRef.current) {
      controlRef.current.reportHealth(event);
    }
  };

  // Render content frames
  if (!sequence) {
    return null;
  }

  const currentContent = sequence.items[currentIndex];
  const nextContent = sequence.items[(currentIndex + 1) % sequence.items.length];

  return (
    <div className="relative w-full h-full overflow-hidden bg-black">
      <ContentFrame
        ref={primaryFrameRef}
        content={currentContent}
        isActive={!isTransitioning}
        onLoad={() => scheduleNextContent(currentContent)}
        onError={handleContentError}
        onHealthEvent={handleHealthEvent}
      />
      
      <ContentFrame
        ref={secondaryFrameRef}
        content={nextContent}
        isActive={isTransitioning}
        onLoad={() => {}}
        onError={handleContentError}
        onHealthEvent={handleHealthEvent}
      />

      <TransitionOverlay
        isActive={isTransitioning}
        config={currentContent.transition}
      />
    </div>
  );
};
