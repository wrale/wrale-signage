import React from 'react';
import { ContentController } from './components/ContentController';
import type { DisplayConfig } from './types';

function App() {
  const params = new URLSearchParams(window.location.search);
  const displayId = params.get('id') || 'unregistered';
  const wsURL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1alpha1/displays/ws`;

  // Initial display configuration
  const config: DisplayConfig = {
    displayId,
    name: displayId,
    location: {
      siteId: params.get('site') || 'unknown',
      zone: params.get('zone') || 'default',
      position: params.get('position') || 'main'
    },
    settings: {
      logLevel: 'info',
      healthCheckInterval: 5000,
      reconnectInterval: 1000,
      maxRetryCount: 10
    }
  };

  return (
    <div className="h-screen w-screen bg-black overflow-hidden">
      <ContentController 
        config={config}
        wsURL={wsURL}
      />
    </div>
  );
}

export default App;
