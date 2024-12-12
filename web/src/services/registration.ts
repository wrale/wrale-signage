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
  private isRegistered = false;

  constructor(
    private readonly config: DisplayConfig,
    private readonly onTokensChanged?: (tokens: AuthTokens | null) => void
  ) {
    // Initialize auth service
    this.auth = new AuthService(config.displayId);

    // Subscribe to token changes if callback provided
    if (onTokensChanged) {
      this.tokenChangedUnsubscribe = this.auth.onTokensChanged(tokens => {
        onTokensChanged(tokens);
        
        // Track registration state
        if (tokens === null) {
          this.isRegistered = false;
        }
      });
    }

    // Check if already registered
    const displayId = localStorage.getItem(`display-id-${config.displayId}`);
    this.isRegistered = displayId !== null && this.auth.getAccessToken() !== null;
  }

  /**
   * Check if display is registered and has valid tokens
   */
  async isValidRegistration(): Promise<boolean> {
    if (!this.isRegistered) {
      return false;
    }

    // Try to get a valid token
    const token = await this.auth.getValidToken();
    return token !== null;
  }

  /**
   * Start registration flow to get device code
   */
  async startRegistration(): Promise<RegistrationResponse> {
    // Clear any existing registration
    await this.auth.clearTokens();
    localStorage.removeItem(`display-id-${this.config.displayId}`);
    this.isRegistered = false;

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
  private async activate(deviceCode: string): Promise<void> {
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
    this.isRegistered = true;
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
   * Get current access token or trigger registration
   */
  async getValidToken(): Promise<string> {
    // Try to get valid token
    const token = await this.auth.getValidToken();
    if (token) {
      return token;
    }

    // No valid token, need to register
    const code = await this.startRegistration();
    await this.pollForActivation(code.deviceCode, code.pollInterval);
    
    // Get fresh token after registration
    const newToken = await this.auth.getValidToken();
    if (!newToken) {
      throw new Error('Failed to get valid token after registration');
    }

    return newToken;
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