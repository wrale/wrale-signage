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
  mediaMetrics?: MediaMetrics;  // New field for media stats
  context?: Record<string, string>;
}

export type ContentEventType = 
  | 'CONTENT_LOADED'       // Content successfully loaded
  | 'CONTENT_ERROR'        // Error loading/rendering content
  | 'CONTENT_VISIBLE'      // Content became visible
  | 'CONTENT_HIDDEN'       // Content was hidden
  | 'CONTENT_INTERACTIVE'  // Content ready for user interaction
  | 'VIDEO_STATS'         // Video playback metrics
  | 'IMAGE_STATS';        // Image display metrics

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
  cpuTime?: number;      // CPU time used
  memoryUsage?: {        // Memory stats if available
    jsHeapSizeLimit: number;
    totalJSHeapSize: number;
    usedJSHeapSize: number;
  };
}

// Media Monitoring Types
export type MediaType = 'video' | 'image';

export interface MediaMetrics {
  type: MediaType;
  video?: VideoMetrics;
  image?: ImageMetrics;
}

export interface VideoMetrics {
  // Playback quality
  droppedFrames: number;
  totalFrames: number;
  fps: number;
  resolution: {
    width: number;
    height: number;
  };
  
  // Buffer health
  bufferSize: number;      // in seconds
  bufferFills: number;     // count of buffer depletions
  stallEvents: number;     // count of playback stalls
  stallDuration: number;   // total ms stalled
  
  // Performance
  decodingTime: number;    // ms to start decode
  playbackTime: number;    // ms to start playing
  cpuTime?: number;        // if available
  memoryUsage?: number;    // if available
}

export interface ImageMetrics {
  // Load performance
  decodeTime: number;      // Time to decode image
  renderTime: number;      // Time to render image
  
  // Image properties
  naturalSize: {
    width: number;
    height: number;
  };
  displaySize: {
    width: number;
    height: number;
  };
  
  // Quality metrics
  isProgressive: boolean;
  compressionRatio?: number;
  optimizationScore?: number;
  scalingRatio: number;    // display size / natural size
}

// Display Status
export interface DisplayStatus {
  displayId: string;
  currentUrl: string;
  currentSequenceVersion: string;
  state: DisplayState;
  lastError?: ContentError;
  lastEvent?: ContentEvent;
  mediaStatus?: {
    videoPlaybackQuality?: Pick<VideoMetrics, 'fps' | 'stallEvents' | 'bufferSize'>;
    imageQuality?: Pick<ImageMetrics, 'scalingRatio' | 'optimizationScore'>;
  };
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
