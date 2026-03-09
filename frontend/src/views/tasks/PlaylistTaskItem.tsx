import { useState, useMemo } from 'react';
import { Card, Badge, Tag } from 'antd';
import { DownOutlined, RightOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { Task } from '@/types';
import { TaskStatus, SourceType } from '@/data/variables';
import { TaskItem } from './TaskItem';
import { useCategoryStore } from '@/store/useCategoryStore';
import { useSettingStore } from '@/store/useSettingStore';
import { useStore } from 'zustand';

interface PlaylistTaskItemProps {
  task: Task;
  childrenTasks: Task[];
  onViewLog: (taskId: string) => void;
}

export function PlaylistTaskItem({ task, childrenTasks, onViewLog }: PlaylistTaskItemProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const categories = useStore(useCategoryStore, (state) => state.categories);
  const themeColor = useStore(useSettingStore, (state) => state.themeColor);

  const completedCount = childrenTasks.filter((t) => t.status === TaskStatus.Completed).length;
  const failedCount = childrenTasks.filter((t) => t.status === TaskStatus.Error).length;
  const totalCount = childrenTasks.length;

  const siteLabel = useMemo(() => {
    const match = task.url.match(/https?:\/\/(?:www\.)?([a-z0-9-]+)\./i);
    const rawLabel = match?.[1];
    if (!rawLabel) return '';
    return `${rawLabel[0].toUpperCase()}${rawLabel.slice(1)}`;
  }, [task.url]);

  const category = useMemo(() => {
    if (!task.category_id) return null;
    return categories?.find((c) => c.id === task.category_id);
  }, [task.category_id, categories]);

  return (
    <Badge.Ribbon text={siteLabel}>
      <Card
        hoverable
        variant="borderless"
        styles={{ body: { padding: '0' } }}
        className="overflow-hidden"
      >
        <div
          className="flex items-center gap-3 p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
          onClick={() => setExpanded(!expanded)}
        >
          <div className="text-muted-foreground text-sm">
            {expanded ? <DownOutlined /> : <RightOutlined />}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-1.5 mb-1" title={task.title || task.url}>
              {category && (
                <Tag color={themeColor} className="py-0">
                  {category.name}
                </Tag>
              )}
              <div className="font-semibold text-[15px] truncate text-foreground">
                {task.title || task.url}
              </div>
            </div>
            <div className="text-xs text-muted-foreground flex items-center gap-1">
              <span>
                {task.source_type === SourceType.RSS
                  ? t('tasks.playlist.rssPrefix')
                  : t('tasks.playlist.statusPrefix')}
              </span>
              {totalCount > 0 && completedCount === totalCount ? (
                <span className="text-green-600 dark:text-green-400">
                  {t('tasks.playlist.allCompleted')}
                </span>
              ) : totalCount > 0 && failedCount === totalCount ? (
                <span className="text-red-500 dark:text-red-400">
                  {t('tasks.playlist.allFailed')}
                </span>
              ) : (
                <>
                  <span className="text-green-600 dark:text-green-400">
                    {t('tasks.playlist.progress', {
                      completed: completedCount,
                      total: totalCount,
                    })}
                  </span>
                  {failedCount > 0 && (
                    <>
                      <span>·</span>
                      <span className="text-red-500 dark:text-red-400">
                        {failedCount}
                        &ensp;
                        {t('tasks.playlist.partiallyFailed')}
                      </span>
                    </>
                  )}
                </>
              )}
              {task.created_at && (
                <>
                  <span>·</span>
                  <span>{dayjs.unix(task.created_at).format('YYYY-MM-DD HH:mm')}</span>
                </>
              )}
            </div>
          </div>
          <div className="w-8 h-8"></div>
        </div>

        {expanded && (
          <div className="bg-gray-50/50 dark:bg-black/20 border-t border-gray-100 dark:border-white/5 p-2 space-y-2 pl-4">
            {childrenTasks.map((child) => (
              <TaskItem
                key={child.id}
                task={child}
                showSiteLabel={false}
                showCategoryTag={false}
                onViewLog={() => onViewLog(child.id)}
              />
            ))}
          </div>
        )}
      </Card>
    </Badge.Ribbon>
  );
}
