import type { ControlMessage, ContentEvent, DisplayConfig } from '../types';
import { DisplayEventManager } from './events';

interface DisplayControlOptions {
  url: string;
  displayId: string;
  config: DisplayConfig;
  reconnectInterval?: number;
  maxReconnectAttempts?: number;
}

/**
 * Manages WebSocket connection and control protocol for displays.
 * Handles connection lifecycle, message processing, and health reporting.
 */
export class DisplayControl {
  private ws: WebSocket | null = null;
  private events: DisplayEventManager;
  private reconnectCount = 0;
  private reconnectTimer?: number;
  private messageQueue: ControlMessage[] = [];
  
  private readonly baseReconnectDelay: number;
  private readonly maxReconnectAttempts: number;

  constructor(
    private readonly options: DisplayControlOptions
  ) {
    this.events = new DisplayEventManager();
    this.baseReconnectDelay = options.reconnectInterval ?? 1000;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
  }

  /**
   * Initialize WebSocket connection
   */
  async connect(): Promise<void> {
    if (this.ws) {
      this.ws.close();
    }

    return new Promise((resolve, reject) => {
      try {
        this.ws = new WebSocket(this.options.url);
        
        this.ws.onopen = () => {
          this.onConnected();
          resolve();
        };
        
        this.ws.onclose = () => this.onDisconnected();
        this.ws.onerror = (error) => this.onError(error);
        this.ws.onmessage = (event) => this.onMessage(event);
      } catch (err) {
        reject(err);
      }
    });
  }

  /**
   * Send control message to server
   */
  send(message: ControlMessage): boolean {
    if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
      this.messageQueue.push(message);
      return false;
    }

    try {
      this.ws.send(JSON.stringify(message));
      return true;
    } catch (err) {
      console.error('Error sending message:', err);
      this.messageQueue.push(message);
      return false;
    }
  }

  /**
   * Report content health event
   */
  reportHealth(event: ContentEvent) {
    this.events.bufferHealthEvent(event);
  }

  /**
   * Subscribe to display events
   */
  on<K extends keyof EventMap>(
    event: K,
    handler: EventHandler<EventMap[K]>
  ) {
    this.events.on(event, handler);
  }

  /**
   * Unsubscribe from display events
   */
  off<K extends keyof EventMap>(
    event: K,
    handler: EventHandler<EventMap[K]>
  ) {
    this.events.off(event, handler);
  }

  private onConnected() {
    console.log('WebSocket connected');
    this.reconnectCount = 0;
    this.events.emit('reconnect', undefined);
    
    // Send authentication
    this.send({
      type: 'STATUS_RESPONSE',
      timestamp: new Date().toISOString(),
      messageId: crypto.randomUUID(),
      config: this.options.config
    });

    // Process queued messages
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift();
      if (message) {
        this.send(message);
      }
    }
  }

  private onDisconnected() {
    this.ws = null;
    this.events.emit('close', undefined);

    // Attempt reconnection
    if (this.reconnectCount < this.maxReconnectAttempts) {
      const delay = Math.min(
        this.baseReconnectDelay * Math.pow(2, this.reconnectCount),
        30000 // Max 30 second delay
      );
      
      this.reconnectTimer = window.setTimeout(() => {
        this.reconnectCount++;
        this.connect().catch(err => {
          console.error('Reconnection failed:', err);
        });
      }, delay);
    }
  }

  private onError(error: Event) {
    console.error('WebSocket error:', error);
    this.events.emit('error', {
      code: 'WEBSOCKET_ERROR',
      message: 'WebSocket connection error',
      details: error
    });
  }

  private async onMessage(event: MessageEvent) {
    try {
      const message = JSON.parse(event.data) as ControlMessage;
      
      switch (message.type) {
        case 'SEQUENCE_UPDATE':
          this.events.emit('sequence', message);
          break;
          
        case 'STATUS_REQUEST':
          // Send immediate status response
          this.send({
            type: 'STATUS_RESPONSE',
            timestamp: new Date().toISOString(),
            messageId: crypto.randomUUID(),
            status: message.status
          });
          break;
          
        case 'ERROR':
          this.events.emit('error', message.error!);
          break;
          
        default:
          console.warn('Unknown message type:', message.type);
      }
    } catch (err) {
      console.error('Error processing message:', err);
    }
  }

  /**
   * Clean up resources
   */
  dispose() {
    if (this.ws) {
      this.ws.close();
    }
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer);
    }
    this.events.dispose();
  }
}
