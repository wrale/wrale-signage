/**
 * Media performance monitoring for video and image content.
 */

interface VideoMetrics {
  // Playback quality
  droppedFrames: number
  totalFrames: number
  fps: number
  resolution: {
    width: number
    height: number
  }
  
  // Buffer health
  bufferSize: number // in seconds
  bufferFills: number
  stallEvents: number
  stallDuration: number // total ms stalled
  
  // Performance
  decodingTime: number // ms to start decode
  playbackTime: number // ms to start playing
  cpuTime?: number     // if available
  memoryUsage?: number // if available
}

interface ImageMetrics {
  // Load performance
  decodeTime: number
  renderTime: number
  
  // Image properties
  naturalSize: {
    width: number
    height: number
  }
  displaySize: {
    width: number
    height: number
  }
  
  // Quality
  isProgressive: boolean
  compressionRatio?: number
  optimizationScore?: number
  scalingRatio: number
}

type MediaEventCallback = (event: MediaEvent) => void;

interface MediaEvent {
  type: 'video' | 'image'
  url: string
  timestamp: number
  metrics: VideoMetrics | ImageMetrics
  error?: {
    code: string
    message: string
    details?: unknown
  }
}

interface VideoState {
  startTime: number
  metrics: VideoMetrics
  lastStallTime: number
  checkInterval?: number
}

interface ImageState {
  startTime: number
  metrics: ImageMetrics
  reported: boolean
}

export class MediaMonitor {
  private videoElements: Map<HTMLVideoElement, VideoState> = new Map()
  private imageElements: Map<HTMLImageElement, ImageState> = new Map()
  private observer: MutationObserver
  private onEvent: MediaEventCallback

  constructor(onEvent: MediaEventCallback) {
    this.onEvent = onEvent
    
    // Watch for media elements
    this.observer = new MutationObserver(mutations => {
      mutations.forEach(mutation => {
        mutation.addedNodes.forEach(node => {
          if (node instanceof HTMLVideoElement) {
            this.watchVideo(node)
          } else if (node instanceof HTMLImageElement) {
            this.watchImage(node)
          }
        })

        // Check removed nodes
        mutation.removedNodes.forEach(node => {
          if (node instanceof HTMLVideoElement) {
            this.unwatchVideo(node)
          }
        })
      })
    })
    
    // Start observing
    this.observer.observe(document.body, {
      childList: true,
      subtree: true
    })
    
    // Watch existing media
    document.querySelectorAll('video').forEach(video => this.watchVideo(video))
    document.querySelectorAll('img').forEach(img => this.watchImage(img))
  }

  dispose() {
    this.observer.disconnect()
    
    // Clean up video monitoring
    this.videoElements.forEach((state, video) => {
      this.unwatchVideo(video)
    })
    this.videoElements.clear()
    this.imageElements.clear()
  }

  private watchVideo(video: HTMLVideoElement) {
    if (this.videoElements.has(video)) return

    const state: VideoState = {
      startTime: performance.now(),
      metrics: {
        droppedFrames: 0,
        totalFrames: 0,
        fps: 0,
        resolution: {
          width: video.videoWidth,
          height: video.videoHeight
        },
        bufferSize: 0,
        bufferFills: 0,
        stallEvents: 0,
        stallDuration: 0,
        decodingTime: 0,
        playbackTime: 0
      },
      lastStallTime: 0
    }

    this.videoElements.set(video, state)

    // Track load timing
    const loadStart = performance.now()
    video.addEventListener('loadeddata', () => {
      state.metrics.decodingTime = performance.now() - loadStart
      state.metrics.resolution = {
        width: video.videoWidth,
        height: video.videoHeight
      }
    })

    video.addEventListener('playing', () => {
      if (!state.metrics.playbackTime) {
        state.metrics.playbackTime = performance.now() - state.startTime
      }
      if (state.lastStallTime) {
        state.metrics.stallDuration += performance.now() - state.lastStallTime
        state.lastStallTime = 0
      }
    })

    // Track stalls
    video.addEventListener('waiting', () => {
      if (!state.lastStallTime) {
        state.lastStallTime = performance.now()
        state.metrics.stallEvents++
      }
    })

    // Track errors
    video.addEventListener('error', () => {
      this.onEvent({
        type: 'video',
        url: video.src,
        timestamp: Date.now(),
        metrics: state.metrics,
        error: {
          code: 'VIDEO_ERROR',
          message: `Video error: ${video.error?.message || 'unknown error'}`,
          details: {
            errorCode: video.error?.code,
            message: video.error?.message,
            time: performance.now() - state.startTime
          }
        }
      })
    })

    // Start periodic checks
    state.checkInterval = window.setInterval(() => {
      this.checkVideo(video, state)
    }, 1000) // Check every second
  }

