export const parseSize = (s: string) => {
  if (!s) return 0;
  const m = s.match(/([\d.]+)\s*([A-Za-z]+)/);
  if (!m) return 0;
  const u = m[2].toUpperCase();
  const p = ['B', 'K', 'M', 'G', 'T'].findIndex((x) => u.startsWith(x));
  return parseFloat(m[1]) * Math.pow(1024, p > 0 ? p : 0);
};

export const formatSize = (b: number) => {
  if (!b) return '~';
  const u = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
  const i = Math.floor(Math.log(b) / Math.log(1024));
  return `${(b / Math.pow(1024, i)).toFixed(2)} ${u[i]}`;
};
