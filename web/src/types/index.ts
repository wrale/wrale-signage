export interface ContentSequence {
  items: ContentItem[];
}

export interface ContentItem {
  url: string;
  duration: ContentDuration;
  transition: ContentTransition;
}

export interface ContentDuration {
  type: 'fixed' | 'video';
  value?: number;
}

export interface ContentTransition {
  type: string;
  duration: number;
}

export interface DisplayStatus {
  currentUrl: string;
  lastError?: string;
  updatedAt: string;
}

export type ControlMessageType = 
  | 'SEQUENCE_UPDATE'
  | 'RELOAD'
  | 'STATUS';

export interface ControlMessage {
  type: ControlMessageType;
  timestamp: string;
  sequence?: ContentSequence;
  status?: DisplayStatus;
}
