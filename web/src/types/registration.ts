export type RegistrationState = 
  | 'initializing'
  | 'registering'
  | 'polling'
  | 'registered'
  | 'error';

export type TokenState =
  | 'valid'
  | 'refreshing'
  | 'expired'
  | 'error';

export interface RegistrationError extends Error {
  code?: string;
  isRateLimit?: boolean;
  isAuthError?: boolean;
  retryable?: boolean;
}

export interface RegistrationResponse {
  deviceCode: string;
  userCode: string;
  expiresIn: number;
  pollInterval: number;
  verificationUri: string;
}

export interface RegistrationResult {
  display: {
    id: string;
    name: string;
    [key: string]: unknown;
  };
  auth: {
    accessToken: string;
    refreshToken: string;
    tokenType: string;
    expiresIn: number;
    refreshExpiresIn: number;
  };
}

export interface DisplayLocation {
  siteId: string;
  zone: string;
  position: string;
}