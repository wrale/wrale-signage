import { ENDPOINTS } from '../constants';

export interface AuthTokens {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  refreshExpiresIn: number;
}

interface StoredTokens extends AuthTokens {
  storedAt: number;
}

type TokenUpdateHandler = (tokens: AuthTokens | null) => void;

/**
 * Manages authentication tokens and token refresh for displays.
 */
export class AuthService {
  private tokens: StoredTokens | null = null;
  private refreshTimer?: number;
  private readonly updateHandlers: Set<TokenUpdateHandler> = new Set();
  private refreshPromise: Promise<void> | null = null;

  constructor(
    private readonly displayId: string,
    private readonly storagePrefix: string = 'display-tokens'
  ) {
    this.restoreTokens();
  }

  /**
   * Get current access token
   */
  getAccessToken(): string | null {
    if (!this.tokens) {
      return null;
    }

    // Check if token is expired
    const now = Date.now();
    const expiresAt = this.tokens.storedAt + (this.tokens.expiresIn * 1000);
    
    if (now >= expiresAt) {
      // Token expired, try refresh
      this.refreshTokens().catch(err => {
        console.error('Token refresh failed:', err);
        this.clearTokens();
      });
      return null;
    }

    return this.tokens.accessToken;
  }

  /**
   * Store new auth tokens
   */
  async setTokens(tokens: AuthTokens): Promise<void> {
    const storedTokens: StoredTokens = {
      ...tokens,
      storedAt: Date.now()
    };

    this.tokens = storedTokens;
    await this.persistTokens();
    
    // Schedule refresh
    this.scheduleRefresh();
    
    // Notify handlers
    this.updateHandlers.forEach(handler => {
      try {
        handler(tokens);
      } catch (err) {
        console.error('Error in token update handler:', err);
      }
    });
  }

  /**
   * Clear stored tokens
   */
  async clearTokens(): Promise<void> {
    this.tokens = null;
    localStorage.removeItem(this.getStorageKey());
    
    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }

    // Notify handlers
    this.updateHandlers.forEach(handler => {
      try {
        handler(null);
      } catch (err) {
        console.error('Error in token update handler:', err);
      }
    });
  }

  /**
   * Subscribe to token updates
   */
  onTokensChanged(handler: TokenUpdateHandler): () => void {
    this.updateHandlers.add(handler);
    return () => this.updateHandlers.delete(handler);
  }

  /**
   * Create a fetch interceptor for auth headers
   */
  createFetchInterceptor(): (url: string, init?: RequestInit) => Promise<Response> {
    return async (url: string, init: RequestInit = {}) => {
      // Get current token
      const token = await this.getValidToken();
      
      if (!token) {
        throw new Error('No valid auth token available');
      }

      // Add auth header
      const headers = new Headers(init.headers);
      headers.set('Authorization', `Bearer ${token}`);

      // Make request
      const response = await fetch(url, {
        ...init,
        headers
      });

      // Handle 401 with refresh
      if (response.status === 401) {
        // Try token refresh
        const refreshed = await this.refreshTokens();
        if (refreshed) {
          // Retry with new token
          headers.set('Authorization', `Bearer ${this.tokens!.accessToken}`);
          return fetch(url, {
            ...init,
            headers
          });
        }
      }

      return response;
    };
  }

  private async getValidToken(): Promise<string | null> {
    const token = this.getAccessToken();
    if (token) {
      return token;
    }

    // No token or expired, try refresh
    if (this.tokens?.refreshToken) {
      const success = await this.refreshTokens();
      if (success) {
        return this.tokens!.accessToken;
      }
    }

    return null;
  }

  private async refreshTokens(): Promise<boolean> {
    // Only allow one refresh at a time
    if (this.refreshPromise) {
      await this.refreshPromise;
      return this.tokens !== null;
    }

    if (!this.tokens?.refreshToken) {
      return false;
    }

    this.refreshPromise = (async () => {
      try {
        const response = await fetch(ENDPOINTS.displays.refreshToken, {
          method: 'POST',
          headers: {
            'Authorization': `Bearer ${this.tokens!.refreshToken}`
          }
        });

        if (!response.ok) {
          throw new Error('Token refresh failed');
        }

        const tokens: AuthTokens = await response.json();
        await this.setTokens(tokens);
        
        return true;
      } catch (err) {
        console.error('Token refresh failed:', err);
        await this.clearTokens();
        return false;
      } finally {
        this.refreshPromise = null;
      }
    })();

    return this.refreshPromise;
  }

  private scheduleRefresh(): void {
    if (!this.tokens?.expiresIn) return;

    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }

    // Schedule refresh at 75% of expiry time
    const refreshTime = (this.tokens.expiresIn * 1000) * 0.75;
    
    this.refreshTimer = window.setTimeout(
      () => {
        this.refreshTokens().catch(err => {
          console.error('Scheduled token refresh failed:', err);
          this.clearTokens();
        });
      },
      refreshTime
    );
  }

  private getStorageKey(): string {
    return `${this.storagePrefix}-${this.displayId}`;
  }

  private restoreTokens(): void {
    const stored = localStorage.getItem(this.getStorageKey());
    if (!stored) return;

    try {
      const tokens: StoredTokens = JSON.parse(stored);
      this.tokens = tokens;
      
      // Schedule refresh if token is still valid
      if (Date.now() < tokens.storedAt + (tokens.refreshExpiresIn * 1000)) {
        this.scheduleRefresh();
      } else {
        // Refresh token expired, clear everything
        this.clearTokens();
      }
    } catch (err) {
      console.error('Failed to restore tokens:', err);
      this.clearTokens();
    }
  }

  private async persistTokens(): Promise<void> {
    if (!this.tokens) {
      await this.clearTokens();
      return;
    }

    try {
      localStorage.setItem(
        this.getStorageKey(),
        JSON.stringify(this.tokens)
      );
    } catch (err) {
      console.error('Failed to persist tokens:', err);
    }
  }

  dispose(): void {
    if (this.refreshTimer) {
      window.clearTimeout(this.refreshTimer);
    }
    this.updateHandlers.clear();
  }
}
