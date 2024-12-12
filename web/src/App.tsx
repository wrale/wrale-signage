import React, { useState, useEffect } from 'react';
import { ContentController } from './components/ContentController';
import { RegistrationService } from './services/registration';
import type { DisplayConfig } from './types';

function App() {
  const params = new URLSearchParams(window.location.search);
  const displayId = params.get('id') || 'unregistered';
  const apiBase = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1alpha1`;
  
  const [registrationCode, setRegistrationCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

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

  // Handle device registration
  useEffect(() => {
    const registration = new RegistrationService(config);

    const checkRegistration = async () => {
      try {
        // Check if already registered
        if (await registration.isValidRegistration()) {
          setRegistrationCode(null);
          return;
        }

        // Start registration flow
        const response = await registration.startRegistration();
        setRegistrationCode(response.userCode);

        // Start polling for activation
        await registration.pollForActivation(
          response.deviceCode, 
          response.pollInterval
        );

        // Activation succeeded
        setRegistrationCode(null);
      } catch (err) {
        console.error('Registration error:', err);
        setError(err instanceof Error ? err.message : 'Registration failed');
      }
    };

    checkRegistration();

    return () => registration.dispose();
  }, [config]);

  // Show error if registration failed
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-screen text-white bg-black">
        <h1 className="text-2xl mb-4">Registration Error</h1>
        <p className="text-red-500">{error}</p>
        <button 
          className="mt-4 px-4 py-2 bg-blue-500 rounded hover:bg-blue-600"
          onClick={() => window.location.reload()}
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="h-screen w-screen bg-black overflow-hidden">
      {registrationCode ? (
        <div className="flex flex-col items-center justify-center h-full text-white">
          <h1 className="text-2xl mb-4">Display Registration</h1>
          <p className="text-lg mb-8">Please visit activate.example.com and enter:</p>
          <div className="text-4xl font-mono bg-gray-800 p-4 rounded-lg mb-8">
            {registrationCode}
          </div>
          <p className="text-sm text-gray-400">
            This code will expire in 15 minutes
          </p>
        </div>
      ) : (
        <ContentController 
          config={config}
          wsURL={`${apiBase}/displays/ws`}
        />
      )}
    </div>
  );
}

export default App;