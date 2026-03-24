import React from 'react';
import { motion } from 'framer-motion';
import { useTranslation } from 'react-i18next';
import { Server, Users, Database, Globe, ArrowUpRight, Trash2, Activity } from 'lucide-react';

interface Project {
  id: string;
  name: string;
  slug: string;
  region: string;
  status: string;
  max_users: number;
  max_conns: number;
  max_storage_mb: number;
}

interface ProjectCardProps {
  project: Project;
  onEnter: (slug: string) => void;
  onDelete: (slug: string) => void;
}

const ProjectCard: React.FC<ProjectCardProps> = ({ project, onEnter, onDelete }) => {
  const { t } = useTranslation('dashboard');

  // Robustness guards for null-safety and visual integrity
  const storageGB = (project.max_storage_mb || 0) / 1024;
  const statusColor = project.status === 'active' ? 'bg-accent-success' : 'bg-content-muted';

  return (
    <motion.div
      whileHover={{ y: -4, scale: 1.01 }}
      className="glass-panel p-6 rounded-2xl group relative overflow-hidden transition-all border border-white/5 hover:border-accent-primary/50 bg-surface-raised/40"
    >
      <div className="absolute top-0 right-0 w-32 h-32 bg-accent-primary/5 blur-3xl rounded-full -mr-16 -mt-16 group-hover:bg-accent-primary/10 transition-colors" />

      <div className="flex justify-between items-start mb-6">
        <div className="flex items-center gap-4">
          <div className="w-12 h-12 rounded-xl bg-accent-primary/10 flex items-center justify-center text-accent-primary">
            <Server size={24} />
          </div>
          <div>
            <h3 className="text-lg font-bold text-content-primary">{project.name || "Sem Nome"}</h3>
            <div className="flex items-center gap-2 mt-0.5 text-[10px] font-mono text-content-muted uppercase tracking-wider">
              <div className={`w-1.5 h-1.5 rounded-full ${statusColor} animate-pulse`} />
              {project.region || "BRA"} • /{project.slug || "unknown"}
            </div>
          </div>
        </div>

        <div className="flex gap-2 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            onClick={() => onDelete(project.slug)}
            className="p-2 rounded-lg bg-accent-danger/10 text-accent-danger hover:bg-accent-danger/20 transition-colors"
          >
            <Trash2 size={16} />
          </button>
          <button
            onClick={() => onEnter(project.slug)}
            className="p-2 rounded-lg bg-accent-primary/10 text-accent-primary hover:bg-accent-primary/20 transition-colors"
          >
            <ArrowUpRight size={16} />
          </button>
        </div>
      </div>

      <div className="grid grid-cols-3 gap-4 border-t border-white/5 pt-6 relative z-10">
        <div className="space-y-1">
          <p className="text-[10px] uppercase tracking-wider text-content-muted flex items-center gap-1 font-bold">
            <Users size={10} /> USERS
          </p>
          <p className="text-sm font-mono text-content-primary tabular-nums">{project.max_users || 0}</p>
        </div>
        <div className="space-y-1">
          <p className="text-[10px] uppercase tracking-wider text-content-muted flex items-center gap-1 font-bold">
            <Activity size={10} /> CONNS
          </p>
          <p className="text-sm font-mono text-content-primary tabular-nums">{project.max_conns || 0}</p>
        </div>
        <div className="space-y-1">
          <p className="text-[10px] uppercase tracking-wider text-content-muted flex items-center gap-1 font-bold">
            <Database size={10} /> STORAGE
          </p>
          <p className="text-sm font-mono text-content-primary tabular-nums">{storageGB.toFixed(1)}GB</p>
        </div>
      </div>

      <div className="mt-6 h-1 w-full bg-white/5 rounded-full overflow-hidden relative">
        <div className="h-full bg-accent-primary w-2/3 opacity-30 shadow-[0_0_8px_rgba(var(--accent-primary-rgb),0.4)]" />
      </div>
    </motion.div>
  );
};

export default ProjectCard;
