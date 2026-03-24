import React from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { X, Check } from 'lucide-react';
import { Link, useLocation } from 'react-router-dom';

export interface NavItem {
  id: string;
  label: string;
  icon: React.ElementType;
  path: string;
}

interface NavigationModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  items: NavItem[];
}

const NavigationModal: React.FC<NavigationModalProps> = ({ isOpen, onClose, title, items }) => {
  const location = useLocation();

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-[100] flex items-center justify-center p-6 bg-surface-pit/40 backdrop-blur-sm">
        <motion.div
          initial={{ opacity: 0, scale: 0.9, y: 10 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.9, y: 10 }}
          className="w-full max-w-md overflow-hidden glass-panel rounded-3xl border border-white/10 shadow-3xl bg-surface-elevated/90"
        >
          {/* Header */}
          <div className="flex items-center justify-between p-6 border-b border-white/5">
            <h3 className="text-xl font-bold text-white tracking-tight">{title}</h3>
            <button
              onClick={onClose}
              className="p-2 rounded-full bg-white/5 text-content-muted hover:text-white transition-colors"
            >
              <X size={20} />
            </button>
          </div>

          {/* Menu Items */}
          <div className="p-4 space-y-2">
            {items.map((item) => {
              const isActive = location.pathname.includes(item.path);
              const Icon = item.icon;

              return (
                <Link
                  key={item.id}
                  to={item.path}
                  onClick={onClose}
                  className={`
                      flex items-center justify-between p-4 rounded-2xl transition-all group
                      ${isActive ? 'bg-accent-primary/20 border border-accent-primary/20' : 'hover:bg-white/5 border border-transparent'}
                    `}
                >
                  <div className="flex items-center gap-4">
                    <div className={`
                          w-10 h-10 rounded-xl flex items-center justify-center transition-colors
                          ${isActive ? 'bg-accent-primary text-white' : 'bg-white/5 text-content-muted group-hover:text-white'}
                       `}>
                      <Icon size={22} />
                    </div>
                    <div>
                      <p className={`font-bold text-sm ${isActive ? 'text-white' : 'text-content-secondary group-hover:text-white'}`}>
                        {item.label}
                      </p>
                    </div>
                  </div>
                  {isActive && (
                    <div className="text-accent-primary">
                      <Check size={18} />
                    </div>
                  )}
                </Link>
              );
            })}
          </div>

          {/* Footer Decoration */}
          <div className="p-6 bg-black/20 flex justify-center">
            <div className="w-12 h-1 bg-white/10 rounded-full" />
          </div>
        </motion.div>
      </div>
    </AnimatePresence>
  );
};

export default NavigationModal;