  private unwatchVideo(video: HTMLVideoElement) {
    const state = this.videoElements.get(video)
    if (state?.checkInterval) {
      window.clearInterval(state.checkInterval)
    }
    this.videoElements.delete(video)
  }

  private checkVideo(video: HTMLVideoElement, state: VideoState) {
    // Update buffer stats
    if (video.buffered.length > 0) {
      const buffered = video.buffered.end(video.buffered.length - 1) - video.currentTime
      state.metrics.bufferSize = buffered
      
      if (buffered < 0.5) { // Less than 500ms buffer
        state.metrics.bufferFills++
      }
    }

    // Get playback quality stats if available
    const playbackStats = (video as any).getVideoPlaybackQuality?.()
    if (playbackStats) {
      state.metrics.droppedFrames = playbackStats.droppedVideoFrames
      state.metrics.totalFrames = playbackStats.totalVideoFrames
      
      if (state.metrics.totalFrames > 0) {
        const elapsed = (performance.now() - state.startTime) / 1000
        state.metrics.fps = Math.round((state.metrics.totalFrames / elapsed) * 10) / 10
      }
    }

    // Try to get performance stats
    if ('performance' in window) {
      const perf = performance as any
      if (perf.memory) {
        state.metrics.memoryUsage = perf.memory.usedJSHeapSize
      }
    }

    // Report current metrics
    this.onEvent({
      type: 'video',
      url: video.src,
      timestamp: Date.now(),
      metrics: state.metrics
    })
  }

  private watchImage(img: HTMLImageElement) {
    if (this.imageElements.has(img)) return

    const state: ImageState = {
      startTime: performance.now(),
      metrics: {
        decodeTime: 0,
        renderTime: 0,
        naturalSize: {
          width: 0,
          height: 0
        },
        displaySize: {
          width: 0,
          height: 0
        },
        isProgressive: false,
        scalingRatio: 1
      },
      reported: false
    }

    this.imageElements.set(img, state)

    // Track load timing
    const loadStart = performance.now()

    const reportImageMetrics = () => {
      if (state.reported) return
      state.reported = true

      state.metrics.decodeTime = performance.now() - loadStart
      state.metrics.renderTime = performance.now() - state.startTime

      // Get dimensions
      state.metrics.naturalSize = {
        width: img.naturalWidth,
        height: img.naturalHeight
      }
      state.metrics.displaySize = {
        width: img.width || img.clientWidth,
        height: img.height || img.clientHeight
      }

      // Calculate scaling
      const naturalArea = img.naturalWidth * img.naturalHeight
      const displayArea = (img.width || img.clientWidth) * (img.height || img.clientHeight)
      state.metrics.scalingRatio = Math.round((displayArea / naturalArea) * 100) / 100

      // Check progressive loading
      try {
        const canvas = document.createElement('canvas')
        canvas.width = 2
        canvas.height = 2
        const ctx = canvas.getContext('2d')
        if (ctx) {
          ctx.drawImage(img, 0, 0, 2, 2)
          const data = ctx.getImageData(0, 0, 1, 1)
          state.metrics.isProgressive = data.data[3] !== 255
        }
      } catch (err) {
        // Canvas operations may fail in some contexts
        state.metrics.isProgressive = false
      }

      // Report metrics
      this.onEvent({
        type: 'image',
        url: img.src,
        timestamp: Date.now(),
        metrics: state.metrics
      })
    }

    // Track successful load
    img.addEventListener('load', reportImageMetrics)

    // Track errors
    img.addEventListener('error', () => {
      this.onEvent({
        type: 'image',
        url: img.src,
        timestamp: Date.now(),
        metrics: state.metrics,
        error: {
          code: 'IMAGE_ERROR',
          message: 'Failed to load image',
          details: {
            src: img.src,
            time: performance.now() - state.startTime,
            naturalWidth: img.naturalWidth,
            naturalHeight: img.naturalHeight
          }
        }
      })
    })
  }
}
