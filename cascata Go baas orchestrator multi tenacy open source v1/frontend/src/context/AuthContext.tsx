import React, { createContext, useContext, useState, useEffect } from 'react';

interface AuthState {
  token: string | null;
  memberName: string | null;
  expiresAt: number | null; // Timestamp
  isAuthenticated: boolean;
  isLoading: boolean;
}

interface AuthContextType extends AuthState {
  login: (token: string, memberName: string, expiresAt: string) => void;
  logout: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    token: localStorage.getItem('cascata_session'),
    memberName: localStorage.getItem('cascata_member'),
    expiresAt: Number(localStorage.getItem('cascata_expires')) || null,
    isAuthenticated: !!localStorage.getItem('cascata_session'),
    isLoading: true,
  });

  useEffect(() => {
    // Check expiration on mount and periodically
    const checkAuth = () => {
      const { token, expiresAt } = state;
      if (token && expiresAt && Date.now() > expiresAt) {
        logout();
      }
    };

    checkAuth();
    const interval = setInterval(checkAuth, 30000); // Every 30s
    
    setState(prev => ({ ...prev, isLoading: false }));
    return () => clearInterval(interval);
  }, []);

  const login = (token: string, memberName: string, expiresAt: string) => {
    const expiresTimestamp = new Date(expiresAt).getTime();
    
    localStorage.setItem('cascata_session', token);
    localStorage.setItem('cascata_member', memberName);
    localStorage.setItem('cascata_expires', expiresTimestamp.toString());
    
    setState({
      token,
      memberName,
      expiresAt: expiresTimestamp,
      isAuthenticated: true,
      isLoading: false,
    });
  };

  const logout = () => {
    localStorage.removeItem('cascata_session');
    localStorage.removeItem('cascata_member');
    localStorage.removeItem('cascata_expires');
    
    setState({
      token: null,
      memberName: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,
    });
  };

  return (
    <AuthContext.Provider value={{ ...state, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
