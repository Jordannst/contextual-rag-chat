'use client';

import React, { useState } from 'react';

interface SidebarItem {
  id: string;
  icon: React.ReactNode;
  label: string;
  onClick?: () => void;
  active?: boolean;
}

interface ChatSession {
  id: number;
  title: string;
  created_at: string;
}

interface SidebarProps {
  items?: SidebarItem[];
  sessions?: ChatSession[];
  currentSessionId?: number | null;
  onSelectSession?: (sessionId: number) => void;
  onDeleteSession?: (sessionId: number) => void;
  onNewChat?: () => void;
}

export default function Sidebar({ 
  items = [], 
  sessions = [],
  currentSessionId = null,
  onSelectSession,
  onDeleteSession,
  onNewChat
}: SidebarProps) {
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);
  const [hoveredSessionId, setHoveredSessionId] = useState<number | null>(null);
  const [isExpanded, setIsExpanded] = useState(false);

  const defaultItems: SidebarItem[] = [
    {
      id: 'new-chat',
      label: 'New chat',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
      ),
      onClick: onNewChat,
    },
  ];

  const sidebarItems = items.length > 0 ? items : defaultItems;

  return (
    <aside 
      className={`fixed left-0 top-0 h-screen flex flex-col border-r border-white/5 bg-neutral-900/60 backdrop-blur-lg z-40 transition-all duration-300 ${
        isExpanded ? 'w-64' : 'w-16'
      }`}
      onMouseEnter={() => setIsExpanded(true)}
      onMouseLeave={() => setIsExpanded(false)}
    >
      {/* Top Section - Action Buttons */}
      <div className="flex flex-col items-center gap-2 w-full py-4 px-2">
        {sidebarItems.map((item) => (
          <div key={item.id} className="relative w-full flex justify-center">
            <button
              onClick={item.onClick}
              onMouseEnter={() => setHoveredItem(item.id)}
              onMouseLeave={() => setHoveredItem(null)}
              className={`
                w-10 h-10 rounded-full flex items-center justify-center
                transition-colors duration-300
                ${item.active 
                  ? 'bg-neutral-800/60 backdrop-blur-sm text-neutral-100 border border-white/5' 
                  : 'text-neutral-400 hover:bg-neutral-800/60 hover:backdrop-blur-sm hover:text-neutral-100 border border-transparent hover:border-white/5'
                }
              `}
            >
              {item.icon}
            </button>
            
            {/* Tooltip on hover (when collapsed) */}
            {!isExpanded && hoveredItem === item.id && (
              <div className="absolute left-full ml-3 px-3 py-1.5 bg-neutral-900/80 backdrop-blur-md text-neutral-100 text-xs rounded-lg whitespace-nowrap pointer-events-none animate-fade-slide-up border border-white/10 shadow-lg transition-colors duration-300">
                {item.label}
                <div className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1 w-2 h-2 bg-neutral-900/80 backdrop-blur-md border-l border-b border-white/10 rotate-45"></div>
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Divider */}
      <div className="border-t border-white/5 my-2"></div>

      {/* Chat History Section */}
      <div className="flex-1 overflow-y-auto px-2">
        {isExpanded && (
          <div className="px-2 py-2">
            <h3 className="text-xs font-semibold text-neutral-500 uppercase tracking-wider mb-2">
              Recent Chats
            </h3>
          </div>
        )}
        
        <div className="space-y-1">
          {sessions.map((session) => (
            <div
              key={session.id}
              className={`
                group relative flex items-center gap-2 px-2 py-2 rounded-lg
                transition-colors duration-200
                ${currentSessionId === session.id
                  ? 'bg-neutral-800/60 backdrop-blur-sm text-neutral-100 border border-white/5'
                  : 'text-neutral-400 hover:bg-neutral-800/50 hover:backdrop-blur-sm hover:text-neutral-100 border border-transparent hover:border-white/5'
                }
              `}
              onMouseEnter={() => setHoveredSessionId(session.id)}
              onMouseLeave={() => setHoveredSessionId(null)}
            >
              {/* Chat Icon */}
              <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-neutral-700 flex items-center justify-center">
                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 12h.01M12 12h.01M16 12h.01M21 12c0 4.418-4.03 8-9 8a9.863 9.863 0 01-4.255-.949L3 20l1.395-3.72C3.512 15.042 3 13.574 3 12c0-4.418 4.03-8 9-8s9 3.582 9 8z" />
                </svg>
              </div>

              {/* Session Title (only when expanded) */}
              {isExpanded && (
                <div 
                  className="flex-1 min-w-0 cursor-pointer"
                  onClick={() => onSelectSession?.(session.id)}
                >
                  <div className="text-sm truncate">{session.title}</div>
                </div>
              )}

              {/* Delete Button (only when expanded and hovered) */}
              {isExpanded && hoveredSessionId === session.id && onDeleteSession && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    onDeleteSession(session.id);
                  }}
                  className="flex-shrink-0 p-1.5 rounded hover:bg-red-900/20 text-neutral-400 hover:text-red-400 transition-colors"
                  title="Delete chat"
                >
                  <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                  </svg>
                </button>
              )}

              {/* Click handler for collapsed state */}
              {!isExpanded && (
                <div 
                  className="absolute inset-0 cursor-pointer"
                  onClick={() => onSelectSession?.(session.id)}
                />
              )}
            </div>
          ))}
        </div>

        {sessions.length === 0 && isExpanded && (
          <div className="px-2 py-4 text-center text-neutral-500 text-sm">
            No chat history yet
          </div>
        )}
      </div>
    </aside>
  );
}
