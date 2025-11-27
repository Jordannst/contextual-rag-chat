'use client';

import React, { useState, useRef, useEffect } from 'react';

interface ChatInputProps {
  onSend: (message: string) => void;
  isLoading?: boolean;
  placeholder?: string;
}

export default function ChatInput({ 
  onSend, 
  isLoading = false,
  placeholder = 'Enter a prompt here'
}: ChatInputProps) {
  const [message, setMessage] = useState('');
  const [isFocused, setIsFocused] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      const scrollHeight = textareaRef.current.scrollHeight;
      textareaRef.current.style.height = `${Math.min(scrollHeight, 120)}px`;
    }
  }, [message]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (message.trim() && !isLoading) {
      onSend(message.trim());
      setMessage('');
      if (textareaRef.current) {
        textareaRef.current.style.height = 'auto';
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  return (
    <div className="sticky bottom-0 bg-neutral-950/80 backdrop-blur-xl border-t border-neutral-800 py-4 px-4 transition-colors duration-300">
      <div className="max-w-screen-md mx-auto">
        <form onSubmit={handleSubmit} className="relative">
          <div
            className={`
              relative flex items-end gap-3 bg-neutral-900 rounded-full border transition-all duration-300
              backdrop-blur-xl
              ${isFocused 
                ? 'border-blue-400 shadow-[0_0_0_4px_rgba(96,165,250,0.25)]' 
                : 'border-neutral-700 shadow-sm'
              }
            `}
          >
            <textarea
              ref={textareaRef}
              value={message}
              onChange={(e) => setMessage(e.target.value)}
              onKeyDown={handleKeyDown}
              onFocus={() => setIsFocused(true)}
              onBlur={() => setIsFocused(false)}
              placeholder={placeholder}
              rows={1}
              className="
                w-full px-5 py-3 pr-12 rounded-full
                text-[15px] text-neutral-100
                placeholder:text-neutral-500
                focus:outline-none
                resize-none overflow-hidden
                bg-transparent
                transition-colors duration-300
              "
              style={{ maxHeight: '120px' }}
              disabled={isLoading}
            />
            
            <button
              type="submit"
              disabled={!message.trim() || isLoading}
              className={`
                absolute right-2 bottom-2 w-8 h-8 rounded-full flex items-center justify-center
                transition-all duration-200
                ${message.trim() && !isLoading
                  ? 'bg-blue-500 text-white hover:bg-blue-400 shadow-sm'
                  : 'bg-neutral-800 text-neutral-500 cursor-not-allowed'
                }
              `}
            >
              {isLoading ? (
                <svg className="w-4 h-4 animate-spin" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
              ) : (
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 7l5 5m0 0l-5 5m5-5H6" />
                </svg>
              )}
            </button>
          </div>
          
          <p className="text-xs text-neutral-500 mt-2 px-5 text-center transition-colors duration-300">
            AI can make mistakes. Check important info.
          </p>
        </form>
      </div>
    </div>
  );
}
