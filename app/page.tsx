'use client';

import React, { useState, useRef, useEffect } from 'react';
import Sidebar from '@/components/layout/Sidebar';
import ChatContainer from '@/components/chat/ChatContainer';
import ChatBubble from '@/components/chat/ChatBubble';
import ChatInput from '@/components/chat/ChatInput';
import TypingIndicator from '@/components/chat/TypingIndicator';
import PromptCard from '@/components/ui/PromptCard';
import UploadCard from '@/components/upload/UploadCard';

interface Message {
  id: string;
  text: string;
  isUser: boolean;
  timestamp: Date;
}

const defaultPrompts = [
  {
    id: '1',
    title: 'Plan a trip',
    description: 'Plan a detailed itinerary for a weekend getaway',
    icon: (
      <svg className="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 21v-4m0 0V5a2 2 0 012-2h6.5l1 1H21l-3 6 3 6h-8.5l-1-1H5a2 2 0 00-2 2zm9-13.5V9" />
      </svg>
    ),
  },
  {
    id: '2',
    title: 'Write a story',
    description: 'Create a short story about a magical adventure',
    icon: (
      <svg className="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
      </svg>
    ),
  },
  {
    id: '3',
    title: 'Explain a concept',
    description: 'Break down complex topics into simple explanations',
    icon: (
      <svg className="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8.228 9c.549-1.165 2.03-2 3.772-2 2.21 0 4 1.343 4 3 0 1.4-1.278 2.575-3.006 2.907-.542.104-.994.54-.994 1.093m0 3h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
      </svg>
    ),
  },
  {
    id: '4',
    title: 'Brainstorm ideas',
    description: 'Generate creative ideas for your next project',
    icon: (
      <svg className="w-4 h-4 text-neutral-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z" />
      </svg>
    ),
  },
];

export default function Home() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const chatEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  const handleSendMessage = async (text: string) => {
    const userMessage: Message = {
      id: Date.now().toString(),
      text,
      isUser: true,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMessage]);
    setIsLoading(true);

    try {
      // Convert messages to history format for backend
      // Include all previous messages (before the current user message)
      const history = messages.map((msg) => ({
        role: msg.isUser ? 'user' : 'model',
        content: msg.text,
      }));

      // Call chat API with history
      const response = await fetch('http://localhost:5000/api/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          question: text,
          history: history,
        }),
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to get response');
      }

      const data = await response.json();

      const aiResponse: Message = {
        id: (Date.now() + 1).toString(),
        text: data.response || 'No response received',
        isUser: false,
        timestamp: new Date(),
      };

      setMessages((prev) => [...prev, aiResponse]);
    } catch (error) {
      const errorMessage: Message = {
        id: (Date.now() + 1).toString(),
        text: `Error: ${error instanceof Error ? error.message : 'Failed to get response from server'}`,
        isUser: false,
        timestamp: new Date(),
      };
      setMessages((prev) => [...prev, errorMessage]);
    } finally {
      setIsLoading(false);
    }
  };

  const handlePromptClick = (prompt: typeof defaultPrompts[0]) => {
    handleSendMessage(prompt.title);
  };

  const handleNewChat = () => {
    setMessages([]);
  };

  const handleFileUpload = async (file: File) => {
    try {
      // Create form data
      const formData = new FormData();
      formData.append('document', file);

      // Call upload API
      const response = await fetch('http://localhost:5000/api/upload', {
        method: 'POST',
        body: formData,
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to upload file');
      }

      const data = await response.json();

      // Tambahkan pesan selamat datang
      const welcomeMessage: Message = {
        id: Date.now().toString(),
        text: data.message || `Dokumen "${file.name}" berhasil diunggah! Anda sekarang dapat menanyakan sesuatu tentang dokumen ini.`,
        isUser: false,
        timestamp: new Date(),
      };
      setMessages([welcomeMessage]);
    } catch (error) {
      const errorMessage: Message = {
        id: Date.now().toString(),
        text: `Error: ${error instanceof Error ? error.message : 'Failed to upload file'}`,
        isUser: false,
        timestamp: new Date(),
      };
      setMessages([errorMessage]);
    }
  };

  return (
    <div className="min-h-screen bg-gradient-to-b from-neutral-950 to-neutral-900 flex transition-colors duration-300">
      {/* Sidebar */}
      <Sidebar 
        items={[
          {
            id: 'new-chat',
            label: 'New chat',
            icon: (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
              </svg>
            ),
            onClick: handleNewChat,
          },
        ]}
      />

      {/* Main Content */}
      <main className="flex-1 flex flex-col ml-16">
        {messages.length === 0 ? (
          /* Hero Section - Gemini 2025 Style */
          <div className="flex-1 flex items-center justify-center px-4 py-12">
            <div className="w-full max-w-[900px] mx-auto">
              <div className="text-center mb-12 animate-fade-slide-up">
                <h1 className="text-5xl sm:text-6xl font-bold text-neutral-100 mb-4 leading-tight tracking-tight transition-colors duration-300">
                  Hello again 👋
                </h1>
                <p className="text-xl sm:text-2xl text-neutral-400 font-normal transition-colors duration-300">
                  What can I help you explore today?
                </p>
              </div>

              {/* Upload Card */}
              <div className="max-w-2xl mx-auto mb-8 animate-fade-slide-up" style={{ animationDelay: '0.1s' }}>
                <UploadCard
                  onFileUpload={handleFileUpload}
                  isUploading={false}
                />
              </div>

              {/* Prompt Cards Grid */}
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 max-w-2xl mx-auto animate-fade-slide-up" style={{ animationDelay: '0.2s' }}>
                {defaultPrompts.map((prompt) => (
                  <PromptCard
                    key={prompt.id}
                    title={prompt.title}
                    description={prompt.description}
                    icon={prompt.icon}
                    onClick={() => handlePromptClick(prompt)}
                  />
                ))}
              </div>

              {/* Additional Info */}
              <div className="mt-12 text-center animate-fade-slide-up" style={{ animationDelay: '0.3s' }}>
                <p className="text-sm text-neutral-400 transition-colors duration-300">
                  You can start a conversation or pick a suggestion
                </p>
              </div>
            </div>
          </div>
        ) : (
          /* Chat Interface */
          <div className="flex-1 flex flex-col h-screen overflow-hidden bg-neutral-950 transition-colors duration-300">
            <ChatContainer>
              {messages.map((message) => (
                <ChatBubble
                  key={message.id}
                  message={message.text}
                  isUser={message.isUser}
                  timestamp={message.timestamp}
                />
              ))}
              {isLoading && <TypingIndicator />}
              <div ref={chatEndRef} />
            </ChatContainer>

            <ChatInput onSend={handleSendMessage} isLoading={isLoading} />
          </div>
        )}
      </main>
    </div>
  );
}
