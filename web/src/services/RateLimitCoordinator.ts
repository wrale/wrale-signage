/**
 * Global rate limit coordinator that manages backoff delays across components
 */

export interface RateLimitState {
  lastAttempt: number;             // Timestamp of last attempt
  lastRateLimitTime: number;       // Timestamp of last 429 response
  nextDelay: number;               // Current backoff delay
  attemptCount: number;            // Number of attempts made
  isRateLimited: boolean;          // Currently in rate limit state
}

export interface BackoffConfig {
  minDelay: number;                // Minimum delay between attempts
  maxDelay: number;                // Maximum delay between attempts
  factor: number;                  // Multiplier for exponential backoff
  jitter: number;                  // Random jitter factor (0-1)
  rateLimitDelay: number;         // Initial delay after rate limit
  maxAttempts: number;            // Maximum attempts before requiring manual retry
}

const DEFAULT_BACKOFF: BackoffConfig = {
  minDelay: 1000,        // 1 second
  maxDelay: 60000,       // 1 minute
  factor: 2,             // Double delay each try
  jitter: 0.1,           // Add 0-10% random jitter
  rateLimitDelay: 5000,  // 5s initial delay after rate limit
  maxAttempts: 5         // Max attempts before manual retry
};

/**
 * Singleton service that coordinates rate limiting across components
 */
export class RateLimitCoordinator {
  private static instance: RateLimitCoordinator;
  private endpointStates = new Map<string, RateLimitState>();
  private apiGroups = new Map<string, Set<string>>();

  private constructor() {
    // Private constructor for singleton
  }

  /**
   * Get singleton instance
   */
  static getInstance(): RateLimitCoordinator {
    if (!RateLimitCoordinator.instance) {
      RateLimitCoordinator.instance = new RateLimitCoordinator();
    }
    return RateLimitCoordinator.instance;
  }

  /**
   * Group related endpoints that should share rate limit state
   */
  addEndpointGroup(groupKey: string, endpoints: string[]): void {
    const endpointSet = new Set(endpoints);
    this.apiGroups.set(groupKey, endpointSet);
    
    // Initialize state for each endpoint
    endpoints.forEach(endpoint => {
      if (!this.endpointStates.has(endpoint)) {
        this.endpointStates.set(endpoint, this.createInitialState());
      }
    });
  }

  /**
   * Check if operation should be delayed due to rate limiting
   */
  async shouldDelay(
    endpoint: string,
    config: Partial<BackoffConfig> = {}
  ): Promise<boolean> {
    const state = this.getEndpointState(endpoint);
    if (!state) return false;

    const backoff = { ...DEFAULT_BACKOFF, ...config };
    const now = Date.now();

    // Check if in rate limit cooldown
    if (state.isRateLimited) {
      const timeSinceRateLimit = now - state.lastRateLimitTime;
      const rateDelay = Math.max(
        backoff.rateLimitDelay * Math.pow(backoff.factor, state.attemptCount),
        backoff.minDelay
      );
      
      if (timeSinceRateLimit < rateDelay) {
        // Still in cooldown, delay needed
        return true;
      }
      // Cooldown complete
      state.isRateLimited = false;
    }

    // Check regular backoff delay
    const timeSinceLastAttempt = now - state.lastAttempt;
    return timeSinceLastAttempt < state.nextDelay;
  }

  /**
   * Get current delay time for endpoint
   */
  getDelay(
    endpoint: string,
    config: Partial<BackoffConfig> = {}
  ): number {
    const state = this.getEndpointState(endpoint);
    if (!state) return 0;

    const backoff = { ...DEFAULT_BACKOFF, ...config };
    const now = Date.now();

    // Calculate remaining rate limit delay
    if (state.isRateLimited) {
      const timeSinceRateLimit = now - state.lastRateLimitTime;
      const rateDelay = Math.max(
        backoff.rateLimitDelay * Math.pow(backoff.factor, state.attemptCount),
        backoff.minDelay
      );
      
      if (timeSinceRateLimit < rateDelay) {
        return rateDelay - timeSinceRateLimit;
      }
    }

    // Calculate remaining regular backoff
    const timeSinceLastAttempt = now - state.lastAttempt;
    if (timeSinceLastAttempt < state.nextDelay) {
      return state.nextDelay - timeSinceLastAttempt;
    }

    return 0;
  }

