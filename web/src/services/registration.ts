import type { DisplayConfig } from '../types';
import { ENDPOINTS } from '../constants';
import { AuthService, AuthTokens } from './auth';

const DEFAULT_BACKOFF = {
  MIN_DELAY: 1000,      // 1 second
  MAX_DELAY: 60000,     // 1 minute
  FACTOR: 2,            // Double delay each try
  JITTER: 0.1          // Add 0-10% random jitter
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

/**
 * Handles display registration flow including device activation and initial authentication.
 * Uses AuthService for token management after successful registration.
 */
export class RegistrationService {
  private readonly auth: AuthService;
  private readonly tokenChangedUnsubscribe: (() => void) | null = null;
  private isRegistered = false;
  private registrationPromise: Promise<void> | null = null;
  
  // Rate limiting state
  private lastDeviceCodeRequest = 0;
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
    // Check if registration is already in progress
    if (this.registrationPromise) {
      throw new Error('Registration already in progress');
    }

    // Clear any existing registration
    await this.auth.clearTokens();
    localStorage.removeItem(`display-id-${this.config.displayId}`);
    this.isRegistered = false;

    // Apply rate limiting backoff
    const now = Date.now();
    const timeSinceLastRequest = now - this.lastDeviceCodeRequest;
    
    if (timeSinceLastRequest < this.nextDeviceCodeDelay) {
      const waitTime = this.nextDeviceCodeDelay - timeSinceLastRequest;
      await new Promise(resolve => setTimeout(resolve, waitTime));
    }

    try {
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

      // Handle rate limiting
      if (response.status === 429) {
        // Apply exponential backoff
        this.deviceCodeAttempts++;
        this.nextDeviceCodeDelay = Math.min(
          DEFAULT_BACKOFF.MAX_DELAY,
          DEFAULT_BACKOFF.MIN_DELAY * Math.pow(DEFAULT_BACKOFF.FACTOR, this.deviceCodeAttempts)
        );

        // Add jitter
        const jitter = Math.random() * DEFAULT_BACKOFF.JITTER * this.nextDeviceCodeDelay;
        this.nextDeviceCodeDelay += jitter;

        throw new Error('Rate limit exceeded, please try again later');
      }

      if (!response.ok) {
        throw new Error('Failed to start device registration');
      }

      // Successful request, reset backoff
      this.deviceCodeAttempts = 0;
      this.nextDeviceCodeDelay = DEFAULT_BACKOFF.MIN_DELAY;
      this.lastDeviceCodeRequest = Date.now();

      return await response.json();
    } catch (err) {
      // Don't retry rate limit errors immediately
      if (err instanceof Error && err.message.includes('Rate limit exceeded')) {
        throw err;
      }

      // For other errors, apply progressive backoff
      this.deviceCodeAttempts++;
      throw err;
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
        throw new Error('Invalid or expired activation code');
      } else if (response.status === 429) {
        throw new Error('Rate limit exceeded, retrying after backoff');
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
    let currentDelay = interval;

    while (attempts < maxAttempts) {
      try {
        await this.activate(deviceCode);
        return; // Success
      } catch (err) {
        if (err instanceof Error) {
          if (err.message === 'Invalid or expired activation code') {
            throw err; // Don't retry invalid codes
          } else if (err.message.includes('Rate limit exceeded')) {
            // Apply exponential backoff for rate limiting
            currentDelay = Math.min(
              DEFAULT_BACKOFF.MAX_DELAY,
              interval * Math.pow(DEFAULT_BACKOFF.FACTOR, attempts)
            );
            // Add jitter
            const jitter = Math.random() * DEFAULT_BACKOFF.JITTER * currentDelay;
            currentDelay += jitter;
          }
        }

        console.warn('Activation pending, will retry');
        await new Promise(resolve => setTimeout(resolve, currentDelay));
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

    // Only start one registration flow at a time
    if (!this.registrationPromise) {
      this.registrationPromise = (async () => {
        try {
          const code = await this.startRegistration();
          await this.pollForActivation(code.deviceCode, code.pollInterval);
        } finally {
          this.registrationPromise = null;
        }
      })();
    }

    await this.registrationPromise;

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