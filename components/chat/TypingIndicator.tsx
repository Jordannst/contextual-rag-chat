import React from 'react';

export default function TypingIndicator() {
  return (
    <div className="flex w-full mb-6 justify-start animate-fade-slide-up">
      <div className="bg-neutral-900 border border-neutral-800 rounded-3xl px-5 py-4 shadow-sm transition-colors duration-300">
        <div className="flex items-center gap-1.5">
          <span className="w-2 h-2 bg-neutral-500 rounded-full animate-blink" style={{ animationDelay: '0ms' }} />
          <span className="w-2 h-2 bg-neutral-500 rounded-full animate-blink" style={{ animationDelay: '150ms' }} />
          <span className="w-2 h-2 bg-neutral-500 rounded-full animate-blink" style={{ animationDelay: '300ms' }} />
        </div>
      </div>
    </div>
  );
}
