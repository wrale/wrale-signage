import type { DisplayConfig } from '../types';

export interface RegistrationResponse {
  deviceCode: string;
  userCode: string;
  verificationUri: string;
  verificationUriComplete: string;
  expiresIn: number;
  interval: number;
}

export interface TokenResponse {
  accessToken: string;
  refreshToken: string;
  expiresIn: number;
  refreshExpiresIn: number;
  tokenType: string;
  deviceGuid: string;
}

/**
 * Handles display registration and authentication using OAuth 2.0
 * Device Authorization Flow (RFC 8628)
 */
export class RegistrationService {
  private refreshTimer?: number;
  private tokens: TokenResponse | null = null;

  constructor(
    private readonly config: DisplayConfig,
    private readonly onTokensChanged?: (tokens: TokenResponse | null) => void
  ) {
    // Restore tokens from localStorage if available
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
   * Start registration flow to get device and user codes
   */
  async startRegistration(): Promise<RegistrationResponse> {
    const response = await fetch('/api/v1alpha1/auth/device/code', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      body: new URLSearchParams({
        client_id: 'Wrale Signage_CLIENT',
        scope: 'display'
      })
    });

    if (!response.ok) {
      throw new Error('Failed to start device registration');
    }

    return await response.json();
  }

  /**
   * Poll for token after user enters code
   */
  async pollForToken(deviceCode: string, interval: number): Promise<TokenResponse> {
    while (true) {
      try {
        const response = await fetch('/api/v1alpha1/auth/device/token', {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded'
          },
          body: new URLSearchParams({
            grant_type: 'urn:ietf:params:oauth:grant-type:device_code',
            device_code: deviceCode,
            client_id: 'Wrale Signage_CLIENT'
          })
        });

        if (response.status === 428) {
          // Not yet authorized, wait and retry
          await new Promise(resolve => setTimeout(resolve, interval * 1000));
          continue;
        }

        if (!response.ok) {
          throw new Error('Token request failed');
        }

        const tokens = await response.json();
        this.setTokens(tokens);
        return tokens;

      } catch (err) {
        console.error('Error polling for token:', err);
        await new Promise(resolve => setTimeout(resolve, interval * 1000));
      }
    }
  }

  /**
   * Get current access token, refreshing if needed
   */
  async getAccessToken(): Promise<string | null> {
    if (!this.tokens) {
      return null;
    }

    // Check if token needs refresh
    const expiryThreshold = 60; // Refresh 60 seconds before expiry
    const now = Math.floor(Date.now() / 1000);
    const tokenExpiry = now + this.tokens.expiresIn;

    if (tokenExpiry - now <= expiryThreshold) {
      await this.refreshTokens();
    }

    return this.tokens?.accessToken ?? null;
  }

  /**
   * Refresh tokens using refresh token
   */
  private async refreshTokens(): Promise<void> {
    if (!this.tokens?.refreshToken) {
      return;
    }

    try {
      const response = await fetch('/api/v1alpha1/auth/token', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: new URLSearchParams({
          grant_type: 'refresh_token',
          refresh_token: this.tokens.refreshToken,
          client_id: 'Wrale Signage_CLIENT'
        })
      });

      if (!response.ok) {
        throw new Error('Token refresh failed');
      }

      const newTokens = await response.json();
      this.setTokens(newTokens);

    } catch (err) {
      console.error('Error refreshing tokens:', err);
      this.clearTokens();
    }
  }

  private setTokens(tokens: TokenResponse | null): void {
    this.tokens = tokens;
    this.onTokensChanged?.(tokens);

    if (tokens) {
      localStorage.setItem(
        `display-tokens-${this.config.displayId}`,
        JSON.stringify(tokens)
      );
      this.scheduleRefresh();
    } else {
      localStorage.removeItem(`display-tokens-${this.config.displayId}`);
      if (this.refreshTimer) {
        window.clearTimeout(this.refreshTimer);
      }
    }
  }

  private clearTokens(): void {
    this.setTokens(null);
  }

  private scheduleRefresh(): void {
    if (!this.tokens) return;

    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }

    // Schedule refresh for 1 minute before expiry
    const refreshIn = (this.tokens.expiresIn - 60) * 1000;
    this.refreshTimer = window.setTimeout(
      () => this.refreshTokens(),
      refreshIn
    );
  }

  dispose(): void {
    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }
  }
}
