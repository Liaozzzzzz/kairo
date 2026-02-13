// Convert Hex to HSL
function hexToHSL(hex: string): { h: number; s: number; l: number } {
  let r = 0,
    g = 0,
    b = 0;
  if (hex.length === 4) {
    r = parseInt('0x' + hex[1] + hex[1]);
    g = parseInt('0x' + hex[2] + hex[2]);
    b = parseInt('0x' + hex[3] + hex[3]);
  } else if (hex.length === 7) {
    r = parseInt('0x' + hex[1] + hex[2]);
    g = parseInt('0x' + hex[3] + hex[4]);
    b = parseInt('0x' + hex[5] + hex[6]);
  }
  r /= 255;
  g /= 255;
  b /= 255;
  const cmin = Math.min(r, g, b),
    cmax = Math.max(r, g, b),
    delta = cmax - cmin;
  let h = 0,
    s = 0,
    l = 0;

  if (delta === 0) h = 0;
  else if (cmax === r) h = ((g - b) / delta) % 6;
  else if (cmax === g) h = (b - r) / delta + 2;
  else h = (r - g) / delta + 4;

  h = Math.round(h * 60);
  if (h < 0) h += 360;

  l = (cmax + cmin) / 2;
  s = delta === 0 ? 0 : delta / (1 - Math.abs(2 * l - 1));
  s = +(s * 100).toFixed(1);
  l = +(l * 100).toFixed(1);

  return { h, s, l };
}

// Convert HSL to Hex
function hslToHex(h: number, s: number, l: number): string {
  s /= 100;
  l /= 100;

  const c = (1 - Math.abs(2 * l - 1)) * s;
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1));
  const m = l - c / 2;
  let r = 0,
    g = 0,
    b = 0;

  if (0 <= h && h < 60) {
    r = c;
    g = x;
    b = 0;
  } else if (60 <= h && h < 120) {
    r = x;
    g = c;
    b = 0;
  } else if (120 <= h && h < 180) {
    r = 0;
    g = c;
    b = x;
  } else if (180 <= h && h < 240) {
    r = 0;
    g = x;
    b = c;
  } else if (240 <= h && h < 300) {
    r = x;
    g = 0;
    b = c;
  } else if (300 <= h && h < 360) {
    r = c;
    g = 0;
    b = x;
  }
  r = Math.round((r + m) * 255);
  g = Math.round((g + m) * 255);
  b = Math.round((b + m) * 255);

  const toHex = (n: number) => {
    const hex = n.toString(16);
    return hex.length === 1 ? '0' + hex : hex;
  };
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`;
}

export interface ThemePalette {
  bodyBg: string;
  siderBg: string;
  containerBg: string;
  elevatedBg: string;
  border: string;
  text: string;
  textSecondary: string;
  itemSelectedBg: string;
}

export function generateLightPalette(brandColor: string): ThemePalette {
  const { h, s } = hexToHSL(brandColor);

  // We keep the Hue of the brand color, but use very high lightness
  // to create a "tinted" light mode.

  // Saturation for backgrounds: 5-8% gives a very subtle tint.
  const bgSat = 6;
  const l = (val: number) => hslToHex(h, bgSat, val);
  const textL = (val: number) => hslToHex(h, 4, val);

  return {
    // Main Body Background (tinted off-white)
    // H, 6%, 96%
    bodyBg: l(96),

    // Sidebar (slightly lighter or white)
    // H, 6%, 98%
    siderBg: l(98),

    // Card/Container (Surface) - Usually white in light mode
    containerBg: '#ffffff',

    // Elevated (Popovers, Dropdowns)
    elevatedBg: '#ffffff',

    // Border
    // H, 6%, 88%
    border: hslToHex(h, 6, 88),

    // Text
    // H, 4%, 15% (Dark grey with slight tint)
    text: textL(15),

    // Secondary Text
    // H, 4%, 45%
    textSecondary: textL(45),

    // Item Selected Background
    // Keep brand saturation but very high lightness (96%)
    itemSelectedBg: hslToHex(h, s, 96),
  };
}

export function generateDarkPalette(brandColor: string): ThemePalette {
  const { h } = hexToHSL(brandColor);

  // We keep the Hue of the brand color, but drastically reduce saturation and lightness
  // to create a "tinted" dark mode.

  // Saturation for backgrounds: 10-15% gives a subtle tint.
  const bgSat = 12;
  const l = (val: number) => hslToHex(h, bgSat, val);
  const textL = (val: number) => hslToHex(h, 10, val);

  return {
    // Deepest background (Main Body)
    // H, 12%, 6%
    bodyBg: l(6),

    // Sidebar (slightly lighter or darker than body, here slightly lighter/tinted)
    // H, 12%, 8%
    siderBg: l(8),

    // Card/Container (Surface)
    // H, 12%, 12%
    containerBg: l(12),

    // Elevated (Popovers, Dropdowns)
    // H, 12%, 16%
    elevatedBg: l(16),

    // Border
    // H, 10%, 20%
    border: hslToHex(h, 10, 20),

    // Text
    // H, 10%, 92% (Off-white with slight tint)
    text: textL(92),

    // Secondary Text
    // H, 5%, 60%
    textSecondary: textL(60),

    // Item Selected Background
    itemSelectedBg: brandColor,
  };
}
