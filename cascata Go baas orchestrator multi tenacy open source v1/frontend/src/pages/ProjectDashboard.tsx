import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { 
  Activity, Users, Database, Table as TableIcon, 
  Shield, LayoutGrid, Settings, ArrowLeft, Terminal,
  HardDrive, Zap, Info, Layers
} from 'lucide-react';
import NavigationModal from '../components/shared/NavigationModal';
import { VARIANTS } from '../lib/motion';

interface Stats {
  total_tables: number;
  schema_size_bytes: number;
  total_users: number;
  status: string;
  table_names: string[];
}

interface ProjectDashboardProps {
  slug: string;
  onExit: () => void;
}

const ProjectDashboard: React.FC<ProjectDashboardProps> = ({ slug, onExit }) => {
  const [activeTab, setActiveTab] = useState<'overview' | 'mpc' | 'settings' | 'members'>('overview');
  const [isNavOpen, setIsNavOpen] = useState(false);
  const [stats, setStats] = useState<Stats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await fetch(`/v1/${slug}/stats`);
        const data = await res.json();
        setStats(data);
      } catch (err) {
        console.error("Fetch failed", err);
      } finally {
        setLoading(false);
      }
    };
    fetchStats();
    const interval = setInterval(fetchStats, 30000);
    return () => clearInterval(interval);
  }, [slug]);

  const navItems = [
    { icon: LayoutGrid, label: 'Overview', id: 'overview' },
    { icon: Shield, label: 'MPC Settings', id: 'mpc' },
    { icon: Settings, label: 'Project Settings', id: 'settings' },
    { icon: Users, label: 'Membros Gestores', id: 'members' },
  ];

  return (
    <div className="min-h-screen bg-surface-base text-content-primary font-sans selection:bg-accent-primary/20">
      <header className="h-20 border-b border-white/5 flex items-center justify-between px-10 glass-panel sticky top-0 z-40">
        <div className="flex items-center gap-6">
          <button onClick={onExit} className="p-2 rounded-xl hover:bg-white/5 text-content-muted hover:text-white transition-all">
            <ArrowLeft size={20} />
          </button>
          <div className="h-8 w-[1px] bg-white/10" />
          <div>
            <h1 className="text-lg font-bold tracking-tight text-white">{slug.toUpperCase()}</h1>
            <p className="text-[10px] uppercase tracking-[0.2em] font-bold text-accent-primary">Project Studio</p>
          </div>
        </div>

        <button 
          onClick={() => setIsNavOpen(true)}
          className="flex items-center gap-3 px-6 py-2.5 rounded-2xl bg-white/5 border border-white/10 hover:border-accent-primary/50 transition-all group"
        >
          <div className="w-2 h-2 rounded-full bg-accent-primary animate-pulse shadow-[0_0_10px_rgba(var(--accent-primary-rgb),0.5)]" />
          <span className="text-sm font-bold tracking-wide group-hover:text-accent-primary">MENU DO PROJETO</span>
          <Layers size={16} className="text-content-muted" />
        </button>
      </header>

      <main className="p-10 pt-12 custom-scrollbar overflow-y-auto h-[calc(100vh-80px)]">
         <AnimatePresence mode="wait">
            {activeTab === 'overview' && (
              <ProjectOverview key="overview" stats={stats} loading={loading} />
            )}
            {activeTab !== 'overview' && (
              <motion.div key="other" variants={VARIANTS.fadeIn} initial="hidden" animate="visible" className="flex items-center justify-center h-full opacity-30 py-20">
                 <Terminal size={48} className="animate-pulse" />
              </motion.div>
            )}
         </AnimatePresence>
      </main>

      <NavigationModal 
        isOpen={isNavOpen} 
        onClose={() => setIsNavOpen(false)} 
        title="Navegação do Projeto" 
        items={navItems}
        onSelect={(id: any) => {
          setActiveTab(id);
          setIsNavOpen(false);
        }}
      />
    </div>
  );
};

