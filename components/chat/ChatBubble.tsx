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
  sources?: string[];
}

// Helper function to process message and replace citations with markdown code spans
// This allows us to use custom code component to style citations
const processMessageForCitations = (text: string): string => {
  // Pattern untuk mendeteksi citations: (nama_file.pdf) atau (file1.pdf, file2.txt)
  // Pattern: ( ... ) yang berisi nama file dengan ekstensi
  const citationPattern = /\(([^)]+\.(pdf|txt|doc|docx|md|rtf|odt|pages)[^)]*)\)/gi;
  
  // Replace citations with markdown code span that has a special class
  return text.replace(citationPattern, (match, fileName) => {
    // Use markdown code span with special prefix to identify citations
    return `\`CITATION:${fileName}\``;
  });
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

// Book icon component for References
const BookIcon = ({ className }: { className?: string }) => (
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
      d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"
    />
  </svg>
);

export default function ChatBubble({ message, isUser, timestamp, attachment, sources }: ChatBubbleProps) {
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
          <>
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
                  code: ({ children, className }) => {
                    // Check if this is a citation (starts with "CITATION:")
                    const text = String(children);
                    if (text.startsWith('CITATION:')) {
                      const fileName = text.replace('CITATION:', '');
                      return (
                        <span className="text-blue-400 font-medium">({fileName})</span>
                      );
                    }
                    // Regular code block
                    return (
                      <code className="bg-neutral-800 px-1.5 py-0.5 rounded text-[13px] font-mono text-neutral-100 border border-neutral-700">
                        {children}
                      </code>
                    );
                  },
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
                {processMessageForCitations(message)}
              </ReactMarkdown>
            </div>
            
            {/* Sources Section - Only show for AI messages with sources */}
            {sources && sources.length > 0 && (
              <div className="mt-4 pt-4 border-t border-neutral-700">
                <div className="flex items-center gap-2 mb-2">
                  <BookIcon className="w-4 h-4 text-neutral-400" />
                  <span className="text-[12px] font-medium text-neutral-400 uppercase tracking-wide">
                    References
                  </span>
                </div>
                <div className="flex flex-wrap gap-2">
                  {sources.map((source, index) => (
                    <span
                      key={index}
                      className="inline-flex items-center px-2.5 py-1 rounded-lg bg-neutral-800 border border-neutral-700 text-[12px] text-neutral-300 hover:bg-neutral-750 transition-colors"
                    >
                      {source}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
