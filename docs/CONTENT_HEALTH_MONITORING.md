# Content Health Monitoring Design

## Overview

Wrale Signage monitors content health through direct display feedback rather than independent probing. This design leverages the display's iframe renderer to provide accurate, real-world health data about upstream content sources.

## Key Requirements

- Monitor content availability without interfering with delivery
- Collect real user experience metrics
- Support future extensibility
- Minimize load on upstream systems
- Maintain separation between content delivery and monitoring

## Message Protocol

The iframe communicates with the parent window using a structured message protocol:

```typescript
interface ContentEvent {
  type: ContentEventType;
  contentUrl: string;
  timestamp: number;
  error?: ContentError;
  metrics?: ContentMetrics;
  context?: Record<string, string>;
}

type ContentEventType = 
  | "CONTENT_LOADED"       // Content successfully loaded
  | "CONTENT_ERROR"        // Error loading/rendering content
  | "CONTENT_VISIBLE"      // Content became visible
  | "CONTENT_HIDDEN"       // Content was hidden
  | "CONTENT_INTERACTIVE"; // Content ready for user interaction

interface ContentError {
  code: string;           // Error classification
  message: string;        // Human-readable description
  details?: unknown;      // Additional error context
}

interface ContentMetrics {
  loadTime: number;       // Time to load content
  renderTime: number;     // Time to first render
  interactiveTime?: number; // Time to interactive
  resourceStats?: {
    imageCount: number;   // Number of images loaded
    scriptCount: number;  // Number of scripts loaded
    totalBytes: number;   // Total bytes transferred
  };
}
```

## Implementation Strategy

### Display Component

1. IframeManager handles:
   - Loading content in iframe
   - Monitoring content lifecycle
   - Reporting events to server
   - Managing content transitions

2. EventReporter manages:
   - Buffering events
   - Batch reporting
   - Retry logic
   - Error aggregation

### Server Components

1. ContentHealthService:
   - Collects display feedback
   - Aggregates health metrics
   - Detects content issues
   - Triggers alerts

2. MetricsAggregator:
   - Time-series metrics storage
   - Statistical analysis
   - Trend detection
   - Performance baseline tracking

## Future Extensions

The message protocol supports future enhancements:

1. Content Performance
   - Detailed timing metrics
   - Resource usage stats
   - User interaction tracking
   - Network performance data

2. Content Validation
   - Screenshot comparison
   - Layout verification
   - Accessibility checks
   - Content policy enforcement

3. Display Diagnostics  
   - Browser capabilities
   - Performance limitations
   - Error patterns
   - Resource constraints

## Security Considerations

1. Message Origin Validation
   - Enforce same-origin policy
   - Validate message source
   - Prevent script injection

2. Data Privacy
   - Limit sensitive data collection
   - Apply data retention policies
   - Support data redaction

3. Rate Limiting
   - Prevent event flooding
   - Implement backoff strategies
   - Monitor resource usage

## Success Metrics

1. Reliability
   - Time to detect content issues
   - False positive/negative rates
   - Alert accuracy

2. Performance Impact
   - Additional network usage
   - Client-side overhead
   - Server resource utilization

3. Operational Value
   - Issue detection rate
   - Resolution time improvement
   - Proactive issue prevention