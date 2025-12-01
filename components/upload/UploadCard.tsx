'use client';

import React, { useState, useCallback } from 'react';
import Card from '../ui/Card';
import Button from '../ui/Button';

interface UploadCardProps {
  onFileSelect: (files: File[]) => void;
  onStartProcessing: () => void;
  onRemoveFile: (index: number) => void;
  selectedFiles: File[];
  isUploading?: boolean;
  uploadProgress?: {
    current: number;
    total: number;
    currentFileName?: string;
  };
}

export default function UploadCard({ 
  onFileSelect, 
  onStartProcessing, 
  onRemoveFile,
  selectedFiles,
  isUploading = false, 
  uploadProgress 
}: UploadCardProps) {
  const [isDragging, setIsDragging] = useState(false);

  const allowedExtensions = ['.pdf', '.txt', '.docx'];

  const filterAllowedFiles = (files: File[]) => {
    const allowed: File[] = [];
    for (const file of files) {
      const lowerName = file.name.toLowerCase();
      const hasAllowedExt = allowedExtensions.some(ext => lowerName.endsWith(ext));
      if (hasAllowedExt) {
        allowed.push(file);
      }
    }
    return allowed;
  };

  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  }, []);

  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  }, []);

  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
    
    const files = Array.from(e.dataTransfer.files);
    const allowedFiles = filterAllowedFiles(files);

    if (allowedFiles.length === 0) {
      // Optional: beri feedback sederhana via alert (bisa diganti toast kalau mau)
      alert('Hanya file PDF, DOCX, dan TXT yang didukung.');
      return;
    }

    if (allowedFiles.length > 0) {
      onFileSelect(allowedFiles);
    }
  }, [onFileSelect]);

  const handleFileInputChange = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      const fileArray = Array.from(files);
      const allowedFiles = filterAllowedFiles(fileArray);

      if (allowedFiles.length === 0) {
        alert('Hanya file PDF, DOCX, dan TXT yang didukung.');
      } else {
        onFileSelect(allowedFiles);
      }
    }
    // Reset input so same file can be selected again
    e.target.value = '';
  }, [onFileSelect]);

  return (
    <Card
      hover={!isUploading}
      className={`
        p-8 border-2 border-dashed transition-all duration-300
        ${isDragging
          ? 'border-blue-500 bg-blue-500/10 scale-105'
          : 'border-neutral-700 hover:border-blue-500'
        }
        ${isUploading ? 'opacity-50 cursor-not-allowed' : ''}
      `}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      <div className="flex flex-col items-center justify-center text-center space-y-4">
        <div className="w-16 h-16 rounded-full bg-gradient-to-br from-blue-500/20 to-indigo-500/20 flex items-center justify-center">
          <svg
            className="w-8 h-8 text-blue-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
            />
          </svg>
        </div>
        
        <div>
          <h3 className="text-lg font-semibold text-neutral-100 mb-2 transition-colors duration-300">
            Upload Dokumen
          </h3>
          <p className="text-sm text-neutral-400 mb-4 transition-colors duration-300">
            Seret dan lepas file di sini, atau klik untuk memilih
          </p>
          <p className="text-xs text-neutral-500 transition-colors duration-300">
            Format yang didukung: PDF, DOCX, TXT (bisa pilih beberapa file sekaligus)
          </p>
        </div>

        {/* Selected Files Preview */}
        {selectedFiles.length > 0 && !isUploading && (
          <div className="w-full mb-4 space-y-2">
            <p className="text-sm font-medium text-neutral-300 mb-2">
              File yang dipilih ({selectedFiles.length}):
            </p>
            <div className="space-y-2 max-h-48 overflow-y-auto">
              {selectedFiles.map((file, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between p-3 bg-neutral-800/50 rounded-lg border border-neutral-700"
                >
                  <div className="flex items-center gap-3 flex-1 min-w-0">
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
                    <span className="text-sm text-neutral-100 truncate flex-1" title={file.name}>
                      {file.name}
                    </span>
                    <span className="text-xs text-neutral-500">
                      {(file.size / 1024).toFixed(1)} KB
                    </span>
                  </div>
                  <button
                    onClick={() => onRemoveFile(index)}
                    className="ml-3 p-1.5 rounded-lg hover:bg-red-900/20 text-neutral-400 hover:text-red-400 transition-colors"
                    title="Hapus file"
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
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Upload Progress Indicator */}
        {isUploading && uploadProgress && (
          <div className="w-full mb-4">
            <div className="flex items-center justify-between text-sm text-neutral-400 mb-2">
              <span>
                Mengunggah {uploadProgress.current} dari {uploadProgress.total} file...
              </span>
              <span>{Math.round((uploadProgress.current / uploadProgress.total) * 100)}%</span>
            </div>
            {uploadProgress.currentFileName && (
              <p className="text-xs text-neutral-500 mb-2 truncate">
                {uploadProgress.currentFileName}
              </p>
            )}
            <div className="w-full bg-neutral-800 rounded-full h-2">
              <div
                className="bg-blue-500 h-2 rounded-full transition-all duration-300"
                style={{ width: `${(uploadProgress.current / uploadProgress.total) * 100}%` }}
              />
            </div>
          </div>
        )}
        
        <input
          type="file"
          id="file-upload"
          className="hidden"
          accept=".pdf,.docx,.txt"
          multiple
          onChange={handleFileInputChange}
          disabled={isUploading}
        />
        
        <div className="flex flex-col gap-2 w-full">
          <Button
            onClick={() => document.getElementById('file-upload')?.click()}
            disabled={isUploading}
            isLoading={false}
            className="w-full"
          >
            {isUploading ? 'Mengunggah...' : 'Pilih File'}
          </Button>
          
          {/* Start Processing Button - Only show if files are selected */}
          {selectedFiles.length > 0 && !isUploading && (
            <Button
              onClick={onStartProcessing}
              disabled={isUploading}
              isLoading={false}
              className="w-full bg-green-600 hover:bg-green-700 text-white"
            >
              Mulai Proses & Chat ({selectedFiles.length} file)
            </Button>
          )}
        </div>
      </div>
    </Card>
  );
}
