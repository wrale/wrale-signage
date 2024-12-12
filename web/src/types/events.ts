import type { ContentEvent, ContentSequence, ContentError, RegistrationResponse } from './index';

export interface EventMap {
  sequence: { sequence: ContentSequence };
  error: ContentError;
  close: undefined;
  reconnect: undefined;
  activation: RegistrationResponse;
  health: ContentEvent;
}

export type EventHandler<T> = (data: T) => void;

export interface EventEmitter<T extends Record<string, any>> {
  on<K extends keyof T>(event: K, handler: EventHandler<T[K]>): void;
  off<K extends keyof T>(event: K, handler: EventHandler<T[K]>): void;
  emit<K extends keyof T>(event: K, data: T[K]): void;
  dispose(): void;
}