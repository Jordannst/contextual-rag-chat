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
  attachment?: {
    fileName: string;
    fileType: string;
  };
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

    // Create initial AI message that will be updated incrementally
    const aiMessageId = (Date.now() + 1).toString();
    const aiMessage: Message = {
      id: aiMessageId,
      text: '',
      isUser: false,
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, aiMessage]);

    try {
      // Convert messages to history format for backend
      // Include all previous messages (before the current user message)
      const history = messages.map((msg) => ({
        role: msg.isUser ? 'user' : 'model',
        content: msg.text,
      }));

      // Call chat API with streaming
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
        // Try to parse error response
        try {
          const errorData = await response.json();
          throw new Error(errorData.message || 'Failed to get response');
        } catch {
          throw new Error(`HTTP ${response.status}: ${response.statusText}`);
        }
      }

      // Check if response is streamable
      if (!response.body) {
        throw new Error('Response body is null');
      }

      // Verify content type is SSE
      const contentType = response.headers.get('content-type');
      if (!contentType || !contentType.includes('text/event-stream')) {
        // Fallback: try to parse as JSON (non-streaming response)
        const errorData = await response.json();
        throw new Error(errorData.message || errorData.error || 'Unexpected response format');
      }

      // Get reader from response stream
      const reader = response.body.getReader();
      const decoder = new TextDecoder();

      // Buffer to accumulate partial SSE messages
      let buffer = '';
      let sources: string[] = [];
      let sourceIds: number[] = [];

      // Read stream chunks
      while (true) {
        const { done, value } = await reader.read();

        if (done) {
          break;
        }

        // Decode chunk
        const chunk = decoder.decode(value, { stream: true });
        buffer += chunk;

        // Process complete SSE messages (separated by double newline)
        const messages = buffer.split('\n\n');
        buffer = messages.pop() || ''; // Keep incomplete message in buffer

        for (const message of messages) {
          if (!message.trim()) {
            continue;
          }

          // Parse SSE message format:
          // event: <type>\ndata: <json>\n\n
          // or just: data: <json>\n\n
          let eventType: string | null = null;
          let dataLine: string | null = null;

          const lines = message.split('\n');
          for (const line of lines) {
            const trimmedLine = line.trim();
            if (trimmedLine.startsWith('event: ')) {
              eventType = trimmedLine.slice(7).trim();
            } else if (trimmedLine.startsWith('data: ')) {
              dataLine = trimmedLine.slice(6).trim();
            }
          }

          // Parse data if available
          if (dataLine) {
            try {
              const data = JSON.parse(dataLine);

              // Handle metadata event (sources info)
              if (eventType === 'metadata' || data.type === 'metadata') {
                sources = data.sources || [];
                sourceIds = data.sourceIds || [];
                console.log('Received metadata:', { sourcesCount: sources.length, sourceIds });
              }
              // Handle chunk data (text streaming)
              else if (data.type === 'chunk' && data.chunk) {
                const chunkText = data.chunk;
                if (chunkText) {
                  // Append chunk text to AI message in real-time
                  setMessages((prev) =>
                    prev.map((msg) =>
                      msg.id === aiMessageId
                        ? { ...msg, text: msg.text + chunkText }
                        : msg
                    )
                  );
                }
              }
              // Handle done event (streaming completed)
              else if (eventType === 'done' || data.type === 'done') {
                console.log('Streaming completed:', {
                  totalChunks: data.totalChunks,
                  fullLength: data.fullLength,
                });
                break; // Exit the message processing loop
              }
              // Handle error event
              else if (eventType === 'error' || data.type === 'error') {
                throw new Error(data.message || data.error || 'Streaming error occurred');
              }
              // Fallback: if no type but has chunk field, treat as chunk
              else if (data.chunk && typeof data.chunk === 'string') {
                setMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === aiMessageId
                      ? { ...msg, text: msg.text + data.chunk }
                      : msg
                  )
                );
              }
            } catch (parseError) {
              // If JSON parsing fails, log but don't break the stream
              console.error('Error parsing SSE data:', parseError, 'Raw data:', dataLine);
            }
          }
        }
      }

      // Process any remaining buffer
      if (buffer.trim()) {
        const lines = buffer.split('\n');
        for (const line of lines) {
          const trimmedLine = line.trim();
          if (trimmedLine.startsWith('data: ')) {
            const dataLine = trimmedLine.slice(6).trim();
            try {
              const data = JSON.parse(dataLine);
              if (data.type === 'chunk' && data.chunk) {
                setMessages((prev) =>
                  prev.map((msg) =>
                    msg.id === aiMessageId
                      ? { ...msg, text: msg.text + data.chunk }
                      : msg
                  )
                );
              }
            } catch (parseError) {
              console.error('Error parsing final buffer:', parseError);
            }
            break;
          }
        }
      }

      // Final check: if AI message is still empty, show error
      setMessages((prev) => {
        const updated = prev.map((msg) =>
          msg.id === aiMessageId && !msg.text.trim()
            ? {
                ...msg,
                text: 'No response received from the server.',
              }
            : msg
        );
        return updated;
      });
    } catch (error) {
      // Remove empty AI message and add error message
      setMessages((prev) => {
        const filtered = prev.filter((msg) => msg.id !== aiMessageId);
        const errorMessage: Message = {
          id: (Date.now() + 2).toString(),
          text: `Error: ${error instanceof Error ? error.message : 'Failed to get response from server'}`,
          isUser: false,
          timestamp: new Date(),
        };
        return [...filtered, errorMessage];
      });
    } finally {
      setIsLoading(false);
      scrollToBottom();
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

      // Get file type from file extension
      const fileExtension = file.name.split('.').pop()?.toLowerCase() || 'file';
      const fileType = fileExtension === 'pdf' ? 'pdf' : 
                      ['doc', 'docx'].includes(fileExtension) ? 'document' :
                      ['xls', 'xlsx'].includes(fileExtension) ? 'spreadsheet' :
                      ['jpg', 'jpeg', 'png', 'gif'].includes(fileExtension) ? 'image' :
                      'file';

      // Add user message with file attachment (not system message)
      const uploadMessage: Message = {
        id: Date.now().toString(),
        text: 'Uploaded a file',
        isUser: true,
        timestamp: new Date(),
        attachment: {
          fileName: file.name,
          fileType: fileType,
        },
      };

      // Add AI confirmation message
      const confirmationMessage: Message = {
        id: (Date.now() + 1).toString(),
        text: `File berhasil diupload. Apa yang ingin kamu tanyakan tentang file ini?`,
        isUser: false,
        timestamp: new Date(),
      };

      setMessages([uploadMessage, confirmationMessage]);
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
              {messages
                .filter((message) => {
                  // Show user messages always
                  // Show AI messages only if they have text or if currently streaming
                  if (message.isUser) return true;
                  // Show AI message if it has text, or if it's the last message and currently loading (streaming)
                  const isLastMessage = messages[messages.length - 1]?.id === message.id;
                  return message.text.trim() || (isLastMessage && isLoading);
                })
                .map((message) => (
                  <ChatBubble
                    key={message.id}
                    message={message.text || (isLoading && messages[messages.length - 1]?.id === message.id ? '' : message.text)}
                    isUser={message.isUser}
                    timestamp={message.timestamp}
                    attachment={message.attachment}
                  />
                ))}
              {/* Show typing indicator only if loading and no AI message bubble is shown */}
              {isLoading && 
               messages.length > 0 && 
               messages[messages.length - 1]?.isUser && (
                <TypingIndicator />
              )}
              <div ref={chatEndRef} />
            </ChatContainer>

            <ChatInput onSend={handleSendMessage} isLoading={isLoading} />
          </div>
        )}
      </main>
    </div>
  );
}
