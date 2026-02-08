import { create } from 'zustand';
import { UpdateSettings } from '@root/wailsjs/go/main/App';
import { config as WailsConfig } from '@root/wailsjs/go/models';

export type AppLanguage = 'zh' | 'en';

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
  proxyUrl: string;
  bilibiliCookie: CookieConfig;
  youtubeCookie: CookieConfig;
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
  proxyUrl: '',
  bilibiliCookie: { ...DEFAULT_COOKIE_CONFIG },
  youtubeCookie: { ...DEFAULT_COOKIE_CONFIG },
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

  const bilibiliCookie = normalizeCookie(value.bilibiliCookie);
  const youtubeCookie = normalizeCookie(value.youtubeCookie);

  return {
    ...DEFAULT_SETTINGS,
    downloadConcurrency,
    maxDownloadSpeed,
    language,
    downloadDir,
    proxyUrl,
    bilibiliCookie,
    youtubeCookie,
  };
};

interface SettingState {
  defaultDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  proxyUrl: string;
  bilibiliCookie: CookieConfig;
  youtubeCookie: CookieConfig;

  // Actions
  setDefaultDir: (dir: string) => void;
  setDownloadConcurrency: (value: number) => void;
  setMaxDownloadSpeed: (value: number | null) => void;
  setLanguage: (value: AppLanguage) => void;
  setProxyUrl: (value: string) => void;
  setBilibiliCookie: (value: CookieConfig) => void;
  setYoutubeCookie: (value: CookieConfig) => void;
  loadSettings: () => void;
}

export const useSettingStore = create<SettingState>((set, get) => ({
  defaultDir: '',
  downloadConcurrency: DEFAULT_SETTINGS.downloadConcurrency,
  maxDownloadSpeed: DEFAULT_SETTINGS.maxDownloadSpeed,
  language: DEFAULT_SETTINGS.language,
  proxyUrl: DEFAULT_SETTINGS.proxyUrl,
  bilibiliCookie: DEFAULT_SETTINGS.bilibiliCookie,
  youtubeCookie: DEFAULT_SETTINGS.youtubeCookie,

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
  setProxyUrl: (value) => {
    set({ proxyUrl: value });
    saveAppSettings(get());
  },
  setBilibiliCookie: (value) => {
    set({ bilibiliCookie: value });
    saveAppSettings(get());
  },
  setYoutubeCookie: (value) => {
    set({ youtubeCookie: value });
    saveAppSettings(get());
  },
  loadSettings: () => {
    const settings = loadAppSettings();
    set({
      defaultDir: settings.downloadDir,
      downloadConcurrency: settings.downloadConcurrency,
      maxDownloadSpeed: settings.maxDownloadSpeed,
      language: settings.language,
      proxyUrl: settings.proxyUrl,
      bilibiliCookie: settings.bilibiliCookie,
      youtubeCookie: settings.youtubeCookie,
    });
    // Sync to backend
    UpdateSettings(toWailsSettings(settings));
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
    proxyUrl: state.proxyUrl,
    bilibiliCookie: state.bilibiliCookie,
    youtubeCookie: state.youtubeCookie,
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
