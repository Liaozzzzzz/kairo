import { Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';

interface HeaderProps {
  filter: string;
  onFilterChange: (filter: string) => void;
}

export function Header({ filter, onFilterChange }: HeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-8">
      <PageHeader title={t('downloads.title')} subtitle={t('downloads.subtitle')} />

      <Segmented
        value={filter}
        onChange={(value) => onFilterChange(value as string)}
        options={[
          {
            label: (
              <span className="px-2 text-[13px] font-medium">
                {t('downloads.filters.downloading')}
              </span>
            ),
            value: 'downloading',
          },
          {
            label: (
              <span className="px-2 text-[13px] font-medium">
                {t('downloads.filters.completed')}
              </span>
            ),
            value: 'completed',
          },
          {
            label: (
              <span className="px-2 text-[13px] font-medium">{t('downloads.filters.all')}</span>
            ),
            value: 'all',
          },
        ]}
        size="large"
        className="font-medium"
      />
    </div>
  );
}
