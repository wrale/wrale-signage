/**
 * Content monitoring injection script.
 * This module exports the monitor code as a string for injection into content frames.
 */

// Import monitor source
import monitorSource from './source';

/**
 * Get content monitor script as a string for injection
 */
export function getMonitorScript(): string {
  // Remove import/export statements and wrap in IIFE
  const script = `
    (function() {
      ${monitorSource}
    })();
  `;

  return script;
}

/**
 * Export monitor script as default for direct use in frame injection
 */
export default getMonitorScript();