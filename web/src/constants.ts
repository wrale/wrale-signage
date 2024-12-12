// API endpoints
export const API_VERSION = 'v1alpha1';
export const API_BASE = `/api/${API_VERSION}`;

export const ENDPOINTS = {
  displays: {
    // Device registration flow
    deviceCode: `${API_BASE}/displays/device/code`,
    activate: `${API_BASE}/displays/activate`,
    
    // Display management
    create: `${API_BASE}/displays`,
    get: (id: string) => `${API_BASE}/displays/${id}`,
    activateDisplay: (id: string) => `${API_BASE}/displays/${id}/activate`,
    lastSeen: (id: string) => `${API_BASE}/displays/${id}/last-seen`,
    
    // WebSocket control
    ws: `${API_BASE}/displays/ws`
  }
} as const;
