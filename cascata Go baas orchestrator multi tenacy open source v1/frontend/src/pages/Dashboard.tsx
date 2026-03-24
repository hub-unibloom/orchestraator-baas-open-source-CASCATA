import React, { useState, useEffect } from 'react';
import { motion } from 'framer-motion';
import { useTranslation } from 'react-i18next';
import { LayoutGrid, List, Search, Plus, Filter, RefreshCcw, Bell } from 'lucide-react';
import ProjectCard from '../components/ProjectCard';
import TenantOnboardingModal from '../components/modals/TenantOnboardingModal';

interface DashboardProps {
  onEnterProject: (slug: string) => void;
}

const Dashboard: React.FC<DashboardProps> = ({ onEnterProject }) => {
  const { t } = useTranslation('dashboard');
  const [projects, setProjects] = useState<any[]>([]);
  const [search, setSearch] = useState('');
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [loading, setLoading] = useState(true);

  const fetchProjects = async () => {
    setLoading(true);
    try {
      const response = await fetch('/v1/system/projects');
      if (response.ok) {
        const data = await response.json();
        setProjects(data);
      }
    } catch (err) {
      console.error("Dashboard: failed to fetch projects", err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchProjects();
  }, []);

  const filteredProjects = projects.filter(p =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.slug.toLowerCase().includes(search.toLowerCase())
  );

  return (
    <div className="min-h-screen bg-surface-base text-content-primary p-12 lg:p-16 overflow-y-auto custom-scrollbar">
      {/* Header Central de Comando */}
      <div className="flex flex-col lg:flex-row justify-between items-start lg:items-center gap-8 mb-16">
        <motion.div initial={{ opacity: 0, x: -20 }} animate={{ opacity: 1, x: 0 }}>
          <h1 className="text-3xl font-extrabold tracking-tight text-white">Ecossistema Cascata</h1>
          <p className="text-content-muted mt-2">Orquestração soberana de inquilinos e infraestrutura.</p>
        </motion.div>

        <motion.div initial={{ opacity: 0, x: 20 }} animate={{ opacity: 1, x: 0 }} className="flex gap-4">
          <div className="relative group">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 text-content-muted group-focus-within:text-accent-primary transition-colors" size={18} />
            <input
              className="bg-surface-raised/40 border border-white/5 rounded-full py-3 pl-10 pr-4 w-64 outline-none focus:border-accent-primary/50 transition-all font-medium text-sm"
              placeholder="Pesquisar projetos..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
            />
          </div>
          <button
            onClick={() => setIsModalOpen(true)}
            className="bg-accent-primary hover:bg-accent-primary-dim text-white px-6 py-3 rounded-full font-bold flex items-center gap-2 transition-all shadow-lg shadow-accent-primary/20 hover:scale-105"
          >
            <Plus size={20} /> Novo Inquilino
          </button>
        </motion.div>
      </div>

      {/* Grid de Tenancies */}
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4 gap-8">
        {loading ? (
          <div className="col-span-full flex flex-col items-center justify-center py-32 space-y-4">
            <RefreshCcw className="animate-spin text-accent-primary" size={48} />
            <p className="text-content-muted font-mono tracking-widest text-[10px] uppercase">Sincronizando Pulso Neural...</p>
          </div>
        ) : filteredProjects.length > 0 ? (
          filteredProjects.map((p, i) => (
            <motion.div
              key={p.slug}
              initial={{ opacity: 0, y: 30 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: i * 0.05 }}
            >
              <ProjectCard
                project={p}
                onEnter={() => onEnterProject(p.slug)}
                onDelete={(slug) => console.log('Delete target:', slug)}
              />
            </motion.div>
          ))
        ) : (
          <div className="col-span-full flex flex-col items-center justify-center py-32 space-y-6">
            <div className="w-24 h-24 rounded-full bg-white/5 flex items-center justify-center text-content-muted border border-white/5">
              <LayoutGrid size={48} />
            </div>
            <div className="text-center">
              <p className="text-lg font-bold text-white">Nenhum Inquilino Detectado</p>
              <p className="text-content-muted mt-2">Inicie a gênese do seu primeiro projeto soberano.</p>
            </div>
          </div>
        )}
      </div>

      {/* Multi-step Onboarding Modal */}
      <TenantOnboardingModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        onSuccess={fetchProjects}
      />
    </div>
  );
};

export default Dashboard;
