import { useEffect, useState } from 'react';
import { theme as antTheme } from 'antd';
import { useSettingStore } from '@/store/useSettingStore';

export function useTheme() {
  const theme = useSettingStore((state) => state.theme);
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    const root = window.document.documentElement;
    const systemTheme = window.matchMedia('(prefers-color-scheme: dark)');

    const updateTheme = () => {
      const isSystemDark = systemTheme.matches;
      const shouldBeDark = theme === 'dark' || (theme === 'system' && isSystemDark);

      setIsDark(shouldBeDark);

      root.classList.remove('light', 'dark');
      root.classList.add(shouldBeDark ? 'dark' : 'light');
    };

    updateTheme();

    systemTheme.addEventListener('change', updateTheme);
    return () => systemTheme.removeEventListener('change', updateTheme);
  }, [theme]);

  return {
    isDark,
    antAlgorithm: isDark ? antTheme.darkAlgorithm : antTheme.defaultAlgorithm,
  };
}
