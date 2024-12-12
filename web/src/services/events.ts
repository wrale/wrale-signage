import type { ContentEvent, ContentError, ControlMessage } from '../types';
import { HealthEventBuffer } from './healthBuffer';

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

  private healthBuffer: HealthEventBuffer;
  
  constructor() {
    this.healthBuffer = new HealthEventBuffer(events => this.emitHealthEvents(events), {
      maxSize: 50,            // Max 50 events before force flush
      flushInterval: 5000,    // Regular flush every 5s
      maxErrorAge: 30000,     // Keep errors for 30s
      offlineTimeout: 30000   // Wait 30s between retries in offline mode
    });
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
   * Handle content health event
   */
  bufferHealthEvent(event: ContentEvent) {
    // Special handling for errors
    if (event.error) {
      this.emit('error', event.error);
    }
    
    // Add to health buffer
    this.healthBuffer.addEvent(event);
  }

  /**
   * Emit batch of health events to handlers
   */
  private emitHealthEvents(events: ContentEvent[]) {
    this.handlers.health?.forEach(handler => {
      try {
        // Send each event in batch to handler
        events.forEach(event => handler(event));
      } catch (err) {
        console.error('Error processing health events:', err);
      }
    });
  }

  dispose() {
    this.healthBuffer.dispose();
    this.handlers = {};
  }
}
