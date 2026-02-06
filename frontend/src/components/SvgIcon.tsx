import { useMemo } from 'react';

interface SvgIconProps extends React.SVGProps<SVGSVGElement> {
  name: string;
  prefix?: string;
}

export default function SvgIcon({ name, prefix = 'icon', className, ...props }: SvgIconProps) {
  const symbolId = useMemo(() => `#${prefix}-${name}`, [prefix, name]);
  return (
    <svg aria-hidden="true" className={className} {...props}>
      <use href={symbolId} />
    </svg>
  );
}
