'use client';

import React, { useState, useCallback } from 'react';
import Card from '../ui/Card';
import Button from '../ui/Button';

interface UploadCardProps {
  onFileUpload: (file: File) => void;
  isUploading?: boolean;
}

export default function UploadCard({ onFileUpload, isUploading = false }: UploadCardProps) {
  const [isDragging, setIsDragging] = useState(false);

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
    if (files.length > 0) {
      onFileUpload(files[0]);
    }
  }, [onFileUpload]);

  const handleFileSelect = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    const files = e.target.files;
    if (files && files.length > 0) {
      onFileUpload(files[0]);
    }
  }, [onFileUpload]);

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
            Format yang didukung: PDF, DOCX, TXT
          </p>
        </div>
        
        <input
          type="file"
          id="file-upload"
          className="hidden"
          accept=".pdf,.docx,.txt"
          onChange={handleFileSelect}
          disabled={isUploading}
        />
        
        <Button
          onClick={() => document.getElementById('file-upload')?.click()}
          disabled={isUploading}
          isLoading={isUploading}
        >
          {isUploading ? 'Mengunggah...' : 'Pilih File'}
        </Button>
      </div>
    </Card>
  );
}
