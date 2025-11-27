'use client';

import React, { useState } from 'react';
import Card from '../ui/Card';

interface ChatHistory {
  id: string;
  title: string;
  timestamp: Date;
}

interface ChatSidebarProps {
  isOpen: boolean;
  onClose: () => void;
  chatHistory: ChatHistory[];
  onSelectChat: (id: string) => void;
  currentChatId?: string;
}

export default function ChatSidebar({
  isOpen,
  onClose,
  chatHistory,
  onSelectChat,
  currentChatId,
}: ChatSidebarProps) {
  return (
    <>
      {/* Mobile overlay */}
      {isOpen && (
        <div
          className="fixed inset-0 bg-black/50 z-40 lg:hidden transition-opacity duration-300"
          onClick={onClose}
        />
      )}
      
      {/* Sidebar */}
      <aside
        className={`
          fixed top-0 left-0 h-screen w-64 bg-neutral-950
          border-r border-neutral-800 z-50
          transform transition-transform duration-300 ease-in-out
          lg:translate-x-0 lg:static lg:h-auto
          ${isOpen ? 'translate-x-0' : '-translate-x-full'}
        `}
      >
        <div className="h-full flex flex-col">
          <div className="p-4 border-b border-neutral-800">
            <div className="flex items-center justify-between">
              <h2 className="font-semibold text-neutral-100 transition-colors duration-300">Riwayat Chat</h2>
              <button
                onClick={onClose}
                className="lg:hidden p-1 rounded-lg hover:bg-neutral-800 text-neutral-400 hover:text-neutral-100 transition-colors duration-300"
              >
                <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                </svg>
              </button>
            </div>
          </div>
          
          <div className="flex-1 overflow-y-auto p-4 space-y-2">
            {chatHistory.length === 0 ? (
              <p className="text-sm text-neutral-400 text-center py-8 transition-colors duration-300">
                Belum ada riwayat chat
              </p>
            ) : (
              chatHistory.map((chat) => (
                <Card
                  key={chat.id}
                  hover
                  onClick={() => onSelectChat(chat.id)}
                  className={`p-3 transition-colors duration-300 ${
                    currentChatId === chat.id
                      ? 'ring-2 ring-blue-500 bg-blue-500/10'
                      : ''
                  }`}
                >
                  <h3 className="font-medium text-sm text-neutral-100 truncate transition-colors duration-300">
                    {chat.title}
                  </h3>
                  <p className="text-xs text-neutral-400 mt-1 transition-colors duration-300">
                    {chat.timestamp.toLocaleDateString('id-ID', {
                      day: 'numeric',
                      month: 'short',
                      hour: '2-digit',
                      minute: '2-digit',
                    })}
                  </p>
                </Card>
              ))
            )}
          </div>
        </div>
      </aside>
    </>
  );
}
