import type { DisplayConfig } from '../types';
import { ENDPOINTS } from '../constants';
import { AuthService, AuthTokens } from './auth';

const DEFAULT_BACKOFF = {
  MIN_DELAY: 1000,      // 1 second
  MAX_DELAY: 60000,     // 1 minute
  FACTOR: 2,            // Double delay each try
  JITTER: 0.1,         // Add 0-10% random jitter
  MAX_ATTEMPTS: 5,      // Maximum attempts before requiring manual retry
  RATE_LIMIT_DELAY: 5000 // Initial delay after rate limit (5s)
};

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

interface RegistrationError extends Error {
  code?: string;
  isRateLimit?: boolean;
  isAuthError?: boolean;
}

/**
 * Handles display registration flow including device activation and initial authentication.
 * Uses AuthService for token management after successful registration.
 */
export class RegistrationService {
  private readonly auth: AuthService;
  private readonly tokenChangedUnsubscribe: (() => void) | null = null;
  
  // Registration state
  private isRegistered = false;
  private registrationLock: Promise<void> | null = null;
  private registrationAbortController: AbortController | null = null;

  // Rate limiting state
  private lastDeviceCodeRequest = 0;
  private lastRateLimitTime = 0;
  private deviceCodeAttempts = 0;
  private nextDeviceCodeDelay = DEFAULT_BACKOFF.MIN_DELAY;

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
    // Cancel any existing registration attempt
    if (this.registrationAbortController) {
      this.registrationAbortController.abort();
      this.registrationAbortController = null;
    }

    // Create new abort controller
    this.registrationAbortController = new AbortController();

    // Apply rate limiting backoff
    const now = Date.now();

    // Special handling for rate limits
    if (this.lastRateLimitTime > 0) {
      const timeSinceRateLimit = now - this.lastRateLimitTime;
      const rateDelay = Math.max(
        DEFAULT_BACKOFF.RATE_LIMIT_DELAY * Math.pow(DEFAULT_BACKOFF.FACTOR, this.deviceCodeAttempts),
        DEFAULT_BACKOFF.MIN_DELAY
      );
      
      if (timeSinceRateLimit < rateDelay) {
        throw this.createError('Rate limit cooldown in progress', { isRateLimit: true });
      }
    }

    // Regular request backoff
    const timeSinceLastRequest = now - this.lastDeviceCodeRequest;
    if (timeSinceLastRequest < this.nextDeviceCodeDelay) {
      const waitTime = this.nextDeviceCodeDelay - timeSinceLastRequest;
      await new Promise(resolve => setTimeout(resolve, waitTime));
    }

    // Clear existing registration state
    await this.auth.clearTokens();
    localStorage.removeItem(`display-id-${this.config.displayId}`);
    this.isRegistered = false;

    try {
      // Try to get device code with timeout
      const response = await Promise.race([
        fetch(ENDPOINTS.displays.deviceCode, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json'
          },
          body: JSON.stringify({
            displayId: this.config.displayId,
            name: this.config.name,
            location: this.config.location
          }),
          signal: this.registrationAbortController.signal
        }),
        new Promise((_, reject) => 
          setTimeout(() => reject(new Error('Request timeout')), 10000)
        )
      ]) as Response;

      // Handle rate limiting
      if (response.status === 429) {
        this.lastRateLimitTime = now;
        this.deviceCodeAttempts++;
        
        if (this.deviceCodeAttempts >= DEFAULT_BACKOFF.MAX_ATTEMPTS) {
          throw this.createError('Maximum registration attempts exceeded', { 
            isRateLimit: true,
            code: 'MAX_ATTEMPTS'
          });
        }

        throw this.createError('Rate limit exceeded', { isRateLimit: true });
      }

      // Handle other errors
      if (!response.ok) {
        throw this.createError(`Request failed: ${response.statusText}`, {
          code: `HTTP_${response.status}`
        });
      }

      // Successful request
      const result = await response.json();
      
      // Reset backoff on success
      this.deviceCodeAttempts = 0;
      this.nextDeviceCodeDelay = DEFAULT_BACKOFF.MIN_DELAY;
      this.lastDeviceCodeRequest = now;
      this.lastRateLimitTime = 0;

      return result;

    } catch (err) {
      // Enhance error info
      const error = err as RegistrationError;
      
      // Apply backoff for non-rate-limit errors
      if (!error.isRateLimit) {
        this.deviceCodeAttempts++;
        this.nextDeviceCodeDelay = Math.min(
          DEFAULT_BACKOFF.MAX_DELAY,
          DEFAULT_BACKOFF.MIN_DELAY * Math.pow(DEFAULT_BACKOFF.FACTOR, this.deviceCodeAttempts)
        );
        
        // Add jitter
        const jitter = Math.random() * DEFAULT_BACKOFF.JITTER * this.nextDeviceCodeDelay;
        this.nextDeviceCodeDelay += jitter;
      }

      throw error;
    }
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
        throw this.createError('Invalid or expired activation code', {
          code: 'INVALID_CODE',
          isAuthError: true
        });
      } else if (response.status === 429) {
        throw this.createError('Rate limit exceeded', { isRateLimit: true });
      }
      throw this.createError('Activation failed', {
        code: `HTTP_${response.status}`
      });
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
    let currentDelay = interval;

    const controller = new AbortController();
    this.registrationAbortController = controller;

    try {
      while (attempts < maxAttempts && !controller.signal.aborted) {
        try {
          await this.activate(deviceCode);
          return; // Success
        } catch (err) {
          const error = err as RegistrationError;
          
          // Don't retry auth errors
          if (error.isAuthError) {
            throw error;
          }
          
          // Handle rate limits
          if (error.isRateLimit) {
            currentDelay = Math.min(
              DEFAULT_BACKOFF.MAX_DELAY,
              interval * Math.pow(DEFAULT_BACKOFF.FACTOR, attempts)
            );
            // Add jitter
            const jitter = Math.random() * DEFAULT_BACKOFF.JITTER * currentDelay;
            currentDelay += jitter;
          }

          attempts++;
          if (attempts >= maxAttempts) {
            throw this.createError('Activation polling timed out', {
              code: 'POLLING_TIMEOUT'
            });
          }

          // Wait before retry
          await new Promise(resolve => setTimeout(resolve, currentDelay));
        }
      }

      if (controller.signal.aborted) {
        throw this.createError('Activation cancelled', {
          code: 'CANCELLED'
        });
      }

    } finally {
      // Clear controller if it's still current
      if (this.registrationAbortController === controller) {
        this.registrationAbortController = null;
      }
    }
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

    // Only start one registration flow at a time
    if (!this.registrationLock) {
      this.registrationLock = (async () => {
        try {
          const code = await this.startRegistration();
          await this.pollForActivation(code.deviceCode, code.pollInterval);
        } finally {
          this.registrationLock = null;
        }
      })();
    }

    await this.registrationLock;

    // Get fresh token after registration
    const newToken = await this.auth.getValidToken();
    if (!newToken) {
      throw this.createError('Failed to get valid token after registration', {
        code: 'TOKEN_ERROR'
      });
    }

    return newToken;
  }

  /**
   * Helper to create error with consistent format
   */
  private createError(message: string, info: Partial<RegistrationError> = {}): RegistrationError {
    const error = new Error(message) as RegistrationError;
    error.code = info.code;
    error.isRateLimit = info.isRateLimit;
    error.isAuthError = info.isAuthError;
    return error;
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
    if (this.registrationAbortController) {
      this.registrationAbortController.abort();
      this.registrationAbortController = null;
    }
    this.auth.dispose();
  }
}
