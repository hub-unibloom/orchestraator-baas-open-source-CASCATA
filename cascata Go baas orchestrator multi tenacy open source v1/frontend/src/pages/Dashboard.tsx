import React from 'react';
import { useTranslation } from 'react-i18next';
import { motion } from 'framer-motion';
import { 
  Activity, Clock, Database, Globe, HardDrive, 
  Cpu, Zap, ShieldAlert, Users, Server, FileCode2,
  ChevronRight, ArrowUpRight, CheckCircle2, AlertCircle
} from 'lucide-react';
import { VARIANTS } from '../lib/motion';

interface DashboardProps {
  currentEnv: 'live' | 'draft';
  onEnvChange: (env: 'live' | 'draft') => void;
}

export default function Dashboard({ currentEnv, onEnvChange }: DashboardProps) {
  const { t } = useTranslation();

  return (
    <motion.div 
      variants={VARIANTS.fadeUp} 
      className="p-8 max-w-[1600px] mx-auto"
    >
      
      {/* HEADER: Magnificent Intro */}
      <header className="mb-10 flex items-end justify-between">
        <div>
          <div className="flex items-center gap-3 mb-2">
            <div className={`
              w-2 h-2 rounded-full animate-soft-pulse
              ${currentEnv === 'live' ? 'bg-accent-success shadow-lg' : 'bg-accent-warning shadow-lg'}
            `} />
            <h1 className="text-2xl font-black tracking-tight text-content-primary">
              {t('dashboard:header.title')}
            </h1>
          </div>
          <p className="text-content-muted font-medium tracking-wide text-sm">
            {t('dashboard:header.subtitle_prefix')} <span className="text-accent-primary font-bold">Cascata-Alpha-01</span>
          </p>
        </div>
        
        <div className="flex gap-3">
          <button className="px-5 py-2.5 bg-surface-raised hover:bg-surface-elevated border border-border-default rounded-[10px] text-sm font-bold tracking-wide text-content-secondary hover:text-content-primary transition-colors backdrop-blur-md">
            {t('dashboard:header.keys_button')}
          </button>
          <button className="px-5 py-2.5 bg-accent-primary hover:bg-accent-dim text-white shadow-lg rounded-[10px] text-sm font-bold tracking-wide transition-all flex items-center gap-2">
            <FileCode2 size={16} />
            {t('dashboard:header.generate_template')}
          </button>
        </div>
      </header>

      {/* BENTO GRID: New Era Web Design (Mac-like smooth corners, glassmorphism) */}
      <div className="grid grid-cols-12 gap-6">

        {/* --- MAIN METRICS ROW --- */}
        <div className="col-span-12 md:col-span-4 bg-surface-raised border border-border-default rounded-[20px] p-6 relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-6 opacity-[0.03] transform translate-x-4 -translate-y-4 group-hover:scale-110 transition-transform duration-700 pointer-events-none">
            <Activity size={100} className="text-content-primary" />
          </div>
          <div className="flex items-center gap-3 text-accent-success mb-6 font-bold tracking-widest text-[11px] uppercase">
            <Globe size={14} /> {t('dashboard:metrics.edge_gateway')}
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-5xl font-mono font-black text-content-primary tracking-tighter tabular-nums">1,204</span>
            <span className="text-content-muted font-bold text-sm">req/s</span>
          </div>
          <div className="mt-4 flex items-center gap-2 text-sm text-accent-success/80 font-medium">
            <ArrowUpRight size={14} /> {t('dashboard:metrics.increase_from_yesterday', { amount: 12 })}
          </div>
        </div>

        <div className="col-span-12 md:col-span-4 bg-surface-raised border border-border-default rounded-[20px] p-6 relative overflow-hidden group">
          <div className="absolute top-0 right-0 p-6 opacity-[0.03] transform translate-x-4 -translate-y-4 group-hover:scale-110 transition-transform duration-700 pointer-events-none">
            <Database size={100} className="text-content-primary" />
          </div>
          <div className="flex items-center gap-3 text-accent-primary mb-6 font-bold tracking-widest text-[11px] uppercase">
            <Cpu size={14} /> {t('dashboard:metrics.db_compute')}
          </div>
          <div className="flex items-baseline gap-2">
            <span className="text-5xl font-mono font-black text-content-primary tracking-tighter tabular-nums">8.4</span>
            <span className="text-content-muted font-bold text-sm">ms lat</span>
          </div>
          <div className="mt-4 flex items-center gap-2 text-sm text-accent-primary/80 font-medium">
            <Server size={14} /> {t('dashboard:metrics.active_connections', { count: 14 })}
          </div>
        </div>

        <div className="col-span-12 md:col-span-4 bg-surface-raised border border-border-default rounded-[20px] p-6 relative overflow-hidden group">
          <div className="flex items-center gap-3 text-accent-danger mb-6 font-bold tracking-widest text-[11px] uppercase">
            <AlertCircle size={14} /> {t('dashboard:metrics.system_health')}
          </div>
          <div className="space-y-4">
            <div className="flex justify-between items-center text-sm">
              <span className="text-content-secondary font-medium">{t('dashboard:metrics.postgres_core')}</span>
              <span className="px-2 py-0.5 bg-accent-success/15 text-accent-success rounded-md font-bold text-[10px] uppercase tracking-wider border border-transparent">{t('dashboard:metrics.healthy')}</span>
            </div>
            <div className="flex justify-between items-center text-sm">
              <span className="text-content-secondary font-medium">{t('dashboard:metrics.nginx_ingress')}</span>
              <span className="px-2 py-0.5 bg-accent-success/15 text-accent-success rounded-md font-bold text-[10px] uppercase tracking-wider border border-transparent">{t('dashboard:metrics.healthy')}</span>
            </div>
            <div className="flex justify-between items-center text-sm">
              <span className="text-content-secondary font-medium">{t('dashboard:metrics.background_cron')}</span>
              <span className="px-2 py-0.5 bg-accent-warning/15 text-accent-warning rounded-md font-bold text-[10px] uppercase tracking-wider border border-transparent">{t('dashboard:metrics.syncing')}</span>
            </div>
          </div>
        </div>

        {/* --- PROJECT CREATION SETTINGS HUB --- */}
        <div className="col-span-12 md:col-span-8 bg-surface-elevated border border-border-default rounded-[20px] flex flex-col overflow-hidden">
          <div className="bg-surface-raised border-b border-border-subtle p-6">
            <h3 className="text-base font-bold text-content-primary">{t('dashboard:configs.title')}</h3>
            <p className="text-xs font-medium text-content-muted mt-1">{t('dashboard:configs.subtitle')}</p>
          </div>
          
          <div className="p-6 grid grid-cols-2 gap-5 flex-1 bg-surface-base">
            <ConfigRow title={t('dashboard:configs.auth_providers')} value="Email, Google, Web3" icon={<Users size={16} className="text-accent-primary" />} />
            <ConfigRow title={t('dashboard:configs.storage_regions')} value="us-east-1, eu-west" icon={<HardDrive size={16} className="text-accent-success" />} />
            <ConfigRow title={t('dashboard:configs.vault_protection')} value="AES-256 (FIPS)" icon={<ShieldAlert size={16} className="text-accent-danger" />} />
            <ConfigRow title={t('dashboard:configs.automations')} value="7 Triggers, 2 Cron" icon={<Zap size={16} className="text-accent-warning" />} />
          </div>
          
          <div className="p-4 bg-surface-raised border-t border-border-subtle flex justify-between items-center">
            <span className="text-[11px] text-content-muted font-medium">{t('dashboard:configs.last_configured', { time: '2 hours ago' })}</span>
            <button className="text-xs font-bold text-accent-primary hover:text-accent-dim flex items-center gap-1 transition-colors">
              {t('dashboard:configs.edit_settings')} <ChevronRight size={14} />
            </button>
          </div>
        </div>

        {/* --- RECENT ACTIVITY / LOGS --- */}
        <div className="col-span-12 md:col-span-4 bg-surface-elevated border border-border-default rounded-[20px] p-6 flex flex-col">
          <div className="flex items-center gap-3 text-content-muted mb-6 font-bold tracking-widest text-[11px] uppercase">
            <Clock size={14} /> {t('dashboard:logs.title')}
          </div>
          
          <div className="space-y-5 flex-1">
            <LogItem message="Draft Rebase completed" time="10m ago" status="success" />
            <LogItem message="RPC 'get_user' updated" time="45m ago" status="success" />
            <LogItem message="Rate limit exceeded (IP: 192.x.x)" time="2h ago" status="warning" />
            <LogItem message="Storage Rule changed" time="5h ago" status="success" />
          </div>

          <button className="w-full mt-4 py-2 border border-border-default rounded-[10px] text-xs font-bold text-content-secondary hover:bg-surface-raised hover:text-content-primary transition-colors">
            {t('dashboard:logs.view_all')}
          </button>
        </div>

      </div>
    </motion.div>
  );
}

