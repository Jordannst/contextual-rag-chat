'use client';

import React from 'react';

export default function AmbientBackground() {
  return (
    <>
      {/* Base dark background layer */}
      <div className="fixed inset-0 bg-neutral-950 -z-10" />
      
      {/* Aurora Blobs Layer */}
      <div className="fixed inset-0 z-0 overflow-hidden pointer-events-none">
        {/* Blob 1: Top Left - Blue */}
        <div 
          className="absolute w-[800px] h-[800px] rounded-full blur-[120px] opacity-70 animate-float-1"
          style={{
            background: 'radial-gradient(circle, rgba(59, 130, 246, 0.4) 0%, rgba(59, 130, 246, 0) 70%)',
            top: '-20%',
            left: '-20%',
          }}
        />
        
        {/* Blob 2: Bottom Right - Purple */}
        <div 
          className="absolute w-[900px] h-[900px] rounded-full blur-[120px] opacity-60 animate-float-2"
          style={{
            background: 'radial-gradient(circle, rgba(168, 85, 247, 0.4) 0%, rgba(168, 85, 247, 0) 70%)',
            bottom: '-20%',
            right: '-20%',
          }}
        />
        
        {/* Blob 3: Center/Moving - Cyan */}
        <div 
          className="absolute w-[700px] h-[700px] rounded-full blur-[120px] opacity-50 animate-float-3"
          style={{
            background: 'radial-gradient(circle, rgba(6, 182, 212, 0.3) 0%, rgba(6, 182, 212, 0) 70%)',
            top: 'calc(50% - 350px)',
            left: 'calc(50% - 350px)',
          }}
        />
        
        {/* Blob 4: Bottom Center - Additional coverage for input area */}
        <div 
          className="absolute w-[600px] h-[600px] rounded-full blur-[100px] opacity-40 animate-float-2"
          style={{
            background: 'radial-gradient(circle, rgba(139, 92, 246, 0.3) 0%, rgba(139, 92, 246, 0) 70%)',
            bottom: '-10%',
            left: 'calc(50% - 300px)',
          }}
        />
      </div>

      {/* Noise Texture Overlay */}
      <div 
        className="fixed inset-0 z-0 opacity-[0.04] pointer-events-none"
        style={{
          backgroundImage: `url("data:image/svg+xml,%3Csvg viewBox='0 0 400 400' xmlns='http://www.w3.org/2000/svg'%3E%3Cfilter id='noiseFilter'%3E%3CfeTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='4' stitchTiles='stitch'/%3E%3C/filter%3E%3Crect width='100%25' height='100%25' filter='url(%23noiseFilter)'/%3E%3C/svg%3E")`,
          backgroundSize: '200px 200px',
        }}
      />
    </>
  );
}

