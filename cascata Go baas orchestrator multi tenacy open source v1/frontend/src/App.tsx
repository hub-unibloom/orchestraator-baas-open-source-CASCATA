import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Layers, Database, Zap, Shield, HardDrive, Smartphone,
  Settings, LogOut, Menu, Rocket, Terminal, Search, Bell, Activity
} from 'lucide-react';

import Dashboard from './pages/Dashboard';
import ProjectDashboard from './pages/ProjectDashboard';
import { useLayoutStore } from './hooks/useLayoutStore';
import { MOTION, VARIANTS } from './lib/motion';
import { AuthProvider, useAuth } from './context/AuthContext';
import { AuthGuard } from './components/AuthGuard';

import './styles/tokens.css'; 
import './i18n';             

type ViewContext = 'SYSTEM_DASH' | 'PROJECT_DASH';

export default function App() {
  const [viewContext, setViewContext] = useState<ViewContext>('SYSTEM_DASH');
  const [activeProject, setActiveProject] = useState<string | null>(null);

  const switchToProject = (slug: string) => {
    setActiveProject(slug);
    setViewContext('PROJECT_DASH');
  };

  const switchToSystem = () => {
    setViewContext('SYSTEM_DASH');
    setActiveProject(null);
  };

  return (
    <AuthProvider>
      <AuthGuard>
        {viewContext === 'SYSTEM_DASH' ? (
          <LayoutOrchestrator onEnterProject={switchToProject} />
        ) : (
          <ProjectDashboard 
            slug={activeProject!} 
            onExit={switchToSystem} 
          />
        )}
      </AuthGuard>
    </AuthProvider>
  );
}

interface LayoutProps {
  onEnterProject: (slug: string) => void;
}

function LayoutOrchestrator({ onEnterProject }: LayoutProps) {
  const { t } = useTranslation('common');
  const { preferences, updatePreference } = useLayoutStore();
  const { logout } = useAuth();
  const { sidebarCollapsed } = preferences;
  
  const setSidebarCollapsed = (val: boolean) => updatePreference('sidebarCollapsed', val);
  const isExpanded = !sidebarCollapsed;

  return (
    <div className="flex h-screen bg-surface-base text-content-primary overflow-hidden font-sans selection:bg-accent-primary/30">
      <motion.aside
        initial={false}
        animate={{ width: isExpanded ? preferences.sidebarWidth : 72 }}
        transition={MOTION.spring}
        className="relative flex flex-col h-full bg-surface-pit border-r border-border-subtle z-50 shrink-0 shadow-lg"
      >
        <div className="h-16 flex items-center px-4 border-b border-border-subtle shrink-0">
          <button 
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="p-1.5 rounded-xl text-accent-primary hover:text-accent-dim hover:bg-surface-raised transition-colors flex items-center justify-center group"
          >
            {isExpanded ? <Layers size={22} /> : <Menu size={22} />}
          </button>
          
          <AnimatePresence initial={false}>
            {isExpanded && (
              <motion.div initial={{ opacity: 0, width: 0 }} animate={{ opacity: 1, width: 'auto', marginLeft: 12 }} exit={{ opacity: 0, width: 0 }} className="overflow-hidden whitespace-nowrap">
                <span className="font-bold text-[15px] tracking-wide text-white block leading-tight">CASCATA</span>
                <span className="text-[9px] font-bold text-accent-primary uppercase tracking-[0.2em] block">Studio v1.0</span>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        <nav className="flex-1 overflow-y-auto custom-scrollbar py-4 px-3 flex flex-col gap-1.5">
           <SidebarItem icon={Activity} label="Início" active expanded={isExpanded} />
           <SidebarItem icon={Database} label="Infraestrutura" expanded={isExpanded} />
           <SidebarItem icon={Shield} label="Segurança" expanded={isExpanded} />
        </nav>

        <div className="p-3 border-t border-border-subtle shrink-0">
          <button onClick={logout} className="w-full mt-2 flex items-center rounded-xl p-3 text-accent-danger hover:bg-accent-danger/10 transition-all">
            <LogOut size={18} />
            {isExpanded && <span className="ml-3 text-[13px] font-medium tracking-wide">Encerrar</span>}
          </button>
        </div>
      </motion.aside>

      <main className="flex-1 relative flex flex-col min-w-0 bg-surface-base overflow-hidden">
        <Dashboard onEnterProject={onEnterProject} />
      </main>
    </div>
  );
}

function SidebarItem({ icon: Icon, label, to, active, expanded }: any) {
  return (
    <motion.div whileTap={{ scale: 0.98 }}>
      <button className={`w-full flex items-center rounded-xl p-3 transition-all ${active ? 'bg-accent-primary/10 text-accent-primary' : 'text-content-muted hover:bg-white/5 hover:text-white'}`}>
        <Icon size={18} />
        {expanded && <span className="ml-3 text-sm font-medium">{label}</span>}
      </button>
    </motion.div>
  );
}
