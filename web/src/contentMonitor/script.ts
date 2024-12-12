/**
 * Content monitoring injection script.
 * This module exports the monitor code as a string for injection into content frames.
 */

// Import monitor source
import monitorSource from './source';

/**
 * Monitor script ready for injection
 */
export const monitorScript = `
  (function() {
    ${monitorSource}
  })();
`;

// Default export for convenience
export default monitorScript;