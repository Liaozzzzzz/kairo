import { create } from 'zustand';
import { UpdateSettings, GetSettings } from '@root/wailsjs/go/main/App';
import { config as WailsConfig } from '@root/wailsjs/go/models';
import { DEFAULT_THEME_COLOR } from '@/data/themeColors';

export type AppLanguage = 'zh' | 'en';
export type AppTheme = 'light' | 'dark' | 'system';

export interface CookieConfig {
  enabled: boolean;
  source: 'browser' | 'file';
  browser: string;
  file: string;
}

export interface AIConfig {
  enabled: boolean;
  provider: string;
  baseUrl: string;
  apiKey: string;
  modelName: string;
  prompt: string;
  language: string;
}

export interface WhisperAIConfig {
  enabled: boolean;
  provider: string;
  baseUrl: string;
  apiKey: string;
  modelName: string;
  prompt: string;
  language: string;
}

export interface AppSettings {
  downloadDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  theme: AppTheme;
  themeColor: string;
  proxyUrl: string;
  userAgent: string;
  referer: string;
  geoBypass: boolean;
  cookie: CookieConfig;
  ai: AIConfig;
  whisperAi: WhisperAIConfig;
  rssCheckInterval: number;
}

const SETTINGS_STORAGE_KEY = 'Kairo.settings';

const DEFAULT_COOKIE_CONFIG: CookieConfig = {
  enabled: false,
  source: 'browser',
  browser: '',
  file: '',
};

const DEFAULT_AI_CONFIG: AIConfig = {
  enabled: false,
  provider: 'openai',
  baseUrl: 'https://api.openai.com/v1',
  apiKey: '',
  modelName: 'gpt-3.5-turbo',
  prompt: '',
  language: 'zh',
};

const DEFAULT_WHISPER_AI_CONFIG: WhisperAIConfig = {
  enabled: false,
  provider: 'openai',
  baseUrl: 'https://api.openai.com/v1',
  apiKey: '',
  modelName: 'whisper-1',
  prompt: '',
  language: 'zh',
};

const DEFAULT_SETTINGS: AppSettings = {
  downloadDir: '',
  downloadConcurrency: 3,
  maxDownloadSpeed: null,
  language: 'zh',
  theme: 'system',
  themeColor: DEFAULT_THEME_COLOR,
  proxyUrl: '',
  userAgent: '',
  referer: '',
  geoBypass: true,
  cookie: { ...DEFAULT_COOKIE_CONFIG },
  ai: { ...DEFAULT_AI_CONFIG },
  whisperAi: { ...DEFAULT_WHISPER_AI_CONFIG },
  rssCheckInterval: 60,
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
  const themeColor = typeof value.themeColor === 'string' ? value.themeColor : DEFAULT_THEME_COLOR;
  const downloadDir = typeof value.downloadDir === 'string' ? value.downloadDir : '';
  const proxyUrl = typeof value.proxyUrl === 'string' ? value.proxyUrl : '';
  const userAgent = typeof value.userAgent === 'string' ? value.userAgent : '';
  const referer = typeof value.referer === 'string' ? value.referer : '';
  const geoBypass = typeof value.geoBypass === 'boolean' ? value.geoBypass : true;
  const rssCheckInterval =
    typeof value.rssCheckInterval === 'number' &&
    Number.isFinite(value.rssCheckInterval) &&
    value.rssCheckInterval >= 1
      ? value.rssCheckInterval
      : 60;

  const normalizeCookie = (c: unknown): CookieConfig => {
    const value = typeof c === 'object' && c !== null ? (c as Partial<CookieConfig>) : {};
    return {
      enabled: !!value.enabled,
      source: value.source === 'file' ? 'file' : 'browser',
      browser: typeof value.browser === 'string' ? value.browser : '',
      file: typeof value.file === 'string' ? value.file : '',
    };
  };

  const normalizeAI = (a: unknown): AIConfig => {
    const value = typeof a === 'object' && a !== null ? (a as Partial<AIConfig>) : {};
    return {
      enabled: !!value.enabled,
      provider: typeof value.provider === 'string' ? value.provider : 'openai',
      baseUrl: typeof value.baseUrl === 'string' ? value.baseUrl : 'https://api.openai.com/v1',
      apiKey: typeof value.apiKey === 'string' ? value.apiKey : '',
      modelName: typeof value.modelName === 'string' ? value.modelName : 'gpt-3.5-turbo',
      prompt: typeof value.prompt === 'string' ? value.prompt : '',
      language: typeof value.language === 'string' ? value.language : 'zh',
    };
  };

  const normalizeWhisperAI = (a: unknown): WhisperAIConfig => {
    const value = typeof a === 'object' && a !== null ? (a as Partial<WhisperAIConfig>) : {};
    return {
      enabled: !!value.enabled,
      provider: typeof value.provider === 'string' ? value.provider : 'openai',
      baseUrl: typeof value.baseUrl === 'string' ? value.baseUrl : 'https://api.openai.com/v1',
      apiKey: typeof value.apiKey === 'string' ? value.apiKey : '',
      modelName: typeof value.modelName === 'string' ? value.modelName : 'whisper-1',
      prompt: typeof value.prompt === 'string' ? value.prompt : '',
      language: typeof value.language === 'string' ? value.language : 'zh',
    };
  };

  const cookie = normalizeCookie(value.cookie);
  const ai = normalizeAI(value.ai);
  const whisperAi = normalizeWhisperAI(value.whisperAi);

  return {
    ...DEFAULT_SETTINGS,
    downloadConcurrency,
    maxDownloadSpeed,
    language,
    theme,
    themeColor,
    downloadDir,
    proxyUrl,
    userAgent,
    referer,
    geoBypass,
    cookie,
    ai,
    whisperAi,
    rssCheckInterval,
  };
};

