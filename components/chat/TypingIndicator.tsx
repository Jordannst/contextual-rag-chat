'use client';

import React from 'react';
import { motion } from 'framer-motion';

export default function TypingIndicator() {
  return (
    <motion.div
      className="flex w-full mb-6 justify-start"
      initial={{ opacity: 0, y: 10 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.3 }}
    >
      <div className="bg-neutral-800/40 backdrop-blur-sm border border-white/5 rounded-3xl px-6 py-5 shadow-md transition-colors duration-300">
        <div className="relative w-32 h-16 flex items-center justify-center" style={{ overflow: 'visible', minWidth: '128px', minHeight: '64px' }}>
          {/* Glowing Circle 1 - Primary (Bright Blue) */}
          <motion.div
            className="absolute rounded-full"
            style={{
              width: '18px',
              height: '18px',
              backgroundColor: '#60a5fa',
              boxShadow: '0 0 25px rgba(96, 165, 250, 1), 0 0 40px rgba(96, 165, 250, 0.9), 0 0 60px rgba(96, 165, 250, 0.7), 0 0 80px rgba(96, 165, 250, 0.5), inset 0 0 10px rgba(96, 165, 250, 0.3)',
              zIndex: 10,
            }}
            animate={{
              scale: [1, 1.6, 1, 1.4, 1],
              rotate: [0, 90, 180, 270, 360],
              x: [-14, 0, 14, 0, -14],
              y: [0, -7, 0, 7, 0],
            }}
            transition={{
              duration: 3.2,
              repeat: Infinity,
              ease: "easeInOut",
            }}
          />
          
          {/* Glowing Circle 2 - Secondary (Bright Purple) */}
          <motion.div
            className="absolute rounded-full"
            style={{
              width: '16px',
              height: '16px',
              backgroundColor: '#c084fc',
              boxShadow: '0 0 25px rgba(192, 132, 252, 1), 0 0 40px rgba(192, 132, 252, 0.9), 0 0 60px rgba(192, 132, 252, 0.7), 0 0 80px rgba(192, 132, 252, 0.5), inset 0 0 10px rgba(192, 132, 252, 0.3)',
              zIndex: 10,
            }}
            animate={{
              scale: [1, 1.7, 1, 1.5, 1],
              rotate: [360, 270, 180, 90, 0],
              x: [14, 0, -14, 0, 14],
              y: [0, 7, 0, -7, 0],
            }}
            transition={{
              duration: 3.8,
              repeat: Infinity,
              ease: "easeInOut",
              delay: 0.6,
            }}
          />
          
          {/* Glowing Circle 3 - Accent (Bright Cyan) */}
          <motion.div
            className="absolute rounded-full"
            style={{
              width: '17px',
              height: '17px',
              backgroundColor: '#22d3ee',
              boxShadow: '0 0 25px rgba(34, 211, 238, 1), 0 0 40px rgba(34, 211, 238, 0.9), 0 0 60px rgba(34, 211, 238, 0.7), 0 0 80px rgba(34, 211, 238, 0.5), inset 0 0 10px rgba(34, 211, 238, 0.3)',
              zIndex: 10,
            }}
            animate={{
              scale: [1, 1.8, 1, 1.55, 1],
              rotate: [0, -90, -180, -270, -360],
              x: [0, 12, 0, -12, 0],
              y: [-12, 0, 12, 0, -12],
            }}
            transition={{
              duration: 4.2,
              repeat: Infinity,
              ease: "easeInOut",
              delay: 1.2,
            }}
          />
        </div>
      </div>
    </motion.div>
  );
}
