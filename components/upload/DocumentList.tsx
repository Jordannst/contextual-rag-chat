'use client';

import React, { useState, useEffect } from 'react';

interface DocumentListProps {
  onDocumentDeleted?: () => void;
  refreshTrigger?: number;
}

export default function DocumentList({ onDocumentDeleted, refreshTrigger }: DocumentListProps) {
  const [documents, setDocuments] = useState<string[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deletingFile, setDeletingFile] = useState<string | null>(null);
  const [isSyncing, setIsSyncing] = useState(false);
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(null);

  // Fetch documents on mount
  useEffect(() => {
    fetchDocuments();
  }, []);

  // Refresh when refreshTrigger changes (triggered after upload completes)
  useEffect(() => {
    if (refreshTrigger !== undefined && refreshTrigger > 0) {
      fetchDocuments();
    }
  }, [refreshTrigger]);

  const fetchDocuments = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const response = await fetch('http://localhost:5000/api/documents');
      
      if (!response.ok) {
        throw new Error('Failed to fetch documents');
      }

      const data = await response.json();
      setDocuments(data.documents || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load documents');
      console.error('Error fetching documents:', err);
    } finally {
      setIsLoading(false);
    }
  };

  // Show toast notification
  const showToast = (message: string, type: 'success' | 'error' = 'success') => {
    setToast({ message, type });
    setTimeout(() => {
      setToast(null);
    }, 4000); // Auto-hide after 4 seconds
  };

  const handleSync = async () => {
    try {
      setIsSyncing(true);
      setError(null);

      const response = await fetch('http://localhost:5000/api/documents/sync', {
        method: 'POST',
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to sync documents');
      }

      const data = await response.json();
      const deletedCount = data.deleted_count || 0;
      const addedCount = data.added_count || 0;

      // Show success toast with both deleted and added counts
      let message = 'Sinkronisasi selesai.';
      const parts: string[] = [];
      
      if (deletedCount > 0) {
        parts.push(`${deletedCount} dokumen hantu dihapus`);
      }
      
      if (addedCount > 0) {
        parts.push(`${addedCount} file baru diimpor`);
      }
      
      if (parts.length > 0) {
        message += ' ' + parts.join(', ') + '.';
      } else {
        message += ' Tidak ada perubahan.';
      }
      
      showToast(message, 'success');

      // Refresh the list
      await fetchDocuments();
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to sync documents';
      setError(errorMessage);
      showToast(errorMessage, 'error');
      console.error('Error syncing documents:', err);
    } finally {
      setIsSyncing(false);
    }
  };

  const handleDelete = async (fileName: string) => {
    // Confirm deletion
    if (!confirm(`Are you sure you want to delete "${fileName}"? This will remove all chunks associated with this file.`)) {
      return;
    }

    try {
      setDeletingFile(fileName);
      setError(null);

      // URL encode the filename to handle special characters
      const encodedFileName = encodeURIComponent(fileName);
      const response = await fetch(`http://localhost:5000/api/documents/${encodedFileName}`, {
        method: 'DELETE',
      });

      if (!response.ok) {
        const errorData = await response.json();
        throw new Error(errorData.message || 'Failed to delete document');
      }

      // Refresh the list
      await fetchDocuments();
      
      // Notify parent component if callback provided
      if (onDocumentDeleted) {
        onDocumentDeleted();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to delete document');
      console.error('Error deleting document:', err);
    } finally {
      setDeletingFile(null);
    }
  };

  if (isLoading) {
    return (
      <div className="w-full max-w-2xl mx-auto mt-8 p-6 bg-neutral-900 border border-neutral-800 rounded-2xl">
        <div className="flex items-center gap-2 text-neutral-400">
          <div className="w-4 h-4 border-2 border-neutral-600 border-t-neutral-400 rounded-full animate-spin"></div>
          <span className="text-sm">Loading documents...</span>
        </div>
      </div>
    );
  }

  if (error && documents.length === 0) {
    return (
      <div className="w-full max-w-2xl mx-auto mt-8 p-6 bg-neutral-900 border border-neutral-800 rounded-2xl">
        <div className="text-red-400 text-sm">{error}</div>
        <button
          onClick={fetchDocuments}
          className="mt-3 px-4 py-2 bg-neutral-800 hover:bg-neutral-700 text-neutral-100 rounded-lg text-sm transition-colors"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <>
      {/* Toast Notification */}
      {toast && (
        <div className="fixed top-4 right-4 z-50 animate-fade-slide-up">
          <div
            className={`
              px-4 py-3 rounded-lg shadow-lg border flex items-center gap-3
              ${toast.type === 'success'
                ? 'bg-green-900/90 border-green-700 text-green-100'
                : 'bg-red-900/90 border-red-700 text-red-100'
              }
            `}
          >
            {toast.type === 'success' ? (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
              </svg>
            ) : (
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            )}
            <span className="text-sm font-medium">{toast.message}</span>
            <button
              onClick={() => setToast(null)}
              className="ml-2 text-current opacity-70 hover:opacity-100"
            >
              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}

      <div className="w-full max-w-2xl mx-auto mt-8 p-6 bg-neutral-900 border border-neutral-800 rounded-2xl">
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold text-neutral-100">Uploaded Documents</h2>
          <div className="flex items-center gap-2">
            <button
              onClick={handleSync}
              disabled={isSyncing}
              className={`
                px-3 py-1.5 text-xs rounded-lg transition-colors flex items-center gap-1.5
                ${isSyncing
                  ? 'bg-neutral-700 text-neutral-400 cursor-not-allowed'
                  : 'bg-blue-900/50 hover:bg-blue-900/70 text-blue-300 border border-blue-700/50'
                }
              `}
              title="Sync database with physical files"
            >
              {isSyncing ? (
                <>
                  <div className="w-3 h-3 border-2 border-neutral-600 border-t-neutral-400 rounded-full animate-spin"></div>
                  <span>Syncing...</span>
                </>
              ) : (
                <>
                  <svg className="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                  </svg>
                  <span>Sync</span>
                </>
              )}
            </button>
            <button
              onClick={fetchDocuments}
              className="px-3 py-1.5 text-xs bg-neutral-800 hover:bg-neutral-700 text-neutral-300 rounded-lg transition-colors"
              title="Refresh"
            >
              Refresh
            </button>
          </div>
        </div>

      {error && (
        <div className="mb-4 p-3 bg-red-900/20 border border-red-800/50 rounded-lg text-red-400 text-sm">
          {error}
        </div>
      )}

      {documents.length === 0 ? (
        <div className="text-center py-8 text-neutral-400 text-sm">
          No documents uploaded yet. Upload a file to get started.
        </div>
      ) : (
        <div className="space-y-2">
          {documents.map((fileName, index) => (
            <div
              key={index}
              className="flex items-center justify-between p-3 bg-neutral-800/50 hover:bg-neutral-800 rounded-lg transition-colors group"
            >
              <div className="flex items-center gap-3 flex-1 min-w-0">
                {/* File Icon */}
                <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-neutral-700 flex items-center justify-center">
                  <svg
                    className="w-4 h-4 text-neutral-400"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                    />
                  </svg>
                </div>
                
                {/* File Name */}
                <span className="text-sm text-neutral-100 truncate flex-1" title={fileName}>
                  {fileName}
                </span>
              </div>

              {/* Delete Button */}
              <button
                onClick={() => handleDelete(fileName)}
                disabled={deletingFile === fileName}
                className={`
                  flex-shrink-0 ml-3 p-2 rounded-lg transition-colors
                  ${deletingFile === fileName
                    ? 'opacity-50 cursor-not-allowed'
                    : 'hover:bg-red-900/20 text-neutral-400 hover:text-red-400'
                  }
                `}
                title="Delete document"
              >
                {deletingFile === fileName ? (
                  <div className="w-4 h-4 border-2 border-neutral-600 border-t-neutral-400 rounded-full animate-spin"></div>
                ) : (
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
                      d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                    />
                  </svg>
                )}
              </button>
            </div>
          ))}
        </div>
      )}

      {documents.length > 0 && (
        <div className="mt-4 text-xs text-neutral-500 text-center">
          {documents.length} document{documents.length !== 1 ? 's' : ''} in database
        </div>
      )}
      </div>
    </>
  );
}

