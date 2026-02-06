import { Button, Segmented } from 'antd';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface HeaderProps {
  onOpenAdd: () => void;
  filter: string;
  onFilterChange: (filter: string) => void;
}

export function Header({ onOpenAdd, filter, onFilterChange }: HeaderProps) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-8">
        <div className="space-y-1">
          <h1 className="text-3xl font-bold tracking-tight text-foreground">
            {t('downloads.title')}
          </h1>
          <p className="text-[13px] text-muted-foreground font-medium">{t('downloads.subtitle')}</p>
        </div>

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
      <Button
        type="primary"
        size="large"
        icon={<PlusOutlined className="w-4 h-4" />}
        onClick={onOpenAdd}
        style={{
          borderRadius: '8px', // Apple style rounded rect, not full pill
          height: '36px',
          paddingLeft: '16px',
          paddingRight: '16px',
          fontSize: '13px',
          fontWeight: 500,
          boxShadow: '0 1px 2px rgba(0,0,0,0.05)',
        }}
      >
        {t('downloads.newTask')}
      </Button>
    </div>
  );
}
