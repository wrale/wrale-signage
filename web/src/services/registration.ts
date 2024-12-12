import type { DisplayConfig } from '../types';
import { ENDPOINTS } from '../constants';
import { AuthService, AuthTokens } from './auth';

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
  auth: AuthTokens;
}

/**
 * Handles display registration flow including device activation and initial authentication.
 * Uses AuthService for token management after successful registration.
 */
export class RegistrationService {
  private readonly auth: AuthService;
  private readonly tokenChangedUnsubscribe: (() => void) | null = null;

  constructor(
    private readonly config: DisplayConfig,
    private readonly onTokensChanged?: (tokens: AuthTokens | null) => void
  ) {
    this.auth = new AuthService(config.displayId);

    if (onTokensChanged) {
      this.tokenChangedUnsubscribe = this.auth.onTokensChanged(onTokensChanged);
    }
  }

  /**
   * Start registration flow to get device code
   */
  async startRegistration(): Promise<RegistrationResponse> {
    const response = await fetch(ENDPOINTS.displays.deviceCode, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        displayId: this.config.displayId,
        name: this.config.name,
        location: this.config.location
      })
    });

    if (!response.ok) {
      throw new Error('Failed to start device registration');
    }

    return await response.json();
  }

  /**
   * Activate display using registration info
   */
  async activate(deviceCode: string): Promise<void> {
    const response = await fetch(ENDPOINTS.displays.activate, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        activationCode: deviceCode,
        name: this.config.name,
        location: this.config.location
      })
    });

    if (!response.ok) {
      if (response.status === 404) {
        throw new Error('Invalid or expired activation code');
      }
      throw new Error('Activation failed');
    }

    const result: RegistrationResult = await response.json();

    // Store registered display ID
    localStorage.setItem(`display-id-${this.config.displayId}`, result.display.id);

    // Store auth tokens
    await this.auth.setTokens(result.auth);
  }

  /**
   * Poll for successful activation
   */
  async pollForActivation(deviceCode: string, interval: number): Promise<void> {
    let attempts = 0;
    const maxAttempts = 180; // 15 minutes at 5-second intervals

    while (attempts < maxAttempts) {
      try {
        await this.activate(deviceCode);
        return; // Success
      } catch (err) {
        if (err instanceof Error && err.message === 'Invalid or expired activation code') {
          throw err; // Don't retry invalid codes
        }
        console.warn('Activation pending, will retry');
        await new Promise(resolve => setTimeout(resolve, interval * 1000));
        attempts++;
      }
    }

    throw new Error('Activation timed out');
  }

  /**
   * Get current access token
   */
  getAccessToken(): string | null {
    return this.auth.getAccessToken();
  }

  /**
   * Create fetch interceptor for auth headers
   */
  createFetchInterceptor(): (url: string, init?: RequestInit) => Promise<Response> {
    return this.auth.createFetchInterceptor();
  }

  dispose(): void {
    if (this.tokenChangedUnsubscribe) {
      this.tokenChangedUnsubscribe();
    }
    this.auth.dispose();
  }
}