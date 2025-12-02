'use client';

import React, { useState, useRef, useEffect } from 'react';
import { AnimatePresence, motion } from 'framer-motion';
import Sidebar from '@/components/layout/Sidebar';
import ChatContainer from '@/components/chat/ChatContainer';
import ChatBubble from '@/components/chat/ChatBubble';
import ChatInput from '@/components/chat/ChatInput';
import TypingIndicator from '@/components/chat/TypingIndicator';
import UploadCard from '@/components/upload/UploadCard';
import DocumentList from '@/components/upload/DocumentList';
import PDFViewerPanel from '@/components/ui/PDFViewerPanel';
import ConfirmDialog from '@/components/ui/ConfirmDialog';

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
  const [availableDocuments, setAvailableDocuments] = useState<string[]>([]); // List of available documents for filtering
  const [selectedDocFilters, setSelectedDocFilters] = useState<string[]>([]); // Selected document filters for chat
  const [currentSessionId, setCurrentSessionId] = useState<number | null>(null); // Current chat session ID
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null);
  const [sessions, setSessions] = useState<Array<{ id: number; title: string; created_at: string }>>([]); // Chat sessions for sidebar
  const [deleteConfirm, setDeleteConfirm] = useState<{ isOpen: boolean; sessionId: number | null; sessionTitle: string }>({
    isOpen: false,
    sessionId: null,
    sessionTitle: '',
  });

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

  // Fetch available documents for filter
  const fetchDocuments = async () => {
    try {
      const response = await fetch('http://localhost:5000/api/documents');
      if (response.ok) {
        const data = await response.json();
        setAvailableDocuments(data.documents || []);
      }
    } catch (error) {
      console.error('Error fetching documents for filter:', error);
    }
  };

  useEffect(() => {
    fetchDocuments();
    // Refresh when document list changes
    const interval = setInterval(fetchDocuments, 5000); // Refresh every 5 seconds
    return () => clearInterval(interval);
  }, [documentListRefreshTrigger]);

  // Show toast notification
  const showToast = (message: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToast({ message, type });
    setTimeout(() => {
      setToast(null);
    }, 4000);
  };

  // Load chat sessions on mount
  useEffect(() => {
    const fetchSessions = async () => {
      try {
        const response = await fetch('http://localhost:5000/api/sessions');
        if (response.ok) {
          const data = await response.json();
          setSessions(data.sessions || []);
        }
      } catch (error) {
        console.error('Error fetching sessions:', error);
      }
    };

    fetchSessions();
  }, []);

  // Function to handle session selection
  const handleSelectSession = async (sessionId: number) => {
    setCurrentSessionId(sessionId);
    
    try {
      const response = await fetch(`http://localhost:5000/api/sessions/${sessionId}`);
      if (response.ok) {
        const data = await response.json();
        const dbMessages = data.messages || [];
        
        // Convert DB messages to UI messages
        const uiMessages: Message[] = dbMessages.map((msg: any) => ({
          id: msg.id.toString(),
          text: msg.content,
          isUser: msg.role === 'user',
          timestamp: new Date(msg.created_at),
        }));
        
        setMessages(uiMessages);
      } else {
        console.error('Failed to load session messages');
      }
    } catch (error) {
      console.error('Error loading session messages:', error);
    }
  };

  // Function to open delete confirmation dialog
  const handleDeleteClick = (sessionId: number) => {
    const session = sessions.find(s => s.id === sessionId);
    const sessionTitle = session?.title || 'this chat';
    setDeleteConfirm({
      isOpen: true,
      sessionId,
      sessionTitle,
    });
  };

  // Function to handle session deletion (after confirmation)
  const handleDeleteSession = async () => {
    if (!deleteConfirm.sessionId) return;

    const sessionId = deleteConfirm.sessionId;
    
    try {
      const response = await fetch(`http://localhost:5000/api/sessions/${sessionId}`, {
        method: 'DELETE',
      });
      
      if (response.ok) {
        // Close dialog
        setDeleteConfirm({ isOpen: false, sessionId: null, sessionTitle: '' });
        
        // Refresh sessions list
        const sessionsResponse = await fetch('http://localhost:5000/api/sessions');
        if (sessionsResponse.ok) {
          const data = await sessionsResponse.json();
          setSessions(data.sessions || []);
        }
        
        // If deleted session is currently active, reset chat
        if (currentSessionId === sessionId) {
          setCurrentSessionId(null);
          setMessages([]);
        }
      } else {
        console.error('Failed to delete session');
        alert('Failed to delete chat. Please try again.');
      }
    } catch (error) {
      console.error('Error deleting session:', error);
      alert('An error occurred while deleting the chat. Please try again.');
    }
  };

  // Function to handle new chat
  const handleNewChat = () => {
    setCurrentSessionId(null);
    setMessages([]);
  };

  // Function to handle regenerate - regenerate the last AI response
  const handleRegenerate = () => {
    // Find the last user message (the question that was asked)
    const lastUserMessage = [...messages].reverse().find(msg => msg.isUser);
    
    if (!lastUserMessage) {
      console.warn('No user message found to regenerate');
      return;
    }

    // Remove the last AI message (the one we want to regenerate)
    setMessages((prev) => {
      // Find the index of the last non-user message
      let lastAIIndex = -1;
      for (let i = prev.length - 1; i >= 0; i--) {
        if (!prev[i].isUser) {
          lastAIIndex = i;
          break;
        }
      }
      
      // If found, remove it
      if (lastAIIndex >= 0) {
        return prev.filter((_, index) => index !== lastAIIndex);
      }
      
      return prev;
    });

    // Get the selectedFiles from the last user message if available
    // Note: We'll need to track this, but for now we'll regenerate without file filter
    // You can enhance this later to remember the file filter
    
    // Call handleSendMessage with the last user question
    handleSendMessage(lastUserMessage.text);
  };

  const handleSendMessage = async (text: string, selectedFiles?: string[]) => {
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
      const requestBody: {
        question: string;
        history: Array<{ role: string; content: string }>;
        selectedFiles?: string[];
        sessionId?: number;
      } = {
        question: text,
        history: history,
      };

      // Use selectedDocFilters if no selectedFiles provided, or merge them
      const filesToUse = selectedFiles && selectedFiles.length > 0 
        ? selectedFiles 
        : selectedDocFilters.length > 0 
          ? selectedDocFilters 
          : undefined;

      // Add selectedFiles only if provided (not empty)
      if (filesToUse && filesToUse.length > 0) {
        requestBody.selectedFiles = filesToUse;
      }

      // Add sessionId if exists
      if (currentSessionId !== null) {
        requestBody.sessionId = currentSessionId;
      }

      const response = await fetch('http://localhost:5000/api/chat', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(requestBody),
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

              // Handle metadata event (sources info + sessionId)
              if (eventType === 'metadata' || data.type === 'metadata') {
                sources = data.sources || [];
                sourceIds = data.sourceIds || [];
                console.log('Received metadata:', { sourcesCount: sources.length, sourceIds });
                
                // Handle new session ID if provided
                if (data.sessionId && currentSessionId === null) {
                  setCurrentSessionId(data.sessionId);
                  // Refresh sessions list to show new chat
                  const sessionsResponse = await fetch('http://localhost:5000/api/sessions');
                  if (sessionsResponse.ok) {
                    const sessionsData = await sessionsResponse.json();
                    setSessions(sessionsData.sessions || []);
                  }
                }
                
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
              // Handle chart event (data visualization)
              else if (eventType === 'chart' || data.type === 'chart') {
                const chartData = data.chartData;
                if (chartData && typeof chartData === 'string') {
                  // Append chart marker to message content so ChatBubble can render it
                  // Format: [CHART_DATA:...base64...]
                  const chartMarker = `\n[CHART_DATA:${chartData}]`;
                  setMessages((prev) =>
                    prev.map((msg) =>
                      msg.id === aiMessageId
                        ? { ...msg, text: msg.text + chartMarker }
                        : msg
                    )
                  );
                  console.log('Received chart data:', { index: data.index || 0, dataLength: chartData.length });
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

  // Handle file upload from chat input
  const handleChatFileUpload = async (files: File[]) => {
    if (files.length === 0) return;

    // Show loading toast
    showToast(`Mengunggah ${files.length} file...`, 'info');

    const uploadedFileNames: string[] = [];
    const errors: string[] = [];

    // Upload files one by one
    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      
      // Update toast with progress
      if (files.length > 1) {
        showToast(`Mengunggah ${i + 1}/${files.length} file...`, 'info');
      }

      try {
        const formData = new FormData();
        formData.append('document', file);

        const response = await fetch('http://localhost:5000/api/upload', {
          method: 'POST',
          body: formData,
        });

        if (!response.ok) {
          const errorData = await response.json();
          throw new Error(errorData.message || `Failed to upload ${file.name}`);
        }

        const data = await response.json();
        uploadedFileNames.push(data.fileName || file.name);
      } catch (error) {
        const errorMsg = `Error uploading ${file.name}: ${error instanceof Error ? error.message : 'Unknown error'}`;
        errors.push(errorMsg);
        console.error(errorMsg, error);
      }
    }

    // Refresh document list
    await fetchDocuments();

    // Auto-select newly uploaded files
    if (uploadedFileNames.length > 0) {
      setSelectedDocFilters((prev) => {
        const newFilters = [...prev];
        uploadedFileNames.forEach((fileName) => {
          if (!newFilters.includes(fileName)) {
            newFilters.push(fileName);
          }
        });
        return newFilters;
      });

      // Show success toast
      if (uploadedFileNames.length === 1) {
        showToast(`File "${uploadedFileNames[0]}" berhasil ditambahkan ke percakapan.`, 'success');
      } else {
        showToast(`${uploadedFileNames.length} file berhasil ditambahkan ke percakapan.`, 'success');
      }
    }

    // Show error toast if any
    if (errors.length > 0) {
      showToast(`Beberapa file gagal diupload: ${errors.join(', ')}`, 'error');
    }

    // Trigger document list refresh
    setDocumentListRefreshTrigger((prev) => prev + 1);
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
    <div className="h-screen bg-transparent flex overflow-hidden transition-colors duration-300">
      {/* Toast Notification */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 animate-fade-slide-up">
          <div
            className={`
              px-4 py-3 rounded-lg shadow-lg border flex items-center gap-3 backdrop-blur-lg
              ${toast.type === 'success'
                ? 'bg-green-900/90 border-green-700 text-green-100'
                : toast.type === 'error'
                ? 'bg-red-900/90 border-red-700 text-red-100'
                : 'bg-blue-900/90 border-blue-700 text-blue-100'
              }
            `}
          >
            {toast.type === 'success' ? (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            ) : toast.type === 'error' ? (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            ) : (
              <svg className="w-5 h-5 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
            )}
            <span className="text-sm font-medium">{toast.message}</span>
            <button
              onClick={() => setToast(null)}
              className="ml-2 text-current opacity-70 hover:opacity-100 transition-opacity"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}
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
        sessions={sessions}
        currentSessionId={currentSessionId}
        onSelectSession={handleSelectSession}
        onDeleteSession={handleDeleteClick}
        onNewChat={handleNewChat}
      />

      {/* Main Content - Split Screen Layout */}
      <main className="flex-1 flex flex-row ml-16 overflow-hidden">
        {/* Chat Area - Dynamic Width */}
        <div className={`flex-1 flex flex-col transition-all duration-300 ${
          selectedDocument ? 'lg:w-1/2' : 'w-full'
        } ${!isUploading && messages.length > 0 ? 'overflow-hidden' : 'overflow-y-auto'}`}>
        {/* CRITICAL: Chat Interface only appears if NOT uploading AND has messages */}
        {/* This ensures UI stays on upload page during entire upload process */}
        {!isUploading && messages.length > 0 ? (
          /* Chat Interface */
          <div className="flex-1 flex flex-col h-full overflow-hidden bg-transparent transition-colors duration-300">
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
                .map((message, index) => {
                  // Check if this is the last AI message (for regenerate button)
                  const isLastAIMessage = !message.isUser && 
                    (index === messages.length - 1 || 
                     (index < messages.length - 1 && messages[index + 1]?.isUser));
                  
                  return (
                    <ChatBubble
                      key={message.id}
                      message={message.text || (isLoading && index === messages.length - 1 ? '' : message.text)}
                      isUser={message.isUser}
                      timestamp={message.timestamp}
                      attachment={message.attachment}
                      sources={message.sources}
                      onViewDocument={(fileName) => setSelectedDocument(fileName)}
                      onRegenerate={!message.isUser ? handleRegenerate : undefined}
                      isLastMessage={isLastAIMessage}
                    />
                  );
                })}
              {/* Show typing indicator only if loading and no AI message bubble is shown */}
              {isLoading && 
               messages.length > 0 && 
               messages[messages.length - 1]?.isUser && (
                <TypingIndicator />
              )}
              <div ref={chatEndRef} />
            </ChatContainer>

            <ChatInput 
              onSend={handleSendMessage}
              onFileUpload={handleChatFileUpload}
              isLoading={isLoading}
              availableDocuments={availableDocuments}
              selectedDocFilters={selectedDocFilters}
              onSelectedDocFiltersChange={setSelectedDocFilters}
            />
          </div>
        ) : (
          /* Hero Section - Gemini 2025 Style */
          <div className="flex-1 overflow-y-auto px-4 py-12">
            <div className="w-full max-w-[900px] mx-auto min-h-full flex flex-col items-center justify-center">
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
        </div>

        {/* PDF Viewer Panel - Split Screen (Desktop) / Overlay (Mobile) */}
        <AnimatePresence mode="wait">
          {selectedDocument && (
            <motion.div
              key="pdf-panel"
              initial={{ x: '100%', opacity: 0 }}
              animate={{ x: 0, opacity: 1 }}
              exit={{ x: '100%', opacity: 0 }}
              transition={{ type: "spring", stiffness: 300, damping: 30 }}
              className="hidden lg:flex lg:w-1/2"
            >
              <PDFViewerPanel
                fileName={selectedDocument}
                onClose={() => setSelectedDocument(null)}
              />
            </motion.div>
          )}
        </AnimatePresence>
        
        {/* Mobile PDF Viewer (Overlay) - No animation needed, handled by PDFViewerPanel */}
        {selectedDocument && (
          <div className="lg:hidden">
            <PDFViewerPanel
              fileName={selectedDocument}
              onClose={() => setSelectedDocument(null)}
            />
          </div>
        )}

        {/* Delete Confirmation Dialog */}
        <ConfirmDialog
          isOpen={deleteConfirm.isOpen}
          title="Delete Chat"
          message={`Are you sure you want to delete "${deleteConfirm.sessionTitle}"? This action cannot be undone and all messages in this chat will be permanently deleted.`}
          confirmText="Delete"
          cancelText="Cancel"
          onConfirm={handleDeleteSession}
          onCancel={() => setDeleteConfirm({ isOpen: false, sessionId: null, sessionTitle: '' })}
          variant="danger"
        />
      </main>
    </div>
  );
}
