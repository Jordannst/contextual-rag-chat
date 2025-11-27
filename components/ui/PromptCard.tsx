'use client';

import React from 'react';

interface PromptCardProps {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  onClick?: () => void;
}

export default function PromptCard({ title, description, icon, onClick }: PromptCardProps) {
  return (
    <button
      onClick={onClick}
      className="
        w-full text-left p-4 rounded-2xl border border-neutral-800
        bg-neutral-900
        hover:border-neutral-700
        hover:shadow-md
        transition-all duration-300 hover:-translate-y-0.5 hover:scale-[1.01]
        group
      "
    >
      <div className="flex items-start gap-3">
        {icon && (
          <div className="flex-shrink-0 w-8 h-8 rounded-lg bg-neutral-800 flex items-center justify-center group-hover:bg-neutral-700 transition-colors duration-300">
            {icon}
          </div>
        )}
        <div className="flex-1 min-w-0">
          <h3 className="text-[15px] font-medium text-neutral-100 mb-1 transition-colors duration-300">
            {title}
          </h3>
          {description && (
            <p className="text-[13px] text-neutral-400 line-clamp-2 transition-colors duration-300">
              {description}
            </p>
          )}
        </div>
      </div>
    </button>
  );
}
