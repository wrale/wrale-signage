import React, { useState, useEffect, useRef } from 'react';
import { ContentController } from './components/ContentController';
import { RegistrationService } from './services/registration';
import type { DisplayConfig, RegistrationState } from './types';

interface RegistrationHandle {
  registration: RegistrationService;
  currentState: RegistrationState;
  abortController: AbortController;
  registrationPromise: Promise<void> | null;
}

function App() {
  const params = new URLSearchParams(window.location.search);
  const displayId = params.get('id') || 'unregistered';
  const apiBase = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1alpha1`;
  
  // UI state
  const [registrationCode, setRegistrationCode] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  
  // Registration handle
  const registrationRef = useRef<RegistrationHandle | null>(null);

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
    const initializeRegistration = async () => {
      // Cleanup existing registration if any
      if (registrationRef.current) {
        registrationRef.current.abortController.abort();
        registrationRef.current.registration.dispose();
      }

      // Create new registration handle
      const abortController = new AbortController();
      const registration = new RegistrationService(config);

      registrationRef.current = {
        registration,
        currentState: 'initializing',
        abortController,
        registrationPromise: null
      };

      try {
        // Check if already registered
        if (await registration.isValidRegistration()) {
          registrationRef.current.currentState = 'registered';
          setRegistrationCode(null);
          return;
        }

        // Start registration flow
        registrationRef.current.currentState = 'registering';
        const response = await registration.startRegistration();
        
        // Show registration code to user
        setRegistrationCode(response.userCode);
        
        // Start polling for activation
        registrationRef.current.currentState = 'polling';
        await registration.pollForActivation(
          response.deviceCode,
          response.pollInterval
        );

        // Registration complete
        registrationRef.current.currentState = 'registered';
        setRegistrationCode(null);
        setError(null);

      } catch (err) {
        // Only update error state if registration wasn't aborted
        if (!abortController.signal.aborted) {
          console.error('Registration error:', err);
          setError(err instanceof Error ? err.message : 'Registration failed');
          registrationRef.current.currentState = 'error';
        }
      }
    };

    // Start registration
    initializeRegistration();

    // Cleanup function
    return () => {
      if (registrationRef.current) {
        registrationRef.current.abortController.abort();
        registrationRef.current.registration.dispose();
        registrationRef.current = null;
      }
    };
  }, [config, retryCount]); // Include retryCount to allow manual retries

  // Handle retry button click
  const handleRetry = () => {
    setError(null);
    setRegistrationCode(null);
    setRetryCount(prev => prev + 1);
  };

  // Show error if registration failed
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-screen text-white bg-black">
        <h1 className="text-2xl mb-4">Registration Error</h1>
        <p className="text-red-500">{error}</p>
        <button 
          className="mt-4 px-4 py-2 bg-blue-500 rounded hover:bg-blue-600"
          onClick={handleRetry}
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