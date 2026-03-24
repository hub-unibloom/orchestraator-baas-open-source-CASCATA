import React from 'react';
import { useAuth } from '../context/AuthContext';
import Login from '../pages/Login';
import { motion } from 'framer-motion';
import { Shield } from 'lucide-react';
import { VARIANTS } from '../lib/motion';

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, isLoading } = useAuth();

  if (isLoading) {
    return (
      <div className="min-h-screen bg-[#070707] flex items-center justify-center p-6 text-accent-primary">
        <motion.div 
          initial="hidden" animate="visible" variants={VARIANTS.fadeIn}
          className="flex flex-col items-center gap-4"
        >
          <Shield size={48} className="animate-pulse shadow-glow shadow-accent-primary/20" />
          <span className="text-[10px] font-bold uppercase tracking-[0.4em] opacity-50">Sincronizando Identidade...</span>
        </motion.div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Login />;
  }

  return <>{children}</>;
}
