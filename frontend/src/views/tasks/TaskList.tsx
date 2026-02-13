import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from 'antd';
import { useTaskStore } from '@/store/useTaskStore';
import { PlusOutlined } from '@ant-design/icons';
import { MenuItemKey, TaskStatus } from '@/data/variables';
import { TaskItem } from './TaskItem';
import { PlaylistTaskItem } from './PlaylistTaskItem';
import { useAppStore } from '@/store/useAppStore';
import { Task } from '@/types';

interface TaskListProps {
  onViewLog: (taskId: string) => void;
  filter: string;
}

export function TaskList({ onViewLog, filter }: TaskListProps) {
  const { t } = useTranslation();
  const tasks = useTaskStore((state) => state.tasks);
  const setActiveTab = useAppStore((state) => state.setActiveTab);

  const { topLevelTasks, childrenMap } = useMemo(() => {
    const allTasks = Object.values(tasks);
    // Sort by ID descending (newest first)
    allTasks.sort((a, b) => b.id.localeCompare(a.id));

    const topLevel: Task[] = [];
    const children: Record<string, Task[]> = {};

    allTasks.forEach((task) => {
      if (task.parent_id) {
        if (!children[task.parent_id]) {
          children[task.parent_id] = [];
        }
        children[task.parent_id].push(task);
      } else {
        topLevel.push(task);
      }
    });

    // Sort children by ID ascending (creation order for playlist items)
    Object.keys(children).forEach((key) => {
      children[key].sort((a, b) => a.id.localeCompare(b.id));
    });

    return { topLevelTasks: topLevel, childrenMap: children };
  }, [tasks]);

  const taskList = useMemo(() => {
    return topLevelTasks.filter((task) => {
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

        // Only categorize as failed if no tasks are downloading/active AND there are failed tasks
        if (!hasDownloading && hasFailed) {
          isFailed = true;
        }

        if (!hasDownloading && !hasFailed && childs.length > 0) {
          // If all children are completed
          isCompleted = childs.every((c) => c.status === TaskStatus.Completed);
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

      if (filter === 'downloading') {
        return isDownloading;
      }
      if (filter === 'completed') {
        return isCompleted;
      }
      if (filter === 'failed') {
        return isFailed;
      }
      return true;
    });
  }, [topLevelTasks, childrenMap, filter]);

  if (taskList.length === 0) {
    return (
      <button className="w-full flex flex-col items-center justify-center py-20 rounded-2xl border-2 border-dashed border-border bg-card/50 text-muted-foreground">
        <div className="flex items-center justify-center w-12 h-12 rounded-full bg-muted mb-4 text-muted-foreground">
          <PlusOutlined className="text-xl" />
        </div>
        <div className="text-[15px] font-medium text-muted-foreground">
          {t('tasks.noDownloads')}
        </div>
        <div className="flex items-center gap-1.5 mt-2 text-sm text-muted-foreground/60">
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
      {taskList.map((task) => {
        if (task.is_playlist) {
          return (
            <PlaylistTaskItem
              key={task.id}
              task={task}
              childrenTasks={childrenMap[task.id] || []}
              onViewLog={onViewLog}
            />
          );
        }
        return <TaskItem key={task.id} task={task} onViewLog={() => onViewLog(task.id)} />;
      })}
    </div>
  );
}
