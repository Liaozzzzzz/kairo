import { create } from 'zustand';
import { UpdateSettings, GetSettings } from '@root/wailsjs/go/main/App';
import { config as WailsConfig } from '@root/wailsjs/go/models';

export type AppLanguage = 'zh' | 'en';
export type AppTheme = 'light' | 'dark' | 'system';

export interface CookieConfig {
  enabled: boolean;
  source: 'browser' | 'file';
  browser: string;
  file: string;
}

export interface AppSettings {
  downloadDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  theme: AppTheme;
  proxyUrl: string;
  cookie: CookieConfig;
}

const SETTINGS_STORAGE_KEY = 'Kairo.settings';

const DEFAULT_COOKIE_CONFIG: CookieConfig = {
  enabled: false,
  source: 'browser',
  browser: '',
  file: '',
};

const DEFAULT_SETTINGS: AppSettings = {
  downloadDir: '',
  downloadConcurrency: 3,
  maxDownloadSpeed: null,
  language: 'zh',
  theme: 'system',
  proxyUrl: '',
  cookie: { ...DEFAULT_COOKIE_CONFIG },
};

const normalizeSettings = (value: Partial<AppSettings>): AppSettings => {
  const downloadConcurrency =
    typeof value.downloadConcurrency === 'number' &&
    Number.isFinite(value.downloadConcurrency) &&
    value.downloadConcurrency >= 1 &&
    value.downloadConcurrency <= 5
      ? value.downloadConcurrency
      : DEFAULT_SETTINGS.downloadConcurrency;
  const maxDownloadSpeed =
    typeof value.maxDownloadSpeed === 'number' &&
    Number.isFinite(value.maxDownloadSpeed) &&
    value.maxDownloadSpeed >= 0 &&
    value.maxDownloadSpeed <= 150
      ? value.maxDownloadSpeed
      : null;
  const language = value.language === 'en' ? 'en' : 'zh';
  const theme = value.theme === 'light' || value.theme === 'dark' ? value.theme : 'system';
  const downloadDir = typeof value.downloadDir === 'string' ? value.downloadDir : '';
  const proxyUrl = typeof value.proxyUrl === 'string' ? value.proxyUrl : '';

  const normalizeCookie = (c: unknown): CookieConfig => {
    const value = typeof c === 'object' && c !== null ? (c as Partial<CookieConfig>) : {};
    return {
      enabled: !!value.enabled,
      source: value.source === 'file' ? 'file' : 'browser',
      browser: typeof value.browser === 'string' ? value.browser : '',
      file: typeof value.file === 'string' ? value.file : '',
    };
  };

  const cookie = normalizeCookie(value.cookie);

  return {
    ...DEFAULT_SETTINGS,
    downloadConcurrency,
    maxDownloadSpeed,
    language,
    theme,
    downloadDir,
    proxyUrl,
    cookie,
  };
};

interface SettingState {
  defaultDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  theme: AppTheme;
  proxyUrl: string;
  cookie: CookieConfig;

  // Actions
  setDefaultDir: (dir: string) => void;
  setDownloadConcurrency: (value: number) => void;
  setMaxDownloadSpeed: (value: number | null) => void;
  setLanguage: (value: AppLanguage) => void;
  setTheme: (value: AppTheme) => void;
  setProxyUrl: (value: string) => void;
  setCookie: (value: CookieConfig) => void;
  loadSettings: () => void;
}

export const useSettingStore = create<SettingState>((set, get) => ({
  defaultDir: '',
  downloadConcurrency: DEFAULT_SETTINGS.downloadConcurrency,
  maxDownloadSpeed: DEFAULT_SETTINGS.maxDownloadSpeed,
  language: DEFAULT_SETTINGS.language,
  theme: DEFAULT_SETTINGS.theme,
  proxyUrl: DEFAULT_SETTINGS.proxyUrl,
  cookie: DEFAULT_SETTINGS.cookie,

  setDefaultDir: (dir) => {
    set({ defaultDir: dir });
    saveAppSettings(get());
  },
  setDownloadConcurrency: (value) => {
    set({ downloadConcurrency: value });
    saveAppSettings(get());
  },
  setMaxDownloadSpeed: (value) => {
    set({ maxDownloadSpeed: value });
    saveAppSettings(get());
  },
  setLanguage: (value) => {
    set({ language: value });
    saveAppSettings(get());
  },
  setTheme: (value) => {
    set({ theme: value });
    saveAppSettings(get());
  },
  setProxyUrl: (value) => {
    set({ proxyUrl: value });
    saveAppSettings(get());
  },
  setCookie: (value) => {
    set({ cookie: value });
    saveAppSettings(get());
  },
  loadSettings: () => {
    GetSettings()
      .then((settings) => {
        // Backend returns the source of truth
        const normalized = normalizeSettings(settings as unknown as AppSettings);
        set({
          defaultDir: normalized.downloadDir,
          downloadConcurrency: normalized.downloadConcurrency,
          maxDownloadSpeed: normalized.maxDownloadSpeed,
          language: normalized.language,
          theme: normalized.theme,
          proxyUrl: normalized.proxyUrl,
          cookie: normalized.cookie,
        });
      })
      .catch((e) => {
        console.error('Failed to load settings from backend', e);
        // Fallback to local storage
        const settings = loadAppSettings();
        set({
          defaultDir: settings.downloadDir,
          downloadConcurrency: settings.downloadConcurrency,
          maxDownloadSpeed: settings.maxDownloadSpeed,
          language: settings.language,
          theme: settings.theme,
          proxyUrl: settings.proxyUrl,
          cookie: settings.cookie,
        });
      });
  },
}));

function loadAppSettings(): AppSettings {
  try {
    const raw = localStorage.getItem(SETTINGS_STORAGE_KEY);
    if (raw) {
      return normalizeSettings(JSON.parse(raw));
    }
  } catch (e) {
    console.error('Failed to load settings', e);
  }
  return DEFAULT_SETTINGS;
}

function saveAppSettings(state: SettingState) {
  const settings: AppSettings = {
    downloadDir: state.defaultDir,
    downloadConcurrency: state.downloadConcurrency,
    maxDownloadSpeed: state.maxDownloadSpeed,
    language: state.language,
    theme: state.theme,
    proxyUrl: state.proxyUrl,
    cookie: state.cookie,
  };
  localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  UpdateSettings(toWailsSettings(settings));
}

function toWailsSettings(settings: AppSettings): WailsConfig.AppSettings {
  return new WailsConfig.AppSettings({
    ...settings,
    maxDownloadSpeed: settings.maxDownloadSpeed ?? undefined,
  });
}
