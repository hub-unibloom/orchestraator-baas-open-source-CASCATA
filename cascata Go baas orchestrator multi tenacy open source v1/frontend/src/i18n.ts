import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import HttpBackend from 'i18next-http-backend';
import LanguageDetector from 'i18next-browser-languagedetector';

// Sinergy: Implementation of the "Hybrid Bundle" Strategy
// Loads only 'common' on boot and other namespaces lazily.
i18n
  .use(HttpBackend)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: 'en_US',
    ns: ['common', 'dashboard', 'database'], // Available namespaces
    defaultNS: 'common',
    
    interpolation: {
      escapeValue: false, // React already escapes
    },
    
    backend: {
      // Sovereign Path: Internal Cascata API (served from /languages)
      loadPath: '/languages/{{lng}}/{{ns}}.json',
    },

    detection: {
      order: ['localStorage', 'navigator'],
      caches: ['localStorage'],
    },
    
    react: {
      useSuspense: true, // Seamless UI during language loading
    }
  });

export default i18n;
