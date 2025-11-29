'use client';

import React, { useState, useRef, useEffect } from 'react';
import Sidebar from '@/components/layout/Sidebar';
import ChatContainer from '@/components/chat/ChatContainer';
import ChatBubble from '@/components/chat/ChatBubble';
import ChatInput from '@/components/chat/ChatInput';
import TypingIndicator from '@/components/chat/TypingIndicator';
import UploadCard from '@/components/upload/UploadCard';
import DocumentList from '@/components/upload/DocumentList';
import PDFViewerModal from '@/components/ui/PDFViewerModal';

interface Message {
  id: string;
  text: string;
  isUser: boolean;
  timestamp: Date;
  attachment?: {
    fileName: string;
    fileType: string;
  };
  sources?: string[];
}


export default function Home() {
  const [messages, setMessages] = useState<Message[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState<{ current: number; total: number; currentFileName?: string } | undefined>(undefined);
  const [selectedFiles, setSelectedFiles] = useState<File[]>([]); // Files selected but not yet uploaded
  const chatEndRef = useRef<HTMLDivElement>(null);
  const [documentListRefreshTrigger, setDocumentListRefreshTrigger] = useState(0);
  const [selectedDocument, setSelectedDocument] = useState<string | null>(null); // PDF file to view in modal
  const [suggestions, setSuggestions] = useState<string[]>([]); // Question suggestions
  const [isLoadingSuggestions, setIsLoadingSuggestions] = useState(false);

  const scrollToBottom = () => {
    chatEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages, isLoading]);

  // Fetch question suggestions when no messages and not uploading
  useEffect(() => {
    // Clear suggestions when user starts chatting
    if (messages.length > 0) {
      setSuggestions([]);
      return;
    }

    // Don't fetch if uploading or already loading
    if (isUploading || isLoadingSuggestions) {
      return;
    }

    // Fetch suggestions
    const fetchSuggestions = async () => {
      setIsLoadingSuggestions(true);
      try {
        const response = await fetch('http://localhost:5000/api/chat/suggestions');
        if (response.ok) {
          const data = await response.json();
          setSuggestions(data.questions || []);
        } else {
          console.error('Failed to fetch suggestions');
          setSuggestions([]);
        }
      } catch (error) {
        console.error('Error fetching suggestions:', error);
        setSuggestions([]);
      } finally {
        setIsLoadingSuggestions(false);
      }
    };

    fetchSuggestions();
  }, [messages.length, isUploading]);

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
                
                // Save sources to AI message
                if (sources.length > 0) {
                  setMessages((prev) =>
                    prev.map((msg) =>
                      msg.id === aiMessageId
                        ? { ...msg, sources: sources }
                        : msg
                    )
                  );
                }
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


  const handleNewChat = () => {
    setMessages([]);
  };

  // Handle file selection - only add to selectedFiles, don't upload yet
  const handleFileSelect = (files: File[]) => {
    if (files.length === 0) return;
    
    // Add new files to selectedFiles (avoid duplicates by filename)
    setSelectedFiles((prev) => {
      const existingNames = new Set(prev.map(f => f.name));
      const newFiles = files.filter(f => !existingNames.has(f.name));
      return [...prev, ...newFiles];
    });
  };

  // Remove file from selectedFiles
  const handleRemoveFile = (index: number) => {
    setSelectedFiles((prev) => prev.filter((_, i) => i !== index));
  };

  // Start processing - upload all selected files
  const handleStartProcessing = async () => {
    if (selectedFiles.length === 0) return;

    // Set uploading state - this ensures UI stays on upload page
    setIsUploading(true);
    setUploadProgress({ current: 0, total: selectedFiles.length });

    // Temporary accumulator for messages - DO NOT update state messages during loop
    const newMessages: Message[] = [];
    const errors: string[] = [];

    // Process each file one by one
    for (let i = 0; i < selectedFiles.length; i++) {
      const file = selectedFiles[i];
      
      // Update ONLY progress state during loop - this is safe and needed for progress bar
      setUploadProgress({
        current: i,
        total: selectedFiles.length,
        currentFileName: file.name,
      });

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
          throw new Error(errorData.message || `Failed to upload ${file.name}`);
        }

        const data = await response.json();

        // Get file type from file extension
        const fileExtension = file.name.split('.').pop()?.toLowerCase() || 'file';
        const fileType = fileExtension === 'pdf' ? 'pdf' : 
                        ['doc', 'docx'].includes(fileExtension) ? 'document' :
                        ['xls', 'xlsx'].includes(fileExtension) ? 'spreadsheet' :
                        ['jpg', 'jpeg', 'png', 'gif'].includes(fileExtension) ? 'image' :
                        'file';

        // CRITICAL: Only push to local array, DO NOT call setMessages here
        // This prevents React from rendering Chat View during upload process
        const uploadMessage: Message = {
          id: (Date.now() + i).toString(),
          text: 'Uploaded a file',
          isUser: true,
          timestamp: new Date(),
          attachment: {
            fileName: file.name,
            fileType: fileType,
          },
        };

        newMessages.push(uploadMessage);
      } catch (error) {
        const errorMsg = `Error uploading ${file.name}: ${error instanceof Error ? error.message : 'Unknown error'}`;
        errors.push(errorMsg);
        console.error(errorMsg, error);
        // Continue with next file even if one fails
      }
    }

    // Update progress to show completion (while isUploading is still true)
    setUploadProgress({
      current: selectedFiles.length,
      total: selectedFiles.length,
    });

    // AFTER loop completes - prepare all messages in local array
    const allMessages: Message[] = [...newMessages];
    
    // Add confirmation message if any files were uploaded
    if (newMessages.length > 0) {
      const confirmationMessage: Message = {
        id: (Date.now() + selectedFiles.length + 1).toString(),
        text: selectedFiles.length === 1 
          ? `File berhasil diupload. Apa yang ingin kamu tanyakan tentang file ini?`
          : `${newMessages.length} file berhasil diupload. Apa yang ingin kamu tanyakan tentang file-file ini?`,
        isUser: false,
        timestamp: new Date(),
      };
      allMessages.push(confirmationMessage);
    }

    // Add error message if any
    if (errors.length > 0) {
      const errorMessage: Message = {
        id: (Date.now() + selectedFiles.length + 2).toString(),
        text: `Beberapa file gagal diupload:\n${errors.join('\n')}`,
        isUser: false,
        timestamp: new Date(),
      };
      allMessages.push(errorMessage);
    }

    // Refresh document list after all uploads complete
    setDocumentListRefreshTrigger((prev) => prev + 1);

    // Refresh suggestions after upload (new documents available)
    setSuggestions([]); // Clear to trigger refetch

    // Clear selected files
    setSelectedFiles([]);

    // CRITICAL: Only update messages state AFTER loop completes
    // This ensures React doesn't render Chat View during upload process
    if (allMessages.length > 0) {
      setMessages((prev) => [...prev, ...allMessages]);
    }

    // Reset upload state - this will trigger re-render and switch to Chat View
    // because now isUploading is false AND messages.length > 0
    setIsUploading(false);
    setUploadProgress(undefined);
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
        {/* CRITICAL: Chat Interface only appears if NOT uploading AND has messages */}
        {/* This ensures UI stays on upload page during entire upload process */}
        {!isUploading && messages.length > 0 ? (
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
                    sources={message.sources}
                    onViewDocument={(fileName) => setSelectedDocument(fileName)}
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
        ) : (
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
                  onFileSelect={handleFileSelect}
                  onStartProcessing={handleStartProcessing}
                  onRemoveFile={handleRemoveFile}
                  selectedFiles={selectedFiles}
                  isUploading={isUploading}
                  uploadProgress={uploadProgress}
                />
              </div>

              {/* Question Suggestions */}
              {suggestions.length > 0 && (
                <div className="max-w-4xl mx-auto mb-8 animate-fade-slide-up" style={{ animationDelay: '0.15s' }}>
                  <h2 className="text-lg font-semibold text-neutral-300 mb-4 text-center">
                    Suggested Questions
                  </h2>
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                    {suggestions.map((question, index) => (
                      <button
                        key={index}
                        onClick={() => handleSendMessage(question)}
                        disabled={isLoading}
                        className="text-left px-5 py-4 bg-neutral-800/50 hover:bg-neutral-800 border border-neutral-700 hover:border-neutral-600 rounded-xl transition-all duration-200 hover:scale-[1.02] disabled:opacity-50 disabled:cursor-not-allowed group"
                      >
                        <div className="flex items-start gap-3">
                          <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-blue-600/20 flex items-center justify-center mt-0.5 group-hover:bg-blue-600/30 transition-colors">
                            <svg
                              className="w-4 h-4 text-blue-400"
                              fill="none"
                              stroke="currentColor"
                              viewBox="0 0 24 24"
                            >
                              <path
                                strokeLinecap="round"
                                strokeLinejoin="round"
                                strokeWidth={2}
                                d="M8 10h.01M12 10h.01M16 10h.01M9 16H5a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v8a2 2 0 01-2 2h-5l-5 5v-5z"
                              />
                            </svg>
                          </div>
                          <p className="text-neutral-200 text-sm leading-relaxed flex-1">
                            {question}
                          </p>
                        </div>
                      </button>
                    ))}
                  </div>
                </div>
              )}

              {/* Document List */}
              <div className="max-w-2xl mx-auto animate-fade-slide-up" style={{ animationDelay: '0.2s' }}>
                <DocumentList 
                  onDocumentDeleted={() => {
                    // Optionally refresh or show notification when document is deleted
                    console.log('Document deleted, list refreshed');
                    // Refresh suggestions when document is deleted
                    setSuggestions([]);
                  }}
                  refreshTrigger={documentListRefreshTrigger}
                />
              </div>
            </div>
          </div>
        )}

        {/* PDF Viewer Modal */}
        <PDFViewerModal
          isOpen={selectedDocument !== null}
          onClose={() => setSelectedDocument(null)}
          fileName={selectedDocument}
        />
      </main>
    </div>
  );
}