// Subview: ProjectOverview
const ProjectOverview: React.FC<{ stats: Stats | null, loading: boolean }> = ({ stats, loading }) => {
  const storageMB = (stats?.schema_size_bytes || 0) / 1024 / 1024;
  const storageLimit = 512; 
  const storagePercent = Math.min((storageMB / storageLimit) * 100, 100);

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: -20 }} className="max-w-7xl mx-auto space-y-12 pb-20">
       <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-8">
          <MetricCard icon={Activity} label="Status Operacional" value={stats?.status || "Inativo"} color="text-accent-success" loading={loading} />
          <MetricCard icon={Users} label="Membros Totais" value={stats?.total_users || 0} color="text-accent-primary" loading={loading} />
          <MetricCard icon={TableIcon} label="Entidades" value={stats?.total_tables || 0} color="text-accent-secondary" loading={loading} />
          <MetricCard icon={Database} label="Storage" value={`${storageMB.toFixed(2)} MB`} color="text-white" loading={loading} />
       </div>

       <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          <div className="lg:col-span-2 glass-panel p-8 rounded-3xl border border-white/5 bg-surface-raised/40 relative overflow-hidden">
             <div className="absolute top-0 right-0 w-64 h-64 bg-accent-primary/5 blur-[120px] -mr-32 -mt-32" />
             <div className="flex justify-between items-center mb-10 relative z-10">
                <h3 className="text-xl font-bold flex items-center gap-2 text-white"><LayoutGrid size={20} className="text-accent-primary"/> Inventário do Database</h3>
             </div>
             <div className="space-y-4 relative z-10">
                {stats?.table_names && stats.table_names.map((name) => (
                  <div key={name} className="flex items-center justify-between p-5 rounded-2xl bg-black/20 border border-white/5 hover:border-white/10 transition-all group cursor-pointer">
                     <div className="flex items-center gap-4">
                        <div className="w-10 h-10 rounded-lg bg-white/5 flex items-center justify-center text-content-muted group-hover:text-accent-secondary transition-colors">
                           <TableIcon size={18} />
                        </div>
                        <span className="font-mono text-sm text-white">{name}</span>
                     </div>
                     <span className="text-[10px] text-content-muted uppercase font-bold px-3 py-1 bg-white/5 rounded-full group-hover:text-accent-success transition-colors">Protegida</span>
                  </div>
                ))}
                {!loading && (!stats?.table_names || stats.table_names.length === 0) && (
                   <div className="text-center py-10 opacity-40">
                      <Info size={32} className="mx-auto mb-2" />
                      <p className="text-xs uppercase tracking-widest font-bold">Nenhum esquema detectado</p>
                   </div>
                )}
             </div>
          </div>

          <div className="space-y-8">
             <div className="glass-panel p-8 rounded-3xl border border-white/5 bg-surface-raised/40">
                <h3 className="text-lg font-bold mb-8 flex items-center gap-2"><HardDrive size={20} className="text-accent-secondary"/> Monitor de Quota</h3>
                <div className="space-y-6">
                   <div>
                      <div className="flex justify-between text-xs font-bold uppercase tracking-widest text-content-muted mb-3">
                         <span>Consumo</span>
                         <span>{storagePercent.toFixed(1)}%</span>
                      </div>
                      <div className="h-2 w-full bg-white/5 rounded-full overflow-hidden">
                         <div className="h-full bg-gradient-to-r from-accent-primary to-accent-secondary" style={{ width: `${storagePercent}%` }} />
                      </div>
                   </div>
                   <p className="text-xs text-content-muted leading-relaxed">Isolamento físico e lógico ativo conforme padrões MPC.</p>
                </div>
             </div>
          </div>
       </div>
    </motion.div>
  );
};

const MetricCard: React.FC<{ icon: any, label: string, value: any, color: string, loading: boolean }> = ({ icon: Icon, label, value, color, loading }) => (
  <div className="glass-panel p-8 rounded-3xl border border-white/5 bg-surface-raised/30 hover:bg-surface-elevated transition-all group overflow-hidden relative">
    <div className={`absolute top-0 right-0 p-4 opacity-5 group-hover:opacity-10 transition-opacity ${color}`}>
      <Icon size={64} />
    </div>
    <div className="relative z-10 flex flex-col gap-1">
      <span className="text-[10px] font-bold uppercase tracking-widest text-content-muted">{label}</span>
      {loading ? (
        <div className="h-9 w-24 bg-white/5 animate-pulse rounded-lg mt-1" />
      ) : (
        <span className={`text-3xl font-bold tracking-tight font-mono ${color}`}>{value}</span>
      )}
    </div>
  </div>
);

export default ProjectDashboard;
