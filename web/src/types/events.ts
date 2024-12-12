import { DisplayError } from './display';

export interface ControlMessage {
  type: MessageType;
  messageId: string;
  timestamp: string;
  sequence?: ContentSequence;
  status?: ContentStatus;
  error?: ContentError;
}

export type MessageType =
  | 'SEQUENCE_UPDATE'
  | 'STATUS_REQUEST'
  | 'STATUS_RESPONSE'
  | 'ERROR';

export interface ContentSequence {
  id: string;
  version: string;
  items: ContentItem[];
}

export interface ContentItem {
  url: string;
  type: ContentType;
  duration: number | { value: number };
  transition: TransitionConfig;
  options?: ContentOptions;
}

export type ContentType =
  | 'static'
  | 'video'
  | 'stream'
  | 'web';

export interface TransitionConfig {
  type: TransitionType;
  duration: number;
  direction?: TransitionDirection;
}

export type TransitionType =
  | 'none'
  | 'fade'
  | 'slide'
  | 'zoom';

export type TransitionDirection =
  | 'left'
  | 'right'
  | 'up'
  | 'down';

export interface ContentOptions {
  preload?: boolean;
  reload?: boolean;
  interactive?: boolean;
  volume?: number;
  [key: string]: unknown;
}

export interface ContentStatus {
  url: string;
  type: ContentType;
  state: ContentState;
  error?: ContentError;
  metrics?: ContentMetrics;
}

export type ContentState =
  | 'loading'
  | 'playing'
  | 'paused'
  | 'error';

export interface ContentEvent {
  type: ContentEventType;
  contentUrl: string;
  timestamp: number;
  error?: ContentError;
  metrics?: ContentMetrics;
  context?: Record<string, string>;
}

export type ContentEventType =
  | 'CONTENT_LOADED'
  | 'CONTENT_ERROR'  
  | 'CONTENT_VISIBLE'
  | 'CONTENT_HIDDEN'
  | 'CONTENT_INTERACTIVE';

export interface ContentError {
  code: string;
  message: string;
  details?: unknown;
  retryable?: boolean;
}

export interface ContentMetrics {
  loadTime: number;
  renderTime?: number;
  interactiveTime?: number;
  resourceStats?: {
    imageCount: number;
    scriptCount: number;
    totalBytes: number;
    failedResources?: string[];
  };
}

export interface EventMap {
  sequence: ControlMessage;
  status: ControlMessage;
  health: ContentEvent;
  error: DisplayError;
  reconnect: void;
  close: void;
}

export type EventHandler<T> = (event: T) => void;