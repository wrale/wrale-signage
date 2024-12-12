import type { ControlMessage, ContentEvent, DisplayConfig } from '../types';
import { ENDPOINTS } from '../constants';
import { DisplayEventManager } from './events';
import { RegistrationService } from './registration';
import { AuthService } from './auth';

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
  private auth: AuthService;
  private reconnectCount = 0;
  private reconnectTimer?: number;
  private messageQueue: ControlMessage[] = [];
  private tokenUnsubscribe?: () => void;
  
  private readonly baseReconnectDelay: number;
  private readonly maxReconnectAttempts: number;

  constructor(
    private readonly options: DisplayControlOptions
  ) {
    this.events = new DisplayEventManager();
    this.auth = new AuthService(options.displayId);
    this.registration = new RegistrationService(
      options.config,
      null // Auth events handled directly
    );

    // Listen for token changes
    this.tokenUnsubscribe = this.auth.onTokensChanged(tokens => {
      if (!tokens) {
        // Lost auth, need to reconnect
        this.reconnect();
      }
    });
    
    this.baseReconnectDelay = options.reconnectInterval ?? 1000;
    this.maxReconnectAttempts = options.maxReconnectAttempts ?? 10;
  }

  /**
   * Initialize WebSocket connection
   */
  async connect(): Promise<void> {
    // Get valid token or start registration
    const token = this.auth.getAccessToken();
    if (!token) {
      // Need to register first
      const code = await this.registration.startRegistration();
      
      // Show activation screen until activated
      this.events.emit('activation', code);
      
      try {
        await this.registration.pollForActivation(
          code.deviceCode,
          code.pollInterval
        );
      } catch (err) {
        throw new Error('Display activation failed');
      }
    }

    // Close existing connection if any
    if (this.ws) {
      this.ws.close();
    }

    return new Promise((resolve, reject) => {
      try {
        // Get fresh token
        const token = this.auth.getAccessToken();
        if (!token) {
          throw new Error('No valid auth token available');
        }

        // Build WebSocket URL
        const url = new URL(ENDPOINTS.displays.ws, window.location.origin);
        url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';

        // Create connection with auth header
        this.ws = new WebSocket(url.toString(), {
          headers: {
            'Authorization': `Bearer ${token}`
          }
        });
        
        this.ws.onopen = () => {
          this.onConnected();
          resolve();
        };
        
        this.ws.onclose = (event) => {
          // Check if closed due to auth failure
          if (event.code === 4001) { // Custom code for auth failure
            this.auth.clearTokens().catch(err => {
              console.error('Failed to clear invalid tokens:', err);
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
    
    // Update server status using auth interceptor
    const displayId = localStorage.getItem(`display-id-${this.options.displayId}`);
    if (displayId) {
      const interceptor = this.auth.createFetchInterceptor();
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

    // Attempt reconnection if within limits
    if (this.reconnectCount < this.maxReconnectAttempts) {
      const delay = Math.min(
        this.baseReconnectDelay * Math.pow(2, this.reconnectCount),
        30000 // Max 30 second delay
      );
      
      this.reconnectTimer = window.setTimeout(async () => {
        this.reconnectCount++;

        try {
          // Try to get a valid token before connecting
          const token = await this.auth.getValidToken();
          if (!token) {
            throw new Error('No valid auth token available');
          }

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
    if (this.ws) {
      this.ws.close();
    }
    if (this.reconnectTimer) {
      window.clearTimeout(this.reconnectTimer);
    }
    if (this.tokenUnsubscribe) {
      this.tokenUnsubscribe();
    }
    this.events.dispose();
    this.registration.dispose();
    this.auth.dispose();
  }
}
