import React from 'react';
import ChatContainer from './components/Chat/ChatContainer';
import ProviderSelector from './components/Provider/ProviderSelector';
import DiffViewer from './components/Diff/DiffViewer';
import { useChat } from './hooks/useChat';
import { useVoice } from './hooks/useVoice';

function App() {
  const { messages, sendMessage, isLoading, diff } = useChat();
  const { isListening, startListening, stopListening } = useVoice(sendMessage);

  return (
    <div className="min-h-screen bg-gray-50">
      <header className="bg-white shadow-sm border-b">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between items-center py-4">
            <h1 className="text-2xl font-bold text-gray-900">FlyAGI v2</h1>
            <ProviderSelector />
          </div>
        </div>
      </header>
      
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="space-y-4">
            <ChatContainer 
              messages={messages}
              onSendMessage={sendMessage}
              isLoading={isLoading}
              isListening={isListening}
              onStartListening={startListening}
              onStopListening={stopListening}
            />
          </div>
          
          <div className="space-y-4">
            {diff && (
              <DiffViewer diff={diff} />
            )}
          </div>
        </div>
      </main>
    </div>
  );
}

export default App;