'use client';

import React, { useRef, useEffect } from 'react';

interface ChatContainerProps {
  children: React.ReactNode;
}

export default function ChatContainer({ children }: ChatContainerProps) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (container) {
      container.scrollTop = container.scrollHeight;
    }
  }, [children]);

  return (
    <div
      ref={containerRef}
      className="flex-1 overflow-y-auto px-4 py-10 bg-neutral-950 transition-colors duration-300"
      style={{ scrollBehavior: 'smooth' }}
    >
      <div className="max-w-screen-md mx-auto">
        {children}
      </div>
    </div>
  );
}
