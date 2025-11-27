import React from 'react';

interface LoaderProps {
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

export default function Loader({ size = 'md', className = '' }: LoaderProps) {
  const sizes = {
    sm: 'w-1 h-1',
    md: 'w-2 h-2',
    lg: 'w-3 h-3',
  };
  
  return (
    <div className={`flex items-center gap-1.5 ${className}`}>
      <span
        className={`${sizes[size]} bg-blue-500 rounded-full animate-blink`}
        style={{ animationDelay: '0ms' }}
      />
      <span
        className={`${sizes[size]} bg-blue-500 rounded-full animate-blink`}
        style={{ animationDelay: '150ms' }}
      />
      <span
        className={`${sizes[size]} bg-blue-500 rounded-full animate-blink`}
        style={{ animationDelay: '300ms' }}
      />
    </div>
  );
}
