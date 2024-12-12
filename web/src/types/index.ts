// Content Types
export interface ContentSequence {
  items: ContentItem[];
  version: string;
  updateInterval: number;
  fallback: FallbackConfig;
}

export interface ContentItem {
  url: string;
  duration: ContentDuration;
  transition: TransitionConfig;
  reload: boolean;
  prefetch: boolean;
}

export interface ContentDuration {
  type: 'fixed' | 'video';
  value?: number;  // in milliseconds for 'fixed'
}

export interface TransitionConfig {
  type: 'fade' | 'slide' | 'none';
  duration: number;  // in milliseconds
  direction?: 'left' | 'right' | 'up' | 'down';
}

export interface FallbackConfig {
  url: string;
  reloadInterval: number;  // in milliseconds
}

// Health Monitoring Types
export interface ContentEvent {
  type: ContentEventType;
  contentUrl: string;
  timestamp: number;
  error?: ContentError;
  metrics?: ContentMetrics;
  context?: Record<string, string>;
}

export type ContentEventType = 
  | 'CONTENT_LOADED'       // Content successfully loaded
  | 'CONTENT_ERROR'        // Error loading/rendering content
  | 'CONTENT_VISIBLE'      // Content became visible
  | 'CONTENT_HIDDEN'       // Content was hidden
  | 'CONTENT_INTERACTIVE'; // Content ready for user interaction

export interface ContentError {
  code: string;           // Error classification
  message: string;        // Human-readable description
  details?: unknown;      // Additional error context
}

export interface ContentMetrics {
  loadTime: number;       // Time to load content
  renderTime: number;     // Time to first render
  interactiveTime?: number; // Time to interactive
  resourceStats?: {
    imageCount: number;   // Number of images loaded
    scriptCount: number;  // Number of scripts loaded
    totalBytes: number;   // Total bytes transferred
  };
}

// Display Status
export interface DisplayStatus {
  displayId: string;
  currentUrl: string;
  currentSequenceVersion: string;
  state: DisplayState;
  lastError?: ContentError;
  lastEvent?: ContentEvent;
  updatedAt: string;
}

export type DisplayState = 
  | 'LOADING'      // Initial state
  | 'ACTIVE'       // Normal operation
  | 'ERROR'        // Error state
  | 'RECOVERY'     // Attempting recovery
  | 'OFFLINE';     // Connection lost

// WebSocket Control Protocol
export type ControlMessageType = 
  | 'SEQUENCE_UPDATE'    // New content sequence
  | 'DISPLAY_CONFIG'     // Display configuration update
  | 'RELOAD'            // Force content reload
  | 'STATUS_REQUEST'     // Request status update
  | 'STATUS_RESPONSE'    // Status response
  | 'HEALTH_EVENT'       // Health monitoring event
  | 'ERROR'             // Error notification
  | 'RECOVERY';         // Recovery instruction

export interface ControlMessage {
  type: ControlMessageType;
  timestamp: string;
  messageId: string;
  sequence?: ContentSequence;
  status?: DisplayStatus;
  event?: ContentEvent;
  error?: ContentError;
  config?: DisplayConfig;
}

export interface DisplayConfig {
  displayId: string;
  name: string;
  location: {
    siteId: string;
    zone: string;
    position: string;
  };
  settings: {
    logLevel: 'debug' | 'info' | 'warn' | 'error';
    healthCheckInterval: number;
    reconnectInterval: number;
    maxRetryCount: number;
  };
}
