import type { DisplayConfig } from '../types';
import { ENDPOINTS } from '../constants';

export interface RegistrationResponse {
  deviceCode: string;
  userCode: string;
  expiresIn: number;
  pollInterval: number;
  verificationUri: string;
}

export interface TokenResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  refreshExpiresIn: number;
  tokenType: string;
  displayId: string;
}

/**
 * Handles display registration and authentication
 */
export class RegistrationService {
  private tokens: TokenResponse | null = null;
  private refreshTimer?: number;

  constructor(
    private readonly config: DisplayConfig,
    private readonly onTokensChanged?: (tokens: TokenResponse | null) => void
  ) {
    // Try to restore tokens from storage
    const stored = localStorage.getItem(`display-tokens-${config.displayId}`);
    if (stored) {
      try {
        this.tokens = JSON.parse(stored);
        this.onTokensChanged?.(this.tokens);
        this.scheduleRefresh();
      } catch (err) {
        console.error('Failed to restore tokens:', err);
        localStorage.removeItem(`display-tokens-${config.displayId}`);
      }
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

    const result = await response.json();

    // Store display ID from activation response
    if (result.display?.id) {
      localStorage.setItem(`display-id-${this.config.displayId}`, result.display.id);
    }
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
    return this.tokens?.accessToken ?? null;
  }

  private clearTokens(): void {
    this.tokens = null;
    this.onTokensChanged?.(null);

    localStorage.removeItem(`display-tokens-${this.config.displayId}`);
    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }
  }

  private scheduleRefresh(): void {
    // For now just clear tokens after expiry
    // Token refresh will be implemented in the next phase
    if (!this.tokens?.expiresIn) return;

    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }

    this.refreshTimer = window.setTimeout(
      () => this.clearTokens(),
      this.tokens.expiresIn * 1000
    );
  }

  dispose(): void {
    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }
  }
}
