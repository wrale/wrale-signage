import React, { useState, useRef, useEffect } from 'react';
import { ArrowLeft, ArrowRight, RotateCcw, Home, Search } from 'lucide-react';
import { ContentSequence } from '../types';
import { ContentController } from './ContentController';

interface NavigationState {
  path: string;
  title: string;
}

const BrowserInterface: React.FC<{
  displayId: string;
  controlURL: string;
}> = ({ displayId, controlURL }) => {
  const [currentPath, setCurrentPath] = useState<string>('/page1.html');
  const [history, setHistory] = useState<NavigationState[]>([{ path: '/page1.html', title: 'Page 1' }]);
  const [historyIndex, setHistoryIndex] = useState<number>(0);
  const [isTransitioning, setIsTransitioning] = useState<boolean>(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const handlePathChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setCurrentPath(e.target.value);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    navigateToPath(currentPath);
  };

  const navigateToPath = (path: string) => {
    setIsTransitioning(true);
    const formattedPath = path.startsWith('/') ? path : `/${path}`;
    setCurrentPath(formattedPath);

    const newHistory = history.slice(0, historyIndex + 1);
    newHistory.push({ path: formattedPath, title: formattedPath });
    setHistory(newHistory);
    setHistoryIndex(newHistory.length - 1);

    if (iframeRef.current) {
      iframeRef.current.src = formattedPath;
    }

    setTimeout(() => {
      setIsTransitioning(false);
    }, 500);
  };

  const goBack = () => {
    if (historyIndex > 0) {
      setHistoryIndex(historyIndex - 1);
      const prevState = history[historyIndex - 1];
      setCurrentPath(prevState.path);
      if (iframeRef.current) {
        iframeRef.current.src = prevState.path;
      }
    }
  };

  const goForward = () => {
    if (historyIndex < history.length - 1) {
      setHistoryIndex(historyIndex + 1);
      const nextState = history[historyIndex + 1];
      setCurrentPath(nextState.path);
      if (iframeRef.current) {
        iframeRef.current.src = nextState.path;
      }
    }
  };

  const refresh = () => {
    if (iframeRef.current) {
      iframeRef.current.src = currentPath;
    }
  };

  const goHome = () => {
    navigateToPath('/page1.html');
  };

  // Handle control messages
  const handleSequenceUpdate = (sequence: ContentSequence) => {
    if (sequence.items.length > 0) {
      navigateToPath(sequence.items[0].url);
    }
  };

  const handleReloadRequired = () => {
    window.location.reload();
  };

  // Handle messages from iframe content
  useEffect(() => {
    const handleMessage = (event: MessageEvent) => {
      if (event.origin === window.location.origin) {
        if (event.data.type === 'navigation' && event.data.path) {
          navigateToPath(event.data.path);
        }
      }
    };

    window.addEventListener('message', handleMessage);
    return () => window.removeEventListener('message', handleMessage);
  }, [historyIndex]);

  return (
    <div className="flex flex-col w-full h-screen bg-gray-100">
      <div className="bg-white shadow-md p-4">
        <div className="flex items-center gap-2 mb-2">
          <button
            onClick={goBack}
            disabled={historyIndex === 0}
            className="p-2 rounded hover:bg-gray-100 disabled:opacity-50"
            aria-label="Go back"
          >
            <ArrowLeft size={20} />
          </button>
          <button
            onClick={goForward}
            disabled={historyIndex === history.length - 1}
            className="p-2 rounded hover:bg-gray-100 disabled:opacity-50"
            aria-label="Go forward"
          >
            <ArrowRight size={20} />
          </button>
          <button
            onClick={refresh}
            className="p-2 rounded hover:bg-gray-100"
            aria-label="Refresh page"
          >
            <RotateCcw size={20} />
          </button>
          <button
            onClick={goHome}
            className="p-2 rounded hover:bg-gray-100"
            aria-label="Go home"
          >
            <Home size={20} />
          </button>

          <form onSubmit={handleSubmit} className="flex-1 flex items-center">
            <div className="relative flex-1">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search size={16} className="text-gray-400" />
              </div>
              <input
                type="text"
                value={currentPath}
                onChange={handlePathChange}
                className="block w-full pl-10 pr-3 py-2 border border-gray-300 rounded-lg bg-gray-50 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Enter path"
              />
            </div>
          </form>
        </div>
      </div>

      <div className="flex-1 bg-white relative">
        <div className={`absolute inset-0 transition-opacity duration-500 ${isTransitioning ? 'opacity-100' : 'opacity-0 pointer-events-none'} bg-black`} />
        <iframe
          ref={iframeRef}
          src={currentPath}
          className="w-full h-full border-0"
          title="Browser Viewport"
        />
      </div>

      <ContentController
        displayId={displayId}
        wsURL={controlURL}
        onSequenceUpdate={handleSequenceUpdate}
        onReloadRequired={handleReloadRequired}
      />
    </div>
  );
};

export default BrowserInterface;
