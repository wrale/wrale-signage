/**
 * Core content monitor with media support.
 * Provides detailed content health monitoring and metrics.
 */

import type { 
  ContentEvent, 
  ContentMetrics, 
  ContentError,
  MediaMetrics
} from '../types';
import { MediaMonitor } from './mediaMonitor';

interface MonitorConfig {
  contentUrl: string;
  contentType?: 'fixed' | 'video';
  reportInterval?: number;
}

interface ResourceStats {
  imageCount: number;
  scriptCount: number;
  totalBytes: number;
  failedResources: string[];
}

export class ContentMonitor {
  private startTime: number;
  private resourceStats: ResourceStats = {
    imageCount: 0,
    scriptCount: 0,
    totalBytes: 0,
    failedResources: []
  };
  
  private hasReportedLoad = false;
  private hasReportedInteractive = false;
  private mediaMonitor: MediaMonitor;

  constructor(
    private readonly config: MonitorConfig
  ) {
    this.startTime = performance.now();

    // Initialize monitors
    this.setupResourceObserver();
    this.setupErrorHandling();
    this.setupPaintObserver();
    this.trackInteractivity();

    // Set up media monitoring
    this.mediaMonitor = new MediaMonitor(event => {
      this.reportEvent(
        event.type === 'video' ? 'VIDEO_STATS' : 'IMAGE_STATS',
        {
          mediaMetrics: {
            type: event.type,
            [event.type]: event.metrics
          }
        }
      );

      if (event.error) {
        this.reportError({
          code: event.error.code,
          message: event.error.message,
          details: event.error.details
        });
      }
    });
  }

  dispose() {
    this.mediaMonitor.dispose();
  }

  private setupResourceObserver() {
    const observer = new PerformanceObserver((list) => {
      list.getEntries().forEach((entry) => {
        if (entry.entryType === 'resource') {
          const resource = entry as PerformanceResourceTiming;

          // Track resource counts
          if (resource.initiatorType === 'img') {
            this.resourceStats.imageCount++;
          } else if (resource.initiatorType === 'script') {
            this.resourceStats.scriptCount++;
          }

          // Track total bytes
          if (resource.transferSize) {
            this.resourceStats.totalBytes += resource.transferSize;
          }

          // Track failures
          if (resource.responseStatus >= 400) {
            this.resourceStats.failedResources.push(resource.name);
            this.reportError({
              code: 'RESOURCE_ERROR',
              message: `Failed to load ${resource.initiatorType}: ${resource.name}`,
              details: {
                status: resource.responseStatus,
                type: resource.initiatorType,
                duration: resource.duration
              }
            });
          }
        }
      });
    });

    try {
      observer.observe({ entryTypes: ['resource'] });
    } catch (err) {
      console.warn('Resource timing not supported:', err);
    }
  }

  private setupPaintObserver() {
    const observer = new PerformanceObserver((list) => {
      list.getEntries().forEach((entry) => {
        const time = entry.startTime + entry.duration;

        if (entry.name === 'first-paint') {
          this.reportMetrics({ renderTime: time });
        }
      });
    });

    try {
      observer.observe({ entryTypes: ['paint'] });
    } catch (err) {
      console.warn('Paint timing not supported:', err);
    }
  }

  private setupErrorHandling() {
    // Global error handler
    window.addEventListener('error', (event) => {
      this.reportError({
        code: 'RUNTIME_ERROR',
        message: event.message,
        details: {
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
          stack: event.error?.stack
        }
      });
    }, true);

    // Unhandled rejection handler
    window.addEventListener('unhandledrejection', (event) => {
      this.reportError({
        code: 'PROMISE_ERROR',
        message: event.reason?.message || 'Unhandled Promise Rejection',
        details: {
          reason: event.reason,
          stack: event.reason?.stack
        }
      });
    });
  }

  private trackInteractivity() {
    const interactiveEvents = ['click', 'keydown', 'scroll', 'mousemove'];
    
    const onFirstInteraction = () => {
      if (!this.hasReportedInteractive) {
        this.hasReportedInteractive = true;
        this.reportEvent('CONTENT_INTERACTIVE', {
          metrics: {
            interactiveTime: performance.now() - this.startTime
          }
        });
        
        // Remove listeners after first interaction
        interactiveEvents.forEach(event => {
          window.removeEventListener(event, onFirstInteraction, true);
        });
      }
    };

    interactiveEvents.forEach(event => {
      window.addEventListener(event, onFirstInteraction, { 
        once: true, 
        capture: true 
      });
    });
  }

  reportContentLoaded() {
    if (this.hasReportedLoad) return;

    this.hasReportedLoad = true;
    const loadTime = performance.now() - this.startTime;

    // Get memory info if available
    const memoryUsage = ('performance' in window && 'memory' in performance) 
      ? {
          jsHeapSizeLimit: (performance as any).memory.jsHeapSizeLimit,
          totalJSHeapSize: (performance as any).memory.totalJSHeapSize,
          usedJSHeapSize: (performance as any).memory.usedJSHeapSize
        }
      : undefined;

    const metrics: ContentMetrics = {
      loadTime,
      renderTime: loadTime,
      resourceStats: this.resourceStats,
      memoryUsage
    };

    this.reportEvent('CONTENT_LOADED', { metrics });
  }

  private reportError(error: ContentError) {
    this.reportEvent('CONTENT_ERROR', { error });
  }

  private reportMetrics(metrics: Partial<ContentMetrics>) {
    this.reportEvent('CONTENT_LOADED', { metrics });
  }

  private reportEvent(
    type: ContentEvent['type'],
    data: {
      error?: ContentError;
      metrics?: Partial<ContentMetrics>;
      mediaMetrics?: MediaMetrics;
    } = {}
