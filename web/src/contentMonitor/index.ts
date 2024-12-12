/**
 * Content monitoring entry point.
 * Exports monitor script and types for display integration.
 */

export { default as monitorScript } from './script';
export type { ContentEvent, ContentMetrics, ContentError } from '../types';
export { MediaMonitor } from './mediaMonitor';