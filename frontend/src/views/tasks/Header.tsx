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
    const allTasks = Object.values(tasks);
    const topLevelTasks = allTasks.filter((t) => !t.parent_id);
    const childrenMap: Record<string, typeof allTasks> = {};

    allTasks.forEach((task) => {
      if (task.parent_id) {
        if (!childrenMap[task.parent_id]) {
          childrenMap[task.parent_id] = [];
        }
        childrenMap[task.parent_id].push(task);
      }
    });

    let downloading = 0;
    let completed = 0;
    let failed = 0;

    for (const task of topLevelTasks) {
      const childs = childrenMap[task.id] || [];
      let isDownloading = false;
      let isCompleted = false;
      let isFailed = false;

      if (task.is_playlist) {
        const hasDownloading = childs.some(
          (c) =>
            c.status === TaskStatus.Pending ||
            c.status === TaskStatus.Starting ||
            c.status === TaskStatus.Downloading ||
            c.status === TaskStatus.Merging ||
            c.status === TaskStatus.Paused
        );
        const hasFailed = childs.some((c) => c.status === TaskStatus.Error);

        if (hasDownloading || childs.length === 0) {
          isDownloading = true;
        }

        if (!hasDownloading && hasFailed) {
          isFailed = true;
        }

        if (!hasDownloading && !hasFailed && childs.length > 0) {
          const allCompleted = childs.every((c) => c.status === TaskStatus.Completed);
          if (allCompleted) isCompleted = true;
        }
      } else {
        isDownloading =
          task.status === TaskStatus.Pending ||
          task.status === TaskStatus.Starting ||
          task.status === TaskStatus.Downloading ||
          task.status === TaskStatus.Merging ||
          task.status === TaskStatus.Paused;
        isCompleted = task.status === TaskStatus.Completed;
        isFailed = task.status === TaskStatus.Error;
      }

      if (isDownloading) downloading++;
      if (isCompleted) completed++;
      if (isFailed) failed++;
    }

    return {
      downloading,
      completed,
      failed,
      all: topLevelTasks.length,
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
                <span>{t('tasks.filters.failed')}</span>
                <span className="text-[12px] text-muted-foreground">({counts.failed})</span>
              </span>
            ),
            value: 'failed',
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
