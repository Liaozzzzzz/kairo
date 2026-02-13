import { useEffect } from 'react';
import { theme } from 'antd';
import { ThemePalette } from '@/lib/theme';

function hexToHsl(hex: string): string {
  // Handle empty or invalid input
  if (!hex || typeof hex !== 'string') return '0 0% 0%';

  // Remove hash if present
  hex = hex.replace(/^#/, '');

  // Handle shorthand hex (e.g. "03F")
  if (hex.length === 3) {
    hex = hex
      .split('')
      .map((char) => char + char)
      .join('');
  }

  // Parse r, g, b
  let r = parseInt(hex.substring(0, 2), 16);
  let g = parseInt(hex.substring(2, 4), 16);
  let b = parseInt(hex.substring(4, 6), 16);

  r /= 255;
  g /= 255;
  b /= 255;

  const max = Math.max(r, g, b);
  const min = Math.min(r, g, b);
  let h = 0;
  let s = 0;
  const l = (max + min) / 2;

  if (max !== min) {
    const d = max - min;
    s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
    switch (max) {
      case r:
        h = (g - b) / d + (g < b ? 6 : 0);
        break;
      case g:
        h = (b - r) / d + 2;
        break;
      case b:
        h = (r - g) / d + 4;
        break;
    }
    h /= 6;
  }

  return `${(h * 360).toFixed(1)} ${(s * 100).toFixed(1)}% ${(l * 100).toFixed(1)}%`;
}

interface ThemeSyncProps {
  palette: ThemePalette;
}

export default function ThemeSync({ palette }: ThemeSyncProps) {
  const { token } = theme.useToken();

  useEffect(() => {
    const root = document.documentElement;

    // --- Backgrounds ---
    // Main App Background
    root.style.setProperty('--background', hexToHsl(palette.bodyBg));

    // Card / Content Background
    root.style.setProperty('--card', hexToHsl(palette.containerBg));

    // Popover / Elevated Background
    root.style.setProperty('--popover', hexToHsl(palette.elevatedBg));

    // Sidebar Background
    root.style.setProperty('--sidebar-background', hexToHsl(palette.siderBg));

    // --- Foreground / Text ---
    // Primary Text
    root.style.setProperty('--foreground', hexToHsl(palette.text));
    root.style.setProperty('--card-foreground', hexToHsl(palette.text));
    root.style.setProperty('--popover-foreground', hexToHsl(palette.text));

    // Secondary / Muted Text
    root.style.setProperty('--muted-foreground', hexToHsl(palette.textSecondary));

    // --- Brand Colors ---
    // Primary
    root.style.setProperty('--primary', hexToHsl(token.colorPrimary));
    // AntD primary text is white on primary buttons usually
    root.style.setProperty('--primary-foreground', '0 0% 100%');

    // Secondary (often used for muted backgrounds in shadcn)
    // Map to colorFillSecondary
    root.style.setProperty('--secondary', hexToHsl(token.colorFillSecondary));
    root.style.setProperty('--secondary-foreground', hexToHsl(palette.text));

    // Muted (backgrounds)
    root.style.setProperty('--muted', hexToHsl(token.colorFillTertiary));

    // Accent (hover states etc)
    root.style.setProperty('--accent', hexToHsl(token.colorPrimary));
    root.style.setProperty('--accent-foreground', '0 0% 100%');

    // Destructive (Error)
    root.style.setProperty('--destructive', hexToHsl(token.colorError));
    root.style.setProperty('--destructive-foreground', '0 0% 100%');

    // Success
    root.style.setProperty('--success', hexToHsl(token.colorSuccess));
    root.style.setProperty('--success-foreground', '0 0% 100%');

    // Warning
    root.style.setProperty('--warning', hexToHsl(token.colorWarning));
    root.style.setProperty('--warning-foreground', '0 0% 100%');

    // Info
    root.style.setProperty('--info', hexToHsl(token.colorInfo));
    root.style.setProperty('--info-foreground', '0 0% 100%');

    // --- Typography ---
    root.style.setProperty('--font-sans', token.fontFamily);

    // --- Borders & Inputs ---
    root.style.setProperty('--border', hexToHsl(palette.border));
    root.style.setProperty('--input', hexToHsl(palette.border));
    root.style.setProperty('--separator', hexToHsl(token.colorSplit));

    // Ring (Focus)
    root.style.setProperty('--ring', hexToHsl(token.colorPrimary));

    // --- Radius ---
    // Convert number to px string, assuming token.borderRadius is a number
    root.style.setProperty('--radius', `${token.borderRadius}px`);
  }, [token, palette]);

  return null;
}
