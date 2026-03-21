import React, { useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { motion, AnimatePresence } from 'framer-motion';
import {
  Layers, Database, Zap, Shield, HardDrive, Smartphone,
  Settings, LogOut, Menu, Rocket, Terminal, Search, Bell, Activity
} from 'lucide-react';

import Dashboard from './pages/Dashboard';
import { useLayoutStore } from './hooks/useLayoutStore';
import { MOTION, VARIANTS } from './lib/motion';

type Route = 'dashboard' | 'database' | 'logic' | 'auth' | 'storage' | 'events' | 'push' | 'settings';

export default function App() {
  const { t } = useTranslation();
  const { preferences, updatePreference } = useLayoutStore();
  const [currentRoute, setCurrentRoute] = React.useState<Route>('dashboard');
  const [currentEnv, setCurrentEnv] = React.useState<'live' | 'draft'>('draft');

  const { sidebarCollapsed } = preferences;
  const setSidebarCollapsed = (val: boolean) => updatePreference('sidebarCollapsed', val);

  // Keyboard layout for OmniSearch
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        alert('Command+K: OmniSearch Activated');
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const renderContent = () => {
    switch (currentRoute) {
      case 'dashboard':
        return <Dashboard currentEnv={currentEnv} onEnvChange={setCurrentEnv} />;
      default:
        return (
          <motion.div 
            className="flex-1 flex items-center justify-center bg-surface-base"
            initial="hidden"
            animate="visible"
            variants={VARIANTS.fadeIn}
          >
            <div className="text-center opacity-50">
              <Layers size={64} className="mx-auto mb-4 text-content-muted animate-pulse" />
              <h2 className="text-2xl font-bold font-mono tracking-widest text-content-secondary uppercase">
                {t(`modules.${currentRoute}`)}
              </h2>
              <p className="mt-2 text-sm text-content-muted">{t('loading')}</p>
            </div>
          </motion.div>
        );
    }
  };

  const isExpanded = !sidebarCollapsed;

  return (
    <div className="flex h-screen bg-surface-base text-content-primary overflow-hidden font-sans selection:bg-accent-primary/30">
      
      {/* SIDEBAR */}
      <motion.aside
        initial={false}
        animate={{ width: isExpanded ? preferences.sidebarWidth : 72 }}
        transition={MOTION.spring}
        className="relative flex flex-col h-full bg-surface-pit border-r border-border-subtle z-50 shrink-0 shadow-lg"
      >
        {/* Sidebar Header */}
        <div className="h-16 flex items-center px-4 border-b border-border-subtle shrink-0">
          <button 
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="p-1.5 rounded-xl text-accent-primary hover:text-accent-dim hover:bg-surface-raised transition-colors flex items-center justify-center group"
            aria-label="Toggle Sidebar"
          >
            {isExpanded ? (
              <Layers size={22} className="group-hover:scale-105 transition-transform" />
            ) : (
              <Menu size={22} className="group-hover:scale-105 transition-transform" />
            )}
          </button>
          
          <AnimatePresence initial={false}>
            {isExpanded && (
              <motion.div 
                initial={{ opacity: 0, width: 0, marginLeft: 0 }}
                animate={{ opacity: 1, width: 'auto', marginLeft: 12 }}
                exit={{ opacity: 0, width: 0, marginLeft: 0 }}
                transition={MOTION.swift}
                className="overflow-hidden whitespace-nowrap"
              >
                <span className="font-bold text-[15px] tracking-wide text-content-primary block leading-tight">CASCATA</span>
                <span className="text-[9px] font-bold text-accent-primary uppercase tracking-[0.2em] block">Studio v1.0</span>
              </motion.div>
            )}
          </AnimatePresence>
        </div>

        {/* Sidebar Navigation */}
        <nav className="flex-1 overflow-y-auto custom-scrollbar py-4 px-3 flex flex-col gap-1.5">
          <NavItem icon={<Activity />} label={t('modules.dashboard')} route="dashboard" active={currentRoute === 'dashboard'} expanded={isExpanded} onClick={() => setCurrentRoute('dashboard')} />
          
          {isExpanded && <div className="text-[10px] font-bold text-content-muted uppercase tracking-widest mt-4 mb-2 pl-3">{t('headings.architecture')}</div>}
          
          <NavItem icon={<Database />} label={t('modules.database')} route="database" active={currentRoute === 'database'} expanded={isExpanded} onClick={() => setCurrentRoute('database')} />
          <NavItem icon={<Terminal />} label={t('modules.logic')} route="logic" active={currentRoute === 'logic'} expanded={isExpanded} onClick={() => setCurrentRoute('logic')} />
          <NavItem icon={<Shield />} label={t('modules.auth')} route="auth" active={currentRoute === 'auth'} expanded={isExpanded} onClick={() => setCurrentRoute('auth')} />
          <NavItem icon={<HardDrive />} label={t('modules.storage')} route="storage" active={currentRoute === 'storage'} expanded={isExpanded} onClick={() => setCurrentRoute('storage')} />
          
          {isExpanded && <div className="text-[10px] font-bold text-content-muted uppercase tracking-widest mt-4 mb-2 pl-3">{t('headings.engines')}</div>}
          
          <NavItem icon={<Zap />} label={t('modules.events')} route="events" active={currentRoute === 'events'} expanded={isExpanded} onClick={() => setCurrentRoute('events')} />
          <NavItem icon={<Smartphone />} label={t('modules.push')} route="push" active={currentRoute === 'push'} expanded={isExpanded} onClick={() => setCurrentRoute('push')} />
        </nav>

        {/* Sidebar Footer */}
        <div className="p-3 border-t border-border-subtle shrink-0">
          <NavItem icon={<Settings />} label={t('modules.settings')} route="settings" active={currentRoute === 'settings'} expanded={isExpanded} onClick={() => setCurrentRoute('settings')} />
          
          <button 
            className={`
              w-full mt-2 flex items-center rounded-xl transition-all duration-200 group relative sidebar-item
              ${isExpanded ? 'px-3 py-2.5 text-accent-danger hover:bg-accent-danger/10' : 'justify-center p-[12px] text-accent-danger hover:bg-accent-danger/10'}
            `}
            aria-label={t('logout')}
          >
            <LogOut size={isExpanded ? 18 : 20} className="group-hover:scale-105 transition-transform" />
            {isExpanded && <span className="ml-3 text-[13px] font-medium tracking-wide">{t('logout')}</span>}
            {!isExpanded && <div className="mac-tooltip">{t('logout')}</div>}
          </button>
        </div>
      </motion.aside>

      {/* MAIN WORKSPACE */}
      <main className="flex-1 relative flex flex-col min-w-0 bg-surface-base">
        
        {/* Global Omni-Header */}
        <header className="h-16 flex items-center justify-between px-6 bg-surface-base border-b border-border-subtle z-20 shrink-0 relative">
          <div className="flex items-center gap-4">
            
            <div className="flex items-center p-1 bg-surface-raised rounded-full border border-border-default shadow-inner">
              <button 
                onClick={() => setCurrentEnv('live')}
                className={`
                  px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest transition-colors duration-200
                  ${currentEnv === 'live' ? 'bg-accent-success/15 text-accent-success border border-transparent' : 'text-content-muted hover:text-content-primary border border-transparent'}
                `}
              >
                {t('environment.live')}
              </button>
              <button 
                onClick={() => setCurrentEnv('draft')}
                className={`
                  flex items-center gap-1.5 px-4 py-1.5 rounded-full text-xs font-bold uppercase tracking-widest transition-colors duration-200
                  ${currentEnv === 'draft' ? 'bg-accent-warning/15 text-accent-warning border border-transparent' : 'text-content-muted hover:text-content-primary border border-transparent'}
                `}
              >
                <Rocket size={12} className={currentEnv === 'draft' ? 'animate-soft-pulse' : ''} />
                {t('environment.draft')}
              </button>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <button className="flex items-center gap-2 px-3 py-1.5 bg-surface-raised hover:bg-surface-elevated rounded-lg border border-border-default text-content-muted hover:text-content-primary transition-colors">
              <Search size={14} />
              <span className="text-[11px] font-medium tracking-wide">{t('search')}</span>
              <kbd className="ml-2 font-mono text-[10px] bg-surface-elevated border border-border-subtle px-1.5 py-0.5 rounded opacity-80">⌘K</kbd>
            </button>
            
            <button className="p-2 text-content-muted hover:text-content-primary transition-colors relative" aria-label="Notifications">
              <Bell size={18} />
              <span className="absolute top-1.5 right-1.5 w-2 h-2 bg-accent-danger rounded-full shadow-lg border border-surface-base"></span>
            </button>
          </div>
        </header>

        {/* Content Area */}
        <div className="flex-1 overflow-y-auto overflow-x-hidden relative z-10 custom-scrollbar">
          <AnimatePresence mode="wait">
            <motion.div
              key={currentRoute}
              initial="hidden"
              animate="visible"
              exit="hidden"
              variants={VARIANTS.fadeIn}
              className="h-full"
            >
              {renderContent()}
            </motion.div>
          </AnimatePresence>
        </div>
      </main>
    </div>
  );
}

// NavItem Component
function NavItem({ icon, label, route, active, expanded, onClick }: any) {
  const iconSize = expanded ? 18 : 20;
  const TheIcon = React.cloneElement(icon as React.ReactElement, { size: iconSize });

  return (
    <button
      onClick={onClick}
      role="button"
      aria-label={label}
      className={`
        relative flex items-center rounded-[10px] transition-colors duration-200 group sidebar-item select-none
        ${expanded ? 'px-3 py-2.5 mx-0 w-full' : 'p-[12px] mx-auto justify-center aspect-square'}
        ${active 
          ? 'bg-accent-primary/10 text-accent-primary border border-transparent' 
          : 'text-content-secondary hover:bg-surface-raised hover:text-content-primary border border-transparent'}
      `}
    >
      <div className={`
        relative flex items-center justify-center transition-transform duration-200
        ${!active && 'group-hover:scale-105 group-active:scale-95'}
      `}>
        {TheIcon}
      </div>

      <AnimatePresence initial={false}>
        {expanded && (
          <motion.div 
            initial={{ opacity: 0, width: 0 }}
            animate={{ opacity: 1, width: 'auto' }}
            exit={{ opacity: 0, width: 0 }}
            transition={MOTION.swift}
            className="overflow-hidden whitespace-nowrap ml-3 text-sm font-medium tracking-wide"
          >
            {label}
          </motion.div>
        )}
      </AnimatePresence>

      {!expanded && (
        <div className="mac-tooltip">
          {label}
        </div>
      )}
    </button>
  );
}
