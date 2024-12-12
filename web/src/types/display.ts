import { DisplayLocation } from './registration';

export interface DisplayConfig {
  displayId: string;
  name: string;
  location: DisplayLocation;
  settings: {
    logLevel: string;
    healthCheckInterval: number;
    reconnectInterval: number;
    maxRetryCount: number;
  };
}

export interface DisplayStatus {
  displayId: string;
  currentUrl: string;
  currentSequenceVersion: string;
  state: DisplayState;
  lastError?: DisplayError;
  updatedAt: string;
}

export type DisplayState = 
  | 'LOADING'
  | 'ACTIVE'
  | 'ERROR';

export interface DisplayError {
  code: string;
  message: string;
  details?: unknown;
}