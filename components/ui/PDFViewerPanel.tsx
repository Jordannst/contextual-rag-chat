'use client';

import React, { useEffect, useState } from 'react';

interface PDFViewerPanelProps {
  fileName: string | null;
  onClose: () => void;
}

export default function PDFViewerPanel({ fileName, onClose }: PDFViewerPanelProps) {
  const [isCollapsed, setIsCollapsed] = useState(false);

  // Close panel on Escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && fileName) {
        onClose();
      }
    };

    if (fileName) {
      document.addEventListener('keydown', handleEscape);
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
    };
  }, [fileName, onClose]);

  if (!fileName) {
    return null;
  }

  // Construct PDF URL
  const pdfUrl = `http://localhost:5000/api/files/${encodeURIComponent(fileName)}`;

  return (
    <>
      {/* Mobile: Full-screen overlay (Modal behavior) */}
      <div className="lg:hidden fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm animate-fade-in">
        <div
          className="relative w-full h-full max-w-7xl max-h-[90vh] m-4 bg-neutral-900/90 backdrop-blur-lg rounded-2xl shadow-2xl flex flex-col border border-white/10"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-lg bg-red-600/20 flex items-center justify-center">
                <svg
                  className="w-5 h-5 text-red-400"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <div>
                <h2 className="text-lg font-semibold text-neutral-100 truncate max-w-md">
                  {fileName}
                </h2>
                <p className="text-sm text-neutral-400">PDF Document</p>
              </div>
            </div>

            {/* Close Button */}
            <button
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-neutral-800/60 backdrop-blur-sm text-neutral-400 hover:text-neutral-100 transition-colors border border-transparent hover:border-white/10"
              aria-label="Close panel"
            >
              <svg
                className="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>

          {/* PDF Viewer */}
          <div className="flex-1 overflow-hidden">
            <iframe
              src={pdfUrl}
              className="w-full h-full border-0"
              title={fileName}
              allow="fullscreen"
            />
          </div>

          {/* Footer */}
          <div className="px-6 py-3 border-t border-white/10 bg-neutral-800/40 backdrop-blur-sm">
            <div className="flex items-center justify-between text-sm text-neutral-400">
              <span>Press ESC to close</span>
              <a
                href={pdfUrl}
                download={fileName}
                className="text-blue-400 hover:text-blue-300 transition-colors flex items-center gap-1"
              >
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                  />
                </svg>
                Download
              </a>
            </div>
          </div>
        </div>
      </div>

      {/* Desktop: Split-screen panel */}
      <div className="hidden lg:flex flex-col h-full w-full border-l border-white/10 bg-neutral-900/60 backdrop-blur-lg">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-white/10 flex-shrink-0">
          <div className="flex items-center gap-3 flex-1 min-w-0">
            <div className="w-10 h-10 rounded-lg bg-red-600/20 flex items-center justify-center flex-shrink-0">
              <svg
                className="w-5 h-5 text-red-400"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z"
                />
              </svg>
            </div>
            <div className="min-w-0 flex-1">
              <h2 className="text-lg font-semibold text-neutral-100 truncate">
                {fileName}
              </h2>
              <p className="text-sm text-neutral-400">PDF Document</p>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-2 flex-shrink-0">
            <button
              onClick={() => setIsCollapsed(!isCollapsed)}
              className="p-2 rounded-lg hover:bg-neutral-800/60 backdrop-blur-sm text-neutral-400 hover:text-neutral-100 transition-colors border border-transparent hover:border-white/10"
              aria-label={isCollapsed ? 'Expand panel' : 'Collapse panel'}
              title={isCollapsed ? 'Expand' : 'Collapse'}
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                {isCollapsed ? (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 5l7 7-7 7"
                  />
                ) : (
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M15 19l-7-7 7-7"
                  />
                )}
              </svg>
            </button>
            <button
              onClick={onClose}
              className="p-2 rounded-lg hover:bg-neutral-800/60 backdrop-blur-sm text-neutral-400 hover:text-neutral-100 transition-colors border border-transparent hover:border-white/10"
              aria-label="Close panel"
              title="Close"
            >
              <svg
                className="w-5 h-5"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>
        </div>

        {/* PDF Viewer - Collapsible */}
        {!isCollapsed && (
          <div className="flex-1 overflow-hidden">
            <iframe
              src={pdfUrl}
              className="w-full h-full border-0"
              title={fileName}
              allow="fullscreen"
            />
          </div>
        )}

        {/* Footer */}
        {!isCollapsed && (
          <div className="px-6 py-3 border-t border-white/10 bg-neutral-800/40 backdrop-blur-sm flex-shrink-0">
            <div className="flex items-center justify-between text-sm text-neutral-400">
              <span>Press ESC to close</span>
              <a
                href={pdfUrl}
                download={fileName}
                className="text-blue-400 hover:text-blue-300 transition-colors flex items-center gap-1"
              >
                <svg
                  className="w-4 h-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                  />
                </svg>
                Download
              </a>
            </div>
          </div>
        )}
      </div>
    </>
  );
}

