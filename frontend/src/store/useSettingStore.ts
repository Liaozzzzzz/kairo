import { create } from 'zustand';
import { UpdateSettings } from '@root/wailsjs/go/main/App';

export type AppLanguage = 'zh' | 'en';

export interface AppSettings {
  downloadDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  proxyUrl: string;
}

const SETTINGS_STORAGE_KEY = 'Kairo.settings';

const DEFAULT_SETTINGS: AppSettings = {
  downloadDir: '',
  downloadConcurrency: 3,
  maxDownloadSpeed: null,
  language: 'zh',
  proxyUrl: '',
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

  return {
    ...DEFAULT_SETTINGS,
    downloadConcurrency,
    maxDownloadSpeed,
    language,
    downloadDir,
    proxyUrl,
  };
};

interface SettingState {
  defaultDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  proxyUrl: string;

  // Actions
  setDefaultDir: (dir: string) => void;
  setDownloadConcurrency: (value: number) => void;
  setMaxDownloadSpeed: (value: number | null) => void;
  setLanguage: (value: AppLanguage) => void;
  setProxyUrl: (value: string) => void;
  loadSettings: () => void;
}

export const useSettingStore = create<SettingState>((set, get) => ({
  defaultDir: '',
  downloadConcurrency: DEFAULT_SETTINGS.downloadConcurrency,
  maxDownloadSpeed: DEFAULT_SETTINGS.maxDownloadSpeed,
  language: DEFAULT_SETTINGS.language,
  proxyUrl: DEFAULT_SETTINGS.proxyUrl,

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
  loadSettings: () => {
    const settings = loadAppSettings();
    set({
      defaultDir: settings.downloadDir,
      downloadConcurrency: settings.downloadConcurrency,
      maxDownloadSpeed: settings.maxDownloadSpeed,
      language: settings.language,
      proxyUrl: settings.proxyUrl,
    });
    UpdateSettings({
      ...settings,
      maxDownloadSpeed: settings.maxDownloadSpeed ?? undefined,
    }).catch(console.error);
  },
}));

const loadAppSettings = (): AppSettings => {
  if (typeof localStorage === 'undefined') {
    return DEFAULT_SETTINGS;
  }
  try {
    const raw = localStorage.getItem(SETTINGS_STORAGE_KEY);
    if (!raw) {
      return DEFAULT_SETTINGS;
    }

    const parsed = JSON.parse(raw) as Partial<AppSettings>;
    return normalizeSettings(parsed);
  } catch {
    return DEFAULT_SETTINGS;
  }
};

const saveAppSettings = (state: SettingState) => {
  if (typeof localStorage === 'undefined') {
    return;
  }
  const settings: AppSettings = {
    downloadDir: state.defaultDir,
    downloadConcurrency: state.downloadConcurrency,
    maxDownloadSpeed: state.maxDownloadSpeed,
    language: state.language,
    proxyUrl: state.proxyUrl,
  };
  localStorage.setItem(SETTINGS_STORAGE_KEY, JSON.stringify(settings));
  UpdateSettings({
    ...settings,
    maxDownloadSpeed: settings.maxDownloadSpeed ?? undefined,
  }).catch(console.error);
};
