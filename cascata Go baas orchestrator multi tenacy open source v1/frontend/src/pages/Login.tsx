import React, { useState } from 'react';
import { motion, AnimatePresence } from 'framer-motion';
import { Shield, Mail, Lock, Key, ArrowRight, Loader2, Info } from 'lucide-react';
import { useAuth } from '../context/AuthContext';
import { MOTION, VARIANTS } from '../lib/motion';

export default function Login() {
  const { login } = useAuth();
  const [step, setStep] = useState<'credentials' | 'otp'>('credentials');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Form Fields
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [otp, setOtp] = useState('');

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      // Simulate Backend Call (Actual API URL from .env or window.location)
      const res = await fetch('/system/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password, otp: step === 'otp' ? otp : undefined }),
      });

      const data = await res.json();

      if (res.status === 202 && data.status === 'MFA_REQUIRED') {
        setStep('otp');
        setLoading(false);
        return;
      }

      if (res.ok) {
        // Success
        login(data.token, data.member_name, data.expires_at || new Date(Date.now() + 900000).toISOString());
      } else {
        setError(data.error || 'Authentication failed');
      }
    } catch (err) {
      setError('Network error: Is the engine running?');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="min-h-screen bg-[#070707] flex items-center justify-center p-6 overflow-hidden relative selection:bg-accent-primary/50">
      {/* Decorative background elements */}
      <div className="absolute top-[-10%] left-[-5%] w-[40%] h-[50%] bg-accent-primary/5 blur-[120px] rounded-full pointer-events-none" />
      <div className="absolute bottom-[-10%] right-[-5%] w-[40%] h-[50%] bg-accent-primary/5 blur-[120px] rounded-full pointer-events-none" />

      <motion.div 
        initial="hidden"
        animate="visible"
        variants={VARIANTS.fadeIn}
        className="w-full max-w-[420px] z-10"
      >
        {/* Logo and Branding */}
        <div className="text-center mb-10">
          <motion.div 
            className="w-16 h-16 bg-surface-raised border border-border-default rounded-2xl flex items-center justify-center mx-auto mb-6 shadow-2xl"
            whileHover={{ scale: 1.05, rotate: 5 }}
            transition={MOTION.spring}
          >
            <Shield size={32} className="text-accent-primary" />
          </motion.div>
          <h1 className="text-3xl font-bold tracking-tight text-content-primary mb-2">CASCATA</h1>
          <p className="text-content-muted text-sm font-medium tracking-wide">
            {step === 'credentials' ? 'Painel de Gestão Multi-Tenant v1' : 'Verificação de Segundo Fator Requerida'}
          </p>
        </div>

        {/* Login Card */}
        <div className="bg-surface-pit/50 backdrop-blur-3xl border border-white/[0.05] rounded-3xl p-8 shadow-2xl relative">
          <form onSubmit={handleLogin} className="space-y-5">
            <AnimatePresence mode="wait">
              {step === 'credentials' ? (
                <motion.div 
                  key="credentials"
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: 20 }}
                  transition={MOTION.swift}
                  className="space-y-4"
                >
                  <div className="space-y-2">
                    <label className="text-[11px] font-bold text-content-muted uppercase tracking-widest ml-1">Worner E-mail</label>
                    <div className="relative group">
                      <Mail size={16} className="absolute left-4 top-1/2 -translate-y-1/2 text-content-muted group-focus-within:text-accent-primary transition-colors" />
                      <input 
                        type="email"
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        placeholder="admin@cascata.io"
                        required
                        className="w-full h-12 bg-surface-raised/40 border border-border-default rounded-xl pl-12 pr-4 text-[14px] text-content-primary focus:border-accent-primary/50 focus:ring-4 focus:ring-accent-primary/5 outline-none transition-all placeholder:text-content-muted/50"
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <label className="text-[11px] font-bold text-content-muted uppercase tracking-widest ml-1">Senha Mestre</label>
                    <div className="relative group">
                      <Lock size={16} className="absolute left-4 top-1/2 -translate-y-1/2 text-content-muted group-focus-within:text-accent-primary transition-colors" />
                      <input 
                        type="password"
                        value={password}
                        onChange={(e) => setPassword(e.target.value)}
                        placeholder="••••••••••••"
                        required
                        className="w-full h-12 bg-surface-raised/40 border border-border-default rounded-xl pl-12 pr-4 text-[14px] text-content-primary focus:border-accent-primary/50 focus:ring-4 focus:ring-accent-primary/5 outline-none transition-all placeholder:text-content-muted/50"
                      />
                    </div>
                  </div>
                </motion.div>
              ) : (
                <motion.div 
                  key="otp"
                  initial={{ opacity: 0, x: 20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, x: -20 }}
                  transition={MOTION.swift}
                  className="space-y-4"
                >
                  <div className="bg-accent-primary/10 border border-accent-primary/20 rounded-2xl p-4 flex gap-3 text-accent-primary mb-6">
                    <Info size={18} className="shrink-0" />
                    <p className="text-[12px] leading-relaxed">Insira o código gerado pelo Google Authenticator ou similar para liberar o acesso ao núcleo.</p>
                  </div>
                  
                  <div className="space-y-2">
                    <label className="text-[11px] font-bold text-content-muted uppercase tracking-widest ml-1">Código OTP (TOTP)</label>
                    <div className="relative group">
                      <Key size={16} className="absolute left-4 top-1/2 -translate-y-1/2 text-content-muted group-focus-within:text-accent-primary transition-colors" />
                      <input 
                        type="text"
                        value={otp}
                        onChange={(e) => setOtp(e.target.value)}
                        placeholder="000 000"
                        autoFocus
                        required
                        className="w-full h-14 bg-surface-raised/40 border border-border-default rounded-xl pl-12 pr-4 text-[18px] font-mono tracking-[0.4em] text-content-primary focus:border-accent-primary/50 focus:ring-4 focus:ring-accent-primary/5 outline-none transition-all placeholder:text-content-muted/50"
                      />
                    </div>
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            {error && (
              <motion.div 
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="p-3 bg-accent-danger/10 border border-accent-danger/20 rounded-xl text-accent-danger text-xs font-medium text-center"
              >
                {error}
              </motion.div>
            )}

            <button 
              type="submit"
              disabled={loading}
              className="w-full h-12 bg-accent-primary hover:bg-accent-dim disabled:opacity-50 disabled:cursor-not-allowed text-white text-[14px] font-bold rounded-xl shadow-lg shadow-accent-primary/20 transition-all flex items-center justify-center gap-2 group"
            >
              {loading ? (
                <Loader2 size={18} className="animate-spin" />
              ) : (
                <>
                  {step === 'credentials' ? 'Acessar Orquestrador' : 'Validar e Entrar'}
                  <ArrowRight size={16} className="group-hover:translate-x-1 transition-transform" />
                </>
              )}
            </button>
          </form>

          {step === 'otp' && (
            <button 
              onClick={() => setStep('credentials')}
              className="w-full mt-4 text-[11px] font-bold text-content-muted hover:text-content-primary uppercase tracking-widest transition-colors"
            >
              Voltar para login
            </button>
          )}
        </div>

        {/* Help Footer */}
        <div className="mt-8 flex items-center justify-center gap-6 text-[11px] font-bold text-content-muted uppercase tracking-widest">
          <a href="#" className="hover:text-content-primary transition-colors underline-offset-4 hover:underline">Recuperação de Acesso</a>
          <span className="w-1 h-1 bg-border-subtle rounded-full" />
          <a href="#" className="hover:text-content-primary transition-colors underline-offset-4 hover:underline">Status do Sistema</a>
        </div>
      </motion.div>
    </div>
  );
}
