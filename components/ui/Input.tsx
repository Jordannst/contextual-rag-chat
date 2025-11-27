import React from 'react';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export default function Input({
  label,
  error,
  helperText,
  className = '',
  ...props
}: InputProps) {
  return (
    <div className="w-full">
      {label && (
        <label className="block text-sm font-medium text-neutral-300 mb-2 transition-colors duration-300">
          {label}
        </label>
      )}
      <input
        className={`
          w-full px-4 py-3 rounded-xl border transition-all duration-300
          bg-neutral-900
          border-neutral-700
          text-neutral-100
          placeholder-neutral-500
          focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent
          ${error ? 'border-red-500 focus:ring-red-500' : ''}
          ${className}
        `}
        {...props}
      />
      {error && (
        <p className="mt-1 text-sm text-red-400 transition-colors duration-300">{error}</p>
      )}
      {helperText && !error && (
        <p className="mt-1 text-sm text-neutral-400 transition-colors duration-300">{helperText}</p>
      )}
    </div>
  );
}