  /**
   * Record a successful API call
   */
  recordSuccess(endpoint: string): void {
    const state = this.getEndpointState(endpoint);
    if (!state) return;

    // Reset state on success
    state.attemptCount = 0;
    state.nextDelay = DEFAULT_BACKOFF.minDelay;
    state.isRateLimited = false;
    state.lastAttempt = Date.now();
  }

  /**
   * Record a rate limited response
   */
  recordRateLimit(
    endpoint: string,
    config: Partial<BackoffConfig> = {}
  ): void {
    const state = this.getEndpointState(endpoint);
    if (!state) return;

    const backoff = { ...DEFAULT_BACKOFF, ...config };
    const now = Date.now();

    state.lastRateLimitTime = now;
    state.lastAttempt = now;
    state.isRateLimited = true;
    state.attemptCount++;

    // Update delay for rate limit
    state.nextDelay = Math.min(
      backoff.maxDelay,
      backoff.rateLimitDelay * Math.pow(backoff.factor, state.attemptCount)
    );

    // Add jitter
    const jitter = Math.random() * backoff.jitter * state.nextDelay;
    state.nextDelay += jitter;
  }

  /**
   * Record a regular error response
   */
  recordError(
    endpoint: string,
    config: Partial<BackoffConfig> = {}
  ): void {
    const state = this.getEndpointState(endpoint);
    if (!state) return;

    const backoff = { ...DEFAULT_BACKOFF, ...config };
    
    state.lastAttempt = Date.now();
    state.attemptCount++;

    // Update regular backoff delay
    state.nextDelay = Math.min(
      backoff.maxDelay,
      backoff.minDelay * Math.pow(backoff.factor, state.attemptCount)
    );

    // Add jitter
    const jitter = Math.random() * backoff.jitter * state.nextDelay;
    state.nextDelay += jitter;
  }

  /**
   * Record start of request attempt
   */
  recordAttempt(endpoint: string): void {
    const state = this.getEndpointState(endpoint);
    if (state) {
      state.lastAttempt = Date.now();
    }
  }

  /**
   * Reset state for endpoint
   */
  reset(endpoint: string): void {
    const state = this.getEndpointState(endpoint);
    if (state) {
      Object.assign(state, this.createInitialState());
    }
  }

  /**
   * Helper to get state, checking groups
   */
  private getEndpointState(endpoint: string): RateLimitState | null {
    // Check endpoint directly
    let state = this.endpointStates.get(endpoint);
    if (state) return state;

    // Check if endpoint is part of any group
    for (const [, endpoints] of this.apiGroups) {
      if (endpoints.has(endpoint)) {
        // Get first existing state in group or create new
        for (const groupEndpoint of endpoints) {
          state = this.endpointStates.get(groupEndpoint);
          if (state) return state;
        }
        // No state found in group, create new
        state = this.createInitialState();
        this.endpointStates.set(endpoint, state);
        return state;
      }
    }

    return null;
  }

  private createInitialState(): RateLimitState {
    return {
      lastAttempt: 0,
      lastRateLimitTime: 0,
      nextDelay: DEFAULT_BACKOFF.minDelay,
      attemptCount: 0,
      isRateLimited: false
    };
  }
}

// Create default groups
const coordinator = RateLimitCoordinator.getInstance();

coordinator.addEndpointGroup('registration', [
  '/api/v1alpha1/displays/device/code',
  '/api/v1alpha1/displays/activate'
]);

export default coordinator;