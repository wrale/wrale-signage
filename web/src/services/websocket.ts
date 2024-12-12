import type { ControlMessage, ContentEvent, DisplayConfig } from '../types';
import type { EventMap, EventHandler } from '../types/events';
import { ENDPOINTS } from '../constants';
import { DisplayEventManager } from './events';
import { RegistrationService } from './registration';

interface DisplayControlOptions {
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
  private registration: RegistrationService;
  private reconnectCount = 0;
  private reconnectTimer?: number;
  private messageQueue: ControlMessage[] = [];
  private disposed = false;
  
  private readonly baseReconnectDelay: number;
  private readonly maxReconnectAttempts: number;

  constructor(
    private readonly options: DisplayControlOptions
  ) {
    this.events = new DisplayEventManager();
    this.registration = new RegistrationService(options.config);
    
    this.baseReconnectDelay = options.reconnectInterval ?? 1000;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
  }

  /**
   * Initialize WebSocket connection
   */
  async connect(): Promise<void> {
    if (this.disposed) {
      throw new Error('DisplayControl has been disposed');
    }

    // Close existing connection if any
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }

    try {
      // Ensure we have a valid token
      const token = await this.registration.getValidToken();
      
      // Build WebSocket URL with auth token
      const url = new URL(ENDPOINTS.displays.ws, window.location.origin);
      url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
      url.searchParams.set('access_token', token);

      return new Promise((resolve, reject) => {
        try {
          this.ws = new WebSocket(url.toString());
          
          this.ws.onopen = () => {
            this.onConnected();
            resolve();
          };
          
          this.ws.onclose = (event) => {
            // Check if closed due to auth failure
            if (event.code === 4001) { // Custom code for auth failure
              this.registration.startRegistration().catch(err => {
                console.error('Failed to start registration:', err);
              });
            }
            this.onDisconnected();
          };

          this.ws.onerror = (error) => this.onError(error);
          this.ws.onmessage = (event) => this.onMessage(event);
          
        } catch (err) {
          reject(err);
        }
      });

    } catch (err) {
      // Registration/token error
      this.events.emit('error', {
        code: 'AUTH_ERROR',
        message: 'Failed to get valid auth token',
        details: err
      });
      throw err;
    }
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
    
    // Update server status using registration service
    const displayId = localStorage.getItem(`display-id-${this.options.displayId}`);
    if (displayId) {
      const interceptor = this.registration.createFetchInterceptor();
      interceptor(ENDPOINTS.displays.lastSeen(displayId), {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json'
        }
      }).catch(err => {
        console.error('Failed to update last seen:', err);
      });
    }

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
    this.reconnect();
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

  private async reconnect() {
    // Clear any existing reconnect timer
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer);
    }

    // Don't reconnect if disposed
    if (this.disposed) {
      return;
    }

    // Attempt reconnection if within limits
    if (this.reconnectCount < this.maxReconnectAttempts) {
      const delay = Math.min(
        this.baseReconnectDelay * Math.pow(2, this.reconnectCount),
        30000 // Max 30 second delay
      );
      
      this.reconnectTimer = window.setTimeout(async () => {
        this.reconnectCount++;

        try {
          await this.connect();
        } catch (err) {
          console.error('Reconnection failed:', err);
          this.reconnect(); // Try again with increased delay
        }
      }, delay);
    } else {
      // Max retries exceeded, emit error
      this.events.emit('error', {
        code: 'MAX_RETRIES_EXCEEDED',
        message: 'Maximum reconnection attempts exceeded',
      });
    }
  }

  /**
   * Clean up resources
   */
  dispose() {
    this.disposed = true;
    if (this.ws) {
      this.ws.close();
    }
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer);
    }
    this.events.dispose();
    this.registration.dispose();
  }
}