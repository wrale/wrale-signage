import type { ContentEvent, ContentError, ControlMessage } from '../types';

type EventHandler<T> = (event: T) => void;

interface EventMap {
  sequence: ControlMessage;
  status: ControlMessage;
  health: ContentEvent;
  error: ContentError;
  reconnect: void;
  close: void;
}

/**
 * Display event manager handling content events, health monitoring,
 * and control messages.
 */
export class DisplayEventManager {
  private handlers: {
    [K in keyof EventMap]?: Set<EventHandler<EventMap[K]>>;
  } = {};
  
  private healthEventBuffer: ContentEvent[] = [];
  private readonly maxBufferSize = 50;
  private flushTimeout?: number;
  
  constructor(private readonly flushInterval = 5000) {
    this.startFlushTimer();
  }

  on<K extends keyof EventMap>(event: K, handler: EventHandler<EventMap[K]>) {
    if (!this.handlers[event]) {
      this.handlers[event] = new Set();
    }
    this.handlers[event]?.add(handler);
  }

  off<K extends keyof EventMap>(event: K, handler: EventHandler<EventMap[K]>) {
    this.handlers[event]?.delete(handler);
  }

  emit<K extends keyof EventMap>(event: K, data: EventMap[K]) {
    this.handlers[event]?.forEach(handler => {
      try {
        handler(data);
      } catch (err) {
        console.error(`Error in ${event} handler:`, err);
      }
    });
  }

  /**
   * Buffer health events for batch processing
   */
  bufferHealthEvent(event: ContentEvent) {
    this.healthEventBuffer.push(event);
    
    // Flush if buffer is full
    if (this.healthEventBuffer.length >= this.maxBufferSize) {
      this.flushHealthEvents();
    }
  }

  /**
   * Flush buffered health events
   */
  private flushHealthEvents() {
    if (this.healthEventBuffer.length === 0) return;

    const events = this.healthEventBuffer;
    this.healthEventBuffer = [];

    this.handlers.health?.forEach(handler => {
      try {
        // Send events batch
        events.forEach(event => handler(event));
      } catch (err) {
        console.error('Error processing health events:', err);
      }
    });
  }

  private startFlushTimer() {
    this.flushTimeout = window.setInterval(
      () => this.flushHealthEvents(),
      this.flushInterval
    );
  }

  dispose() {
    if (this.flushTimeout) {
      window.clearInterval(this.flushTimeout);
    }
    this.handlers = {};
    this.healthEventBuffer = [];
  }
}
