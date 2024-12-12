/**
 * Content monitoring script that gets injected into content frames.
 * Provides detailed health monitoring and reports back to parent frame.
 */

interface ResourceStats {
  imageCount: number
  scriptCount: number
  totalBytes: number
  failedResources: string[]
}

interface PerformanceMetrics {
  loadTime: number
  renderTime: number
  interactiveTime?: number
  firstPaint?: number
  firstContentfulPaint?: number
  resourceStats: ResourceStats
  memoryInfo?: {
    jsHeapSizeLimit: number
    totalJSHeapSize: number
    usedJSHeapSize: number
  }
}

class ContentMonitor {
  private startTime: number
  private resourceStats: ResourceStats = {
    imageCount: 0,
    scriptCount: 0,
    totalBytes: 0,
    failedResources: []
  }
  private hasReportedLoad = false
  private hasReportedInteractive = false

  constructor() {
    this.startTime = performance.now()
    this.setupResourceObserver()
    this.setupErrorHandling()
    this.setupPaintObserver()
    this.trackInteractivity()
  }

  private setupResourceObserver() {
    // Watch for resource loads
    const observer = new PerformanceObserver((list) => {
      list.getEntries().forEach((entry) => {
        if (entry.entryType === 'resource') {
          const resource = entry as PerformanceResourceTiming

          // Track resource counts
          if (resource.initiatorType === 'img') {
            this.resourceStats.imageCount++
          } else if (resource.initiatorType === 'script') {
            this.resourceStats.scriptCount++
          }

          // Track total bytes if available
          if (resource.transferSize) {
            this.resourceStats.totalBytes += resource.transferSize
          }

          // Check for failed resources
          if (resource.responseStatus >= 400) {
            this.resourceStats.failedResources.push(resource.name)
            this.reportError({
              code: 'RESOURCE_ERROR',
              message: `Failed to load ${resource.initiatorType}: ${resource.name}`,
              details: {
                status: resource.responseStatus,
                type: resource.initiatorType,
                duration: resource.duration
              }
            })
          }
        }
      })
    })

    observer.observe({ entryTypes: ['resource'] })
  }

  private setupPaintObserver() {
    // Watch for paint metrics
    const observer = new PerformanceObserver((list) => {
      const entries = list.getEntries()
      entries.forEach((entry) => {
        const time = entry.startTime + entry.duration

        if (entry.name === 'first-paint') {
          this.reportMetrics({ firstPaint: time })
        } else if (entry.name === 'first-contentful-paint') {
          this.reportMetrics({ firstContentfulPaint: time })
        }
      })
    })

    observer.observe({ entryTypes: ['paint'] })
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
      })
    }, true)

    // Unhandled rejection handler
    window.addEventListener('unhandledrejection', (event) => {
      this.reportError({
        code: 'PROMISE_ERROR',
        message: event.reason?.message || 'Unhandled Promise Rejection',
        details: {
          reason: event.reason,
          stack: event.reason?.stack
        }
      })
    })
  }

  private trackInteractivity() {
    // Track when content becomes interactive
    const interactiveEvents = ['click', 'keydown', 'scroll', 'mousemove']
    
    const onFirstInteraction = () => {
      if (!this.hasReportedInteractive) {
        this.hasReportedInteractive = true
        this.reportMetrics({
          interactiveTime: performance.now() - this.startTime
        })
        
        // Remove listeners after first interaction
        interactiveEvents.forEach(event => {
          window.removeEventListener(event, onFirstInteraction, true)
        })
      }
    }

    interactiveEvents.forEach(event => {
      window.addEventListener(event, onFirstInteraction, { once: true, capture: true })
    })
  }

  reportContentLoaded() {
    if (this.hasReportedLoad) return

    this.hasReportedLoad = true
    const loadTime = performance.now() - this.startTime

    // Get memory info if available
    let memoryInfo
    if ('memory' in performance) {
      const memory = (performance as any).memory
      memoryInfo = {
        jsHeapSizeLimit: memory.jsHeapSizeLimit,
        totalJSHeapSize: memory.totalJSHeapSize,
        usedJSHeapSize: memory.usedJSHeapSize
      }
    }

    const metrics: PerformanceMetrics = {
      loadTime,
      renderTime: loadTime, // Initial estimate, will be updated by paint observers
      resourceStats: this.resourceStats,
      memoryInfo
    }

    this.reportEvent('CONTENT_LOADED', { metrics })
  }

  private reportError(error: {
    code: string
    message: string
    details?: unknown
  }) {
    this.reportEvent('CONTENT_ERROR', { error })
  }

  private reportMetrics(metrics: Partial<PerformanceMetrics>) {
    this.reportEvent('CONTENT_LOADED', { metrics })
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
    }, '*')
  }
}

// Initialize monitor when script loads
const monitor = new ContentMonitor()

// Report initial load
window.addEventListener('load', () => {
  monitor.reportContentLoaded()
})

// Export for use by content scripts
;(window as any).contentMonitor = monitor