// --- MICRO COMPONENTS FOR BENTO GRID ---

function ConfigRow({ title, value, icon }: { title: string, value: string, icon: React.ReactNode }) {
  return (
    <div className="flex items-start gap-3 p-3 rounded-[12px] hover:bg-surface-raised transition-colors border border-transparent hover:border-border-subtle">
      <div className="p-2.5 bg-surface-elevated rounded-[10px] border border-border-subtle shadow-sm">
        {icon}
      </div>
      <div className="flex-1 min-w-0">
        <h4 className="text-[10px] font-bold text-content-muted uppercase tracking-widest mb-0.5 truncate">{title}</h4>
        <p className="text-[13px] font-semibold text-content-primary truncate">{value}</p>
      </div>
    </div>
  );
}

function LogItem({ message, time, status }: { message: string, time: string, status: 'success' | 'warning' | 'error' }) {
  const isSuccess = status === 'success';
  return (
    <div className="flex items-start gap-3">
      <div className="mt-0.5">
        {isSuccess ? (
          <CheckCircle2 size={14} className="text-accent-success" />
        ) : (
          <AlertCircle size={14} className="text-accent-warning" />
        )}
      </div>
      <div className="flex-1 min-w-0">
        <p className={`text-[13px] font-medium truncate ${isSuccess ? 'text-content-secondary' : 'text-accent-warning'}`}>
          {message}
        </p>
        <p className="text-[10px] font-mono font-medium text-content-muted mt-0.5">{time}</p>
      </div>
    </div>
  );
}
