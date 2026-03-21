import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import commonEN from './locales/en/common.json';
import dashboardEN from './locales/en/dashboard.json';

// In a real scenario, this could be lazy loaded via i18next-http-backend
i18n
  .use(initReactI18next)
  .init({
    resources: {
      en: {
        common: commonEN,
        dashboard: dashboardEN
      }
    },
    lng: 'en', // Default locale
    fallbackLng: 'en',
    interpolation: {
      escapeValue: false // React already escapes values
    }
  });

export default i18n;
