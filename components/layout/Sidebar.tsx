'use client';

import React, { useState } from 'react';

interface SidebarItem {
  id: string;
  icon: React.ReactNode;
  label: string;
  onClick?: () => void;
  active?: boolean;
}

interface SidebarProps {
  items?: SidebarItem[];
}

export default function Sidebar({ items = [] }: SidebarProps) {
  const [hoveredItem, setHoveredItem] = useState<string | null>(null);

  const defaultItems: SidebarItem[] = [
    {
      id: 'new-chat',
      label: 'New chat',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
        </svg>
      ),
    },
    {
      id: 'history',
      label: 'History',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    },
    {
      id: 'settings',
      label: 'Settings',
      icon: (
        <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      ),
    },
  ];

  const sidebarItems = items.length > 0 ? items : defaultItems;

  return (
    <aside className="fixed left-0 top-0 h-screen w-16 flex flex-col items-center py-4 border-r border-neutral-800 bg-neutral-950/80 backdrop-blur-sm z-40 transition-colors duration-300">
      <div className="flex flex-col items-center gap-2 w-full">
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
                  ? 'bg-neutral-800 text-neutral-100' 
                  : 'text-neutral-400 hover:bg-neutral-800 hover:text-neutral-100'
                }
              `}
            >
              {item.icon}
            </button>
            
            {/* Tooltip on hover */}
            {hoveredItem === item.id && (
              <div className="absolute left-full ml-3 px-3 py-1.5 bg-neutral-800 text-neutral-100 text-xs rounded-lg whitespace-nowrap pointer-events-none animate-fade-slide-up border border-neutral-700 shadow-lg transition-colors duration-300">
                {item.label}
                <div className="absolute left-0 top-1/2 -translate-y-1/2 -translate-x-1 w-2 h-2 bg-neutral-800 rotate-45"></div>
              </div>
            )}
          </div>
        ))}
      </div>
    </aside>
  );
}
