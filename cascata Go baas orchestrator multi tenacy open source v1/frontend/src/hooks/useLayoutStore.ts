import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export interface LayoutPreferences {
  sidebarWidth: number;
  sidebarOrder: string[];
  sidebarCollapsed: boolean;
  bottomBarHeight: number;
  splitViewEnabled: boolean;
  splitViewRatio: number;
  dbExplorerPinnedTables: string[];
  activeTheme: string;
  activeLocale: string;
  densityMode: 'comfortable' | 'compact' | 'spacious';
}

interface LayoutStore {
  preferences: LayoutPreferences;
  updatePreference: <K extends keyof LayoutPreferences>(key: K, value: LayoutPreferences[K]) => void;
}

const defaultPreferences: LayoutPreferences = {
  sidebarWidth: 240,
  sidebarOrder: ['dashboard', 'database', 'logic', 'auth', 'storage', 'events', 'push', 'settings'],
  sidebarCollapsed: false,
  bottomBarHeight: 300,
  splitViewEnabled: false,
  splitViewRatio: 0.5,
  dbExplorerPinnedTables: [],
  activeTheme: 'cascata-dark',
  activeLocale: 'en',
  densityMode: 'comfortable',
};

// Simple debounced sync mock logic
let syncTimeout: any = null;
const debouncedSync = (data: LayoutPreferences) => {
  if (syncTimeout) clearTimeout(syncTimeout);
  syncTimeout = setTimeout(() => {
    console.log('[API] Syncing layout preferences...', data);
    // TODO: fetch('/api/control/users/preferences', { method: 'PATCH', body: JSON.stringify(data) })
  }, 2000);
};

export const useLayoutStore = create<LayoutStore>()(
  persist(
    (set, get) => ({
      preferences: defaultPreferences,
      updatePreference: (key, value) => {
        set(state => ({
          preferences: { ...state.preferences, [key]: value }
        }));
        debouncedSync(get().preferences);
      },
    }),
    { name: 'cascata-layout' }
  )
);
