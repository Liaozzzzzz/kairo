import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from 'antd';
import { useTaskStore } from '@/store/useTaskStore';
import { PlusOutlined } from '@ant-design/icons';
import { MenuItemKey, TaskStatus } from '@/data/variables';
import { TaskItem } from './TaskItem';
import { useAppStore } from '@/store/useAppStore';

interface TaskListProps {
  onViewLog: (taskId: string) => void;
  filter: string;
}

export function TaskList({ onViewLog, filter }: TaskListProps) {
  const { t } = useTranslation();
  const tasks = useTaskStore((state) => state.tasks);
  const setActiveTab = useAppStore((state) => state.setActiveTab);

  const taskList = useMemo(() => {
    return Object.values(tasks)
      .filter((task) => {
        if (filter === 'downloading') {
          return (
            task.status === TaskStatus.Pending ||
            task.status === TaskStatus.Starting ||
            task.status === TaskStatus.Downloading ||
            task.status === TaskStatus.Merging ||
            task.status === TaskStatus.Paused ||
            task.status === TaskStatus.Error
          );
        }
        if (filter === 'completed') {
          return task.status === TaskStatus.Completed;
        }
        return true;
      })
      .reverse();
  }, [tasks, filter]);

  if (taskList.length === 0) {
    return (
      <button className="w-full flex flex-col items-center justify-center py-20 rounded-2xl border-2 border-dashed border-black/5 bg-card/50 text-muted-foreground">
        <div className="flex items-center justify-center w-12 h-12 rounded-full bg-black/5 mb-4 text-black/40">
          <PlusOutlined className="text-3xl" />
        </div>
        <div className="text-[15px] font-medium text-black/60">{t('tasks.noDownloads')}</div>
        <div className="flex items-center justify-center gap-1 text-[13px] text-black/40 mt-1">
          <span>{t('tasks.startDownloadingPrefix')}</span>
          <Button className="px-0" type="link" onClick={() => setActiveTab(MenuItemKey.Downloads)}>
            {t('tasks.startDownloading')}
          </Button>
          <span>{t('tasks.startDownloadingSuffix')}</span>
        </div>
      </button>
    );
  }

  return (
    <div className="space-y-4">
      {taskList.map((task) => (
        <TaskItem key={task.id} task={task} onViewLog={() => onViewLog(task.id)} />
      ))}
    </div>
  );
}
