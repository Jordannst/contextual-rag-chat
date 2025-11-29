'use client';

import React from 'react';
import ReactMarkdown from 'react-markdown';

interface ChatBubbleProps {
  message: string;
  isUser: boolean;
  timestamp?: Date;
  attachment?: {
    fileName: string;
    fileType: string;
  };
}

// FileText icon component (similar to lucide-react FileText)
const FileTextIcon = ({ className }: { className?: string }) => (
  <svg
    className={className}
    fill="none"
    stroke="currentColor"
    viewBox="0 0 24 24"
    xmlns="http://www.w3.org/2000/svg"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
    />
  </svg>
);

export default function ChatBubble({ message, isUser, timestamp, attachment }: ChatBubbleProps) {
  // If attachment exists, render File Card instead of text
  if (attachment) {
    return (
      <div
        className={`flex w-full mb-6 animate-fade-slide-up ${
          isUser ? 'justify-end' : 'justify-start'
        }`}
      >
        <div
          className={`
            max-w-[85%] sm:max-w-[75%] rounded-2xl px-4 py-3
            transition-colors duration-300
            ${isUser
              ? 'bg-neutral-800 border border-neutral-700'
              : 'bg-neutral-900 border border-neutral-700'
            }
          `}
        >
          {/* File Card */}
          <div className="flex items-center gap-3">
            {/* File Icon */}
            <div className={`
              flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center
              ${isUser
                ? 'bg-neutral-700 text-neutral-300'
                : 'bg-neutral-800 text-neutral-400'
              }
            `}>
              <FileTextIcon className="w-5 h-5" />
            </div>
            
            {/* File Info */}
            <div className="flex-1 min-w-0">
              <p className="text-[15px] font-semibold text-neutral-100 truncate">
                {attachment.fileName}
              </p>
              <p className="text-[12px] text-neutral-400 mt-0.5">
                Uploaded Document
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Regular text message
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
