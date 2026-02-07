import { useMemo } from 'react';
import { Segmented } from 'antd';
import { useTranslation } from 'react-i18next';
import PageHeader from '@/components/PageHeader';
import { TaskStatus } from '@/data/variables';
import { useTaskStore } from '@/store/useTaskStore';

interface HeaderProps {
  filter: string;
  onFilterChange: (filter: string) => void;
}

export function Header({ filter, onFilterChange }: HeaderProps) {
  const { t } = useTranslation();
  const tasks = useTaskStore((state) => state.tasks);

  const counts = useMemo(() => {
    const values = Object.values(tasks);
    let downloading = 0;
    let completed = 0;

    for (const task of values) {
      if (
        task.status === TaskStatus.Pending ||
        task.status === TaskStatus.Starting ||
        task.status === TaskStatus.Downloading ||
        task.status === TaskStatus.Merging ||
        task.status === TaskStatus.Paused ||
        task.status === TaskStatus.Error
      ) {
        downloading += 1;
      }
      if (task.status === TaskStatus.Completed) {
        completed += 1;
      }
    }

    return {
      downloading,
      completed,
      all: values.length,
    };
  }, [tasks]);

  return (
    <div className="flex items-center gap-8">
      <PageHeader title={t('tasks.title')} subtitle={t('tasks.subtitle')} />

      <Segmented
        value={filter}
        onChange={(value) => onFilterChange(value as string)}
        options={[
          {
            label: (
              <span className="px-2 text-[14px] font-medium flex items-center gap-1">
                <span>{t('tasks.filters.downloading')}</span>
                <span className="text-[12px] text-muted-foreground">({counts.downloading})</span>
              </span>
            ),
            value: 'downloading',
          },
          {
            label: (
              <span className="px-2 text-[14px] font-medium flex items-center gap-1">
                <span>{t('tasks.filters.completed')}</span>
                <span className="text-[12px] text-muted-foreground">({counts.completed})</span>
              </span>
            ),
            value: 'completed',
          },
          {
            label: (
              <span className="px-2 text-[14px] font-medium flex items-center gap-1">
                <span>{t('tasks.filters.all')}</span>
                <span className="text-[12px] text-muted-foreground">({counts.all})</span>
              </span>
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
