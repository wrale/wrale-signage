import React from 'react';
import BrowserInterface from './components/BrowserInterface';

function App() {
  const params = new URLSearchParams(window.location.search);
  const displayId = params.get('id') || '';
  const wsURL = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1alpha1/displays/ws`;

  return (
    <div className="h-screen bg-gray-100">
      <BrowserInterface 
        displayId={displayId}
        controlURL={wsURL}
      />
    </div>
  );
}

export default App;
