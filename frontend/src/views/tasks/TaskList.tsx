import { useMemo } from 'react';
import { useAppStore } from '@/store/useAppStore';
import { TaskItem } from './TaskItem';
import { PlusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { TaskStatus } from '@/data/variables';

interface TaskListProps {
  onViewLog: (taskId: string) => void;
  filter: string;
}

export function TaskList({ onViewLog, filter }: TaskListProps) {
  const { t } = useTranslation();
  const tasks = useAppStore((state) => state.tasks);

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
      <div className="flex flex-col items-center justify-center py-20 rounded-2xl border-2 border-dashed border-black/5 bg-card/50 text-muted-foreground">
        <div className="flex items-center justify-center w-12 h-12 rounded-full bg-black/5 mb-4 text-black/40">
          <PlusOutlined className="w-6 h-6" />
        </div>
        <div className="text-[15px] font-medium text-black/60">{t('tasks.noDownloads')}</div>
        <div className="text-[13px] text-black/40 mt-1">{t('tasks.startDownloading')}</div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <div className="grid gap-3">
        {taskList.map((task) => (
          <TaskItem key={task.id} task={task} onViewLog={() => onViewLog(task.id)} />
        ))}
      </div>
    </div>
  );
}
