import React from 'react';

interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  className?: string;
  hover?: boolean;
  onClick?: () => void;
}

export default function Card({ 
  children, 
  className = '', 
  hover = false,
  onClick,
  ...props
}: CardProps) {
  const baseStyles = 'rounded-xl bg-neutral-900 border border-neutral-800 shadow-sm transition-colors duration-300';
  const hoverStyles = hover ? 'hover:shadow-md hover:scale-[1.02] cursor-pointer' : '';
  
  return (
    <div
      className={`${baseStyles} ${hoverStyles} ${className}`}
      onClick={onClick}
      {...props}
    >
      {children}
    </div>
  );
}
