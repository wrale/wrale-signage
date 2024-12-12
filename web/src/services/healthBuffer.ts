import { ContentEvent, ContentEventType, ContentMetrics } from '../types';

interface BufferConfig {
  maxSize?: number;           // Max events before force flush
  flushInterval?: number;     // Regular flush interval (ms)
  maxErrorAge?: number;       // Max age for error events (ms)
  offlineTimeout?: number;    // How long to wait in offline mode (ms)
}

interface MetricSummary {
  loadTime: {
    min: number;
    max: number;
    avg: number;
    count: number;
  };
  resourceStats: {
    totalImages: number;
    totalScripts: number;
    totalBytes: number;
    failedResources: Set<string>;
  };
}

/**
 * Intelligent buffer for health monitoring events.
 * Groups related events, aggregates metrics, and handles offline periods.
 */
export class HealthEventBuffer {
  private buffer: Map<string, ContentEvent[]> = new Map();
  private errorBuffer: ContentEvent[] = [];
  private metrics: Map<string, MetricSummary> = new Map();
  private lastFlushTime = Date.now();
  private offlineMode = false;
  private flushTimeout?: number;

  constructor(
    private readonly onFlush: (events: ContentEvent[]) => void,
    private readonly config: BufferConfig = {}
  ) {
    // Set defaults
    this.config.maxSize = config.maxSize || 50;
    this.config.flushInterval = config.flushInterval || 5000;
    this.config.maxErrorAge = config.maxErrorAge || 30000;
    this.config.offlineTimeout = config.offlineTimeout || 30000;

    this.startFlushTimer();
  }

  /**
   * Add event to buffer with intelligent handling
   */
  addEvent(event: ContentEvent) {
    const key = this.getEventKey(event);

    // Handle errors immediately
    if (event.type === 'CONTENT_ERROR') {
      this.handleError(event);
      return;
    }

    // Update metrics if present
    if (event.metrics) {
      this.updateMetrics(key, event.metrics);
    }

    // Buffer event by key
    let events = this.buffer.get(key);
    if (!events) {
      events = [];
      this.buffer.set(key, events);
    }
    events.push(event);

    // Check if we need to flush
    if (this.shouldFlush()) {
      this.flush();
    }
  }

  /**
   * Force a buffer flush
   */
  flush() {
    if (this.buffer.size === 0 && this.errorBuffer.length === 0) {
      return;
    }

    try {
      // Prepare events for flush
      const events: ContentEvent[] = [];

      // Add error events first
      events.push(...this.getActiveErrors());

      // Add buffered events with metrics
      for (const [key, contentEvents] of this.buffer) {
        const metrics = this.metrics.get(key);
        if (metrics) {
          // Add summary event
          events.push(this.createSummaryEvent(key, contentEvents, metrics));
        }
        // Add important individual events
        events.push(...this.filterImportantEvents(contentEvents));
      }

      // Send to handler
      if (!this.offlineMode) {
        this.onFlush(events);
        this.lastFlushTime = Date.now();
      }
    } catch (err) {
      console.error('Error flushing health events:', err);
      this.enterOfflineMode();
    } finally {
      // Clear old events but keep recent errors
      this.buffer.clear();
      this.metrics.clear();
      this.pruneErrors();
    }
  }

  /**
   * Handle offline mode and recovery
   */
  private enterOfflineMode() {
    if (!this.offlineMode) {
      console.warn('Entering offline mode for health events');
      this.offlineMode = true;

      // Increase flush interval in offline mode
      if (this.flushTimeout) {
        window.clearInterval(this.flushTimeout);
      }
      this.flushTimeout = window.setInterval(
        () => this.flush(),
        this.config.offlineTimeout
      );
    }
  }

  private exitOfflineMode() {
    if (this.offlineMode) {
      console.info('Exiting offline mode for health events');
      this.offlineMode = false;

      // Restore normal flush interval
      if (this.flushTimeout) {
        window.clearInterval(this.flushTimeout);
      }
      this.startFlushTimer();
    }
  }

  /**
   * Metric management
   */
  private updateMetrics(key: string, metrics: ContentMetrics) {
    let summary = this.metrics.get(key);
    if (!summary) {
      summary = {
        loadTime: { min: Infinity, max: 0, avg: 0, count: 0 },
        resourceStats: {
          totalImages: 0,
          totalScripts: 0,
          totalBytes: 0,
          failedResources: new Set()
        }
      };
      this.metrics.set(key, summary);
    }

    // Update load time stats
    const lt = summary.loadTime;
    lt.min = Math.min(lt.min, metrics.loadTime);
    lt.max = Math.max(lt.max, metrics.loadTime);
    lt.avg = ((lt.avg * lt.count) + metrics.loadTime) / (lt.count + 1);
    lt.count++;

    // Update resource stats
    if (metrics.resourceStats) {
      const rs = summary.resourceStats;
      rs.totalImages += metrics.resourceStats.imageCount;
      rs.totalScripts += metrics.resourceStats.scriptCount;
      rs.totalBytes += metrics.resourceStats.totalBytes;
    }
  }

  /**
   * Error handling
   */
  private handleError(event: ContentEvent) {
    this.errorBuffer.push(event);
    // Always flush on errors
    this.flush();
  }

  private pruneErrors() {
    const now = Date.now();
    this.errorBuffer = this.errorBuffer.filter(event => 
      (now - event.timestamp) <= this.config.maxErrorAge!
    );
  }

  private getActiveErrors(): ContentEvent[] {
    this.pruneErrors();
    return this.errorBuffer;
  }

  /**
   * Utility methods
   */
  private getEventKey(event: ContentEvent): string {
    return `${event.contentUrl}:${event.type}`;
  }

  private shouldFlush(): boolean {
    // Check total event count
    let totalEvents = this.errorBuffer.length;
    for (const events of this.buffer.values()) {
      totalEvents += events.length;
    }
    if (totalEvents >= this.config.maxSize!) {
      return true;
    }

    // Check time since last flush
    return (Date.now() - this.lastFlushTime) >= this.config.flushInterval!;
  }

  private filterImportantEvents(events: ContentEvent[]): ContentEvent[] {
    // Keep visibility changes and interactive events
    return events.filter(event => 
      event.type === 'CONTENT_VISIBLE' ||
      event.type === 'CONTENT_INTERACTIVE'
    );
  }

  private createSummaryEvent(
    key: string,
    events: ContentEvent[],
    metrics: MetricSummary
  ): ContentEvent {
    // Create a summary event for the period
    const lastEvent = events[events.length - 1];
    return {
      type: 'CONTENT_LOADED',
      contentUrl: lastEvent.contentUrl,
      timestamp: Date.now(),
      metrics: {
        loadTime: metrics.loadTime.avg,
        renderTime: metrics.loadTime.avg, // Approximation
        resourceStats: {
          imageCount: metrics.resourceStats.totalImages,
          scriptCount: metrics.resourceStats.totalScripts,
          totalBytes: metrics.resourceStats.totalBytes
        }
      },
      context: {
        eventCount: String(events.length),
        minLoadTime: String(metrics.loadTime.min),
        maxLoadTime: String(metrics.loadTime.max),
        failedResources: Array.from(metrics.resourceStats.failedResources).join(',')
      }
    };
  }

  private startFlushTimer() {
    this.flushTimeout = window.setInterval(
      () => this.flush(),
      this.config.flushInterval
    );
  }

  dispose() {
    if (this.flushTimeout) {
      window.clearInterval(this.flushTimeout);
    }
  }
}
