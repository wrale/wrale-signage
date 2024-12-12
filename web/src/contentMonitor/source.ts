/**
 * Core content monitoring implementation.
 * This code gets injected into content frames.
 */

const monitorSource = `
// Monitor configuration
interface Config {
  contentUrl: string;
  contentType?: 'fixed' | 'video';
  reportInterval?: number;
}

// Resource tracking
interface ResourceStats {
  imageCount: number;
  scriptCount: number;
  totalBytes: number;
  failedResources: string[];
}

class ContentMonitor {
  private startTime: number;
  private resourceStats: ResourceStats = {
    imageCount: 0,
    scriptCount: 0,
    totalBytes: 0,
    failedResources: []
  };
  private hasReportedLoad = false;
  private hasReportedInteractive = false;
  private checkInterval: number | null = null;

  constructor(config?: Config) {
    this.startTime = performance.now();
    this.setupResourceObserver();
    this.setupErrorHandling();
    this.setupPaintObserver();
    this.trackInteractivity();

    // Start periodic checks if configured
    if (config?.reportInterval) {
      this.startChecks(config.reportInterval);
    }
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
              message: 'Failed to load ' + resource.initiatorType + ': ' + resource.name,
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
        this.reportMetrics({
          interactiveTime: performance.now() - this.startTime
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

  private startChecks(interval: number) {
    this.checkInterval = window.setInterval(() => {
      this.reportHealthCheck();
    }, interval);
  }

  private reportHealthCheck() {
    // Get memory info if available
    let memoryInfo;
    if ('memory' in performance) {
      const memory = (performance as any).memory;
      memoryInfo = {
        jsHeapSizeLimit: memory.jsHeapSizeLimit,
        totalJSHeapSize: memory.totalJSHeapSize,
        usedJSHeapSize: memory.usedJSHeapSize
      };
    }

    this.reportEvent('CONTENT_HEALTH', {
      metrics: {
        uptime: performance.now() - this.startTime,
        resourceStats: this.resourceStats,
        memoryInfo
      }
    });
  }

  reportContentLoaded() {
    if (this.hasReportedLoad) return;

    this.hasReportedLoad = true;
    const loadTime = performance.now() - this.startTime;

    this.reportEvent('CONTENT_LOADED', {
      metrics: {
        loadTime,
        renderTime: loadTime,
        resourceStats: this.resourceStats
      }
    });
  }

  private reportError(error: {
    code: string;
    message: string;
    details?: unknown;
  }) {
    this.reportEvent('CONTENT_ERROR', { error });
  }

  private reportMetrics(metrics: {
    renderTime?: number;
    interactiveTime?: number;
  }) {
    this.reportEvent('CONTENT_METRICS', { metrics });
  }

  private reportEvent(type: string, data: Record<string, any> = {}) {
    window.parent.postMessage({
      type: 'CONTENT_EVENT',
      event: {
        type,
        timestamp: Date.now(),
        contentUrl: window.location.href,
        ...data
      }
    }, '*');
  }

  dispose() {
    if (this.checkInterval !== null) {
      window.clearInterval(this.checkInterval);
    }
  }
}

// Initialize monitor when injected
const monitor = new ContentMonitor();

// Report initial load
window.addEventListener('load', () => {
  monitor.reportContentLoaded();
});

// Export for content scripts
(window as any).contentMonitor = monitor;
`;

export default monitorSource;