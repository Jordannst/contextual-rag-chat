'use client';

import React from 'react';
import ReactMarkdown from 'react-markdown';

interface ChatBubbleProps {
  message: string;
  isUser: boolean;
  timestamp?: Date;
}

export default function ChatBubble({ message, isUser, timestamp }: ChatBubbleProps) {
  return (
    <div
      className={`flex w-full mb-6 animate-fade-slide-up ${
        isUser ? 'justify-end' : 'justify-start'
      }`}
    >
      <div
        className={`
          max-w-[85%] sm:max-w-[75%] rounded-3xl px-5 py-4
          transition-colors duration-300
          ${isUser
            ? 'bg-neutral-800 border border-neutral-700 text-neutral-100'
            : 'bg-neutral-900 border border-neutral-700 text-neutral-100 shadow-sm hover:shadow-md'
          }
        `}
      >
        {isUser ? (
          <p className="text-[15px] leading-relaxed whitespace-pre-wrap break-words">
            {message}
          </p>
        ) : (
          <div className="prose prose-sm max-w-none prose-invert">
            <ReactMarkdown
              components={{
                p: ({ children }) => (
                  <p className="mb-3 last:mb-0 text-[15px] leading-relaxed text-neutral-100">
                    {children}
                  </p>
                ),
                ul: ({ children }) => (
                  <ul className="list-disc list-inside mb-3 space-y-1.5 text-[15px] text-neutral-100">
                    {children}
                  </ul>
                ),
                ol: ({ children }) => (
                  <ol className="list-decimal list-inside mb-3 space-y-1.5 text-[15px] text-neutral-100">
                    {children}
                  </ol>
                ),
                li: ({ children }) => (
                  <li className="ml-2 text-[15px] leading-relaxed">{children}</li>
                ),
                code: ({ children }) => (
                  <code className="bg-neutral-800 px-1.5 py-0.5 rounded text-[13px] font-mono text-neutral-100 border border-neutral-700">
                    {children}
                  </code>
                ),
                pre: ({ children }) => (
                  <pre className="bg-neutral-800 p-3 rounded-xl overflow-x-auto mb-3 text-[13px] font-mono border border-neutral-700">
                    {children}
                  </pre>
                ),
                strong: ({ children }) => (
                  <strong className="font-semibold text-neutral-50">{children}</strong>
                ),
                h1: ({ children }) => (
                  <h1 className="text-xl font-semibold mb-2 text-neutral-50">{children}</h1>
                ),
                h2: ({ children }) => (
                  <h2 className="text-lg font-semibold mb-2 text-neutral-50">{children}</h2>
                ),
                h3: ({ children }) => (
                  <h3 className="text-base font-semibold mb-2 text-neutral-50">{children}</h3>
                ),
              }}
            >
              {message}
            </ReactMarkdown>
          </div>
        )}
      </div>
    </div>
  );
}
