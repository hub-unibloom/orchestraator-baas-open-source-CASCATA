import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { useTranslation } from 'react-i18next'; // Sinergy with i18n law
import { Shield, Server, Activity, ArrowRight, ArrowLeft, CheckCircle } from 'lucide-react';

interface Quotas {
  max_users: number;
  max_conns: number;
  max_storage_mb: number;
  max_db_weight_mb: number;
}

interface TenantOnboardingModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (project: any) => void;
}

const TenantOnboardingModal: React.FC<TenantOnboardingModalProps> = ({ isOpen, onClose, onSuccess }) => {
  const { t } = useTranslation('dashboard');
  const [step, setStep] = useState(1);
  const [loading, setLoading] = useState(false);

  // Form State
  const [formData, setFormData] = useState({
    name: '',
    slug: '',
    region: 'us-east-1',
    timezone: 'UTC',
    max_users: 100,
    max_conns: 10,
    max_storage_mb: 1024,
    max_db_weight_mb: 1024,
    secondary_secret: ''
  });

  const handleSlugify = (name: string) => {
    const slug = name.toLowerCase().replace(/[^a-z0-9]/g, '-').replace(/-+/g, '-');
    setFormData({ ...formData, name, slug });
  };

  const handleCreate = async () => {
    setLoading(true);
    try {
      const response = await fetch('/v1/system/projects', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(formData)
      });

      if (response.ok) {
        const data = await response.json();
        onSuccess(data);
        onClose();
      }
    } catch (err) {
      console.error("Genesis deployment failed", err);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <AnimatePresence>
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-surface-pit/60 backdrop-blur-md">
        <motion.div
          initial={{ opacity: 0, scale: 0.95, y: 20 }}
          animate={{ opacity: 1, scale: 1, y: 0 }}
          exit={{ opacity: 0, scale: 0.95, y: 20 }}
          className="relative w-full max-w-2xl overflow-hidden glass-panel rounded-2xl border border-white/10 shadow-2xl p-8 bg-surface-base/80"
        >
          {/* Header */}
          <div className="flex items-center justify-between mb-8">
            <div>
              <h2 className="text-2xl font-bold text-content-primary">{t('new_project.title')}</h2>
              <p className="text-content-secondary mt-1">
                {step === 1 && t('new_project.step_info')}
                {step === 2 && t('new_project.step_quotas')}
                {step === 3 && t('new_project.step_security')}
              </p>
            </div>
            <div className="flex gap-2">
              {[1, 2, 3].map(i => (
                <div key={i} className={`h-1.5 w-12 rounded-full transition-all ${step >= i ? 'bg-accent-primary' : 'bg-white/10'}`} />
              ))}
            </div>
          </div>

          {/* Steps Content */}
          <div className="min-h-[340px]">
            {step === 1 && (
              <motion.div initial={{ x: 20, opacity: 0 }} animate={{ x: 0, opacity: 1 }} className="space-y-6">
                <div className="grid grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-content-muted">{t('new_project.fields.name')}</label>
                    <input
                      className="w-full bg-white/5 border border-white/10 rounded-lg p-3 text-content-primary focus:border-accent-primary outline-none transition-all"
                      value={formData.name}
                      onChange={(e) => handleSlugify(e.target.value)}
                      placeholder="Ex: Cascata Marketplace"
                    />
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-content-muted">{t('new_project.fields.slug')}</label>
                    <input
                      disabled
                      className="w-full bg-white/5 border border-white/10 rounded-lg p-3 text-content-muted font-mono"
                      value={formData.slug}
                    />
                  </div>
                </div>
                <div className="grid grid-cols-2 gap-6">
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-content-muted">{t('new_project.fields.region')}</label>
                    <select
                      className="w-full bg-surface-elevated border border-white/10 rounded-lg p-3 text-content-primary focus:border-accent-primary outline-none"
                      value={formData.region}
                      onChange={(e) => setFormData({ ...formData, region: e.target.value })}
                    >
                      <option value="us-east-1">US East (N. Virginia)</option>
                      <option value="sa-east-1">South America (São Paulo)</option>
                      <option value="eu-central-1">Europe (Frankfurt)</option>
                    </select>
                  </div>
                  <div className="space-y-2">
                    <label className="text-sm font-medium text-content-muted">{t('new_project.fields.timezone')}</label>
                    <select className="w-full bg-surface-elevated border border-white/10 rounded-lg p-3 text-content-primary" value={formData.timezone} onChange={(e) => setFormData({ ...formData, timezone: e.target.value })}>
                      <option value="UTC">UTC</option>
                      <option value="America/Sao_Paulo">GMT-3 (São Paulo)</option>
                    </select>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 2 && (
              <motion.div initial={{ x: 20, opacity: 0 }} animate={{ x: 0, opacity: 1 }} className="space-y-8">
                <div className="space-y-4">
                  <div className="flex justify-between">
                    <label className="text-sm font-medium text-content-secondary flex items-center gap-2"><Activity size={16} /> {t('new_project.fields.max_users')}</label>
                    <span className="text-accent-primary font-mono font-bold">{formData.max_users}</span>
                  </div>
                  <input type="range" min="10" max="5000" step="10" className="w-full accent-accent-primary" value={formData.max_users} onChange={(e) => setFormData({ ...formData, max_users: parseInt(e.target.value) })} />
                </div>

                <div className="space-y-4">
                  <div className="flex justify-between">
                    <label className="text-sm font-medium text-content-secondary flex items-center gap-2"><Server size={16} /> {t('new_project.fields.max_conns')}</label>
                    <span className="text-accent-primary font-mono font-bold">{formData.max_conns}</span>
                  </div>
                  <input type="range" min="5" max="200" step="5" className="w-full accent-accent-primary" value={formData.max_conns} onChange={(e) => setFormData({ ...formData, max_conns: parseInt(e.target.value) })} />
                </div>

                <div className="grid grid-cols-2 gap-6">
                  <div className="p-4 rounded-xl border border-white/5 bg-white/5 space-y-2">
                    <p className="text-xs text-content-muted">{t('new_project.fields.max_storage')}</p>
                    <p className="text-lg font-bold text-content-primary">{formData.max_storage_mb / 1024} GB</p>
                  </div>
                  <div className="p-4 rounded-xl border border-white/5 bg-white/5 space-y-2">
                    <p className="text-xs text-content-muted">{t('new_project.fields.max_db')}</p>
                    <p className="text-lg font-bold text-content-primary">{formData.max_db_weight_mb} MB</p>
                  </div>
                </div>
              </motion.div>
            )}

            {step === 3 && (
              <motion.div initial={{ x: 20, opacity: 0 }} animate={{ x: 0, opacity: 1 }} className="space-y-6">
                <div className="p-6 rounded-2xl bg-accent-primary/10 border border-accent-primary/20 flex gap-4">
                  <Shield className="text-accent-primary shrink-0" size={32} />
                  <div>
                    <h4 className="font-bold text-content-primary">{t('new_project.fields.secondary_secret')}</h4>
                    <p className="text-sm text-content-secondary mt-1">{t('new_project.fields.secondary_secret_hint')}</p>
                  </div>
                </div>
                <div className="space-y-2">
                  <input
                    type="password"
                    className="w-full bg-white/5 border border-white/10 rounded-lg p-4 text-content-primary focus:border-accent-primary outline-none transition-all text-center tracking-widest text-xl"
                    placeholder="••••••••••••"
                    value={formData.secondary_secret}
                    onChange={(e) => setFormData({ ...formData, secondary_secret: e.target.value })}
                  />
                </div>
              </motion.div>
            )}
          </div>

          {/* Footer Actions */}
          <div className="mt-12 flex justify-between items-center">
            <button
              onClick={() => step > 1 && setStep(step - 1)}
              className={`flex items-center gap-2 p-3 text-content-secondary hover:text-content-primary transition-colors ${step === 1 ? 'invisible' : ''}`}
            >
              <ArrowLeft size={18} /> {t('modals.cancel')}
            </button>

            {step < 3 ? (
              <button
                onClick={() => setStep(step + 1)}
                className="bg-accent-primary hover:bg-accent-primary-dim text-white px-8 py-3 rounded-full font-bold flex items-center gap-2 transition-all shadow-lg shadow-accent-primary/20"
              >
                {t('modals.continue')} <ArrowRight size={18} />
              </button>
            ) : (
              <button
                onClick={handleCreate}
                disabled={loading}
                className="bg-accent-success hover:bg-accent-success/90 text-white px-10 py-3 rounded-full font-bold flex items-center gap-2 transition-all shadow-lg shadow-accent-success/20 disabled:opacity-50"
              >
                {loading ? t('status.loading') : t('actions.deploy')} <CheckCircle size={18} />
              </button>
            )}
          </div>
        </motion.div>
      </div>
    </AnimatePresence>
  );
};

export default TenantOnboardingModal;