interface SettingState {
  defaultDir: string;
  downloadConcurrency: number;
  maxDownloadSpeed: number | null;
  language: AppLanguage;
  theme: AppTheme;
  themeColor: string;
  proxyUrl: string;
  userAgent: string;
  referer: string;
  geoBypass: boolean;
  cookie: CookieConfig;
  ai: AIConfig;
  whisperAi: WhisperAIConfig;
  rssCheckInterval: number;

  // Actions
  setDefaultDir: (dir: string) => void;
  setDownloadConcurrency: (value: number) => void;
  setMaxDownloadSpeed: (value: number | null) => void;
  setLanguage: (value: AppLanguage) => void;
  setTheme: (value: AppTheme) => void;
  setThemeColor: (value: string) => void;
  setProxyUrl: (value: string) => void;
  setUserAgent: (value: string) => void;
  setReferer: (value: string) => void;
  setGeoBypass: (value: boolean) => void;
  setCookie: (value: CookieConfig) => void;
  setAI: (value: AIConfig) => void;
  setWhisperAI: (value: WhisperAIConfig) => void;
  setRSSCheckInterval: (value: number) => void;
  loadSettings: () => void;
}

export const useSettingStore = create<SettingState>((set, get) => ({
  defaultDir: '',
  downloadConcurrency: DEFAULT_SETTINGS.downloadConcurrency,
  maxDownloadSpeed: DEFAULT_SETTINGS.maxDownloadSpeed,
  language: DEFAULT_SETTINGS.language,
  theme: DEFAULT_SETTINGS.theme,
  themeColor: DEFAULT_SETTINGS.themeColor,
  proxyUrl: DEFAULT_SETTINGS.proxyUrl,
  userAgent: DEFAULT_SETTINGS.userAgent,
  referer: DEFAULT_SETTINGS.referer,
  geoBypass: DEFAULT_SETTINGS.geoBypass,
  cookie: DEFAULT_SETTINGS.cookie,
  ai: DEFAULT_SETTINGS.ai,
  whisperAi: DEFAULT_SETTINGS.whisperAi,
  rssCheckInterval: DEFAULT_SETTINGS.rssCheckInterval,

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
  setThemeColor: (value) => {
    set({ themeColor: value });
    saveAppSettings(get());
  },
  setProxyUrl: (value) => {
    set({ proxyUrl: value });
    saveAppSettings(get());
  },
  setUserAgent: (value) => {
    set({ userAgent: value });
    saveAppSettings(get());
  },
  setReferer: (value) => {
    set({ referer: value });
    saveAppSettings(get());
  },
  setGeoBypass: (value) => {
    set({ geoBypass: value });
    saveAppSettings(get());
  },
  setCookie: (value) => {
    set({ cookie: value });
    saveAppSettings(get());
  },
  setAI: (value) => {
    set({ ai: value });
    saveAppSettings(get());
  },
  setWhisperAI: (value) => {
    set({ whisperAi: value });
    saveAppSettings(get());
  },
  setRSSCheckInterval: (value) => {
    set({ rssCheckInterval: value });
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
          themeColor: normalized.themeColor,
          proxyUrl: normalized.proxyUrl,
          userAgent: normalized.userAgent,
          referer: normalized.referer,
          geoBypass: normalized.geoBypass,
          cookie: normalized.cookie,
          rssCheckInterval: normalized.rssCheckInterval,
          ai: normalized.ai,
          whisperAi: normalized.whisperAi,
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
          themeColor: settings.themeColor,
          proxyUrl: settings.proxyUrl,
          userAgent: settings.userAgent,
          referer: settings.referer,
          geoBypass: settings.geoBypass,
          cookie: settings.cookie,
          rssCheckInterval: settings.rssCheckInterval,
          whisperAi: settings.whisperAi,
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
    themeColor: state.themeColor,
    proxyUrl: state.proxyUrl,
    userAgent: state.userAgent,
    referer: state.referer,
    geoBypass: state.geoBypass,
    cookie: state.cookie,
    rssCheckInterval: state.rssCheckInterval,
    ai: state.ai,
    whisperAi: state.whisperAi,
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
