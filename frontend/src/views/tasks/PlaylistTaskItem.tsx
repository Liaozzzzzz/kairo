import { useState, useMemo } from 'react';
import { Card, Badge, Modal, Dropdown, MenuProps } from 'antd';
import {
  DownOutlined,
  RightOutlined,
  DeleteOutlined,
  FolderOutlined,
  FileTextOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Task } from '@/types';
import { TaskStatus } from '@/data/variables';
import { TaskItem } from './TaskItem';
import { useTaskStore } from '@/store/useTaskStore';
import { OpenTaskDir } from '@root/wailsjs/go/main/App';

interface PlaylistTaskItemProps {
  task: Task;
  childrenTasks: Task[];
  onViewLog: (taskId: string) => void;
}

export function PlaylistTaskItem({ task, childrenTasks, onViewLog }: PlaylistTaskItemProps) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);
  const deleteTask = useTaskStore((state) => state.deleteTask);

  const completedCount = childrenTasks.filter((t) => t.status === TaskStatus.Completed).length;
  const failedCount = childrenTasks.filter((t) => t.status === TaskStatus.Error).length;
  const totalCount = childrenTasks.length;

  const siteLabel = useMemo(() => {
    const match = task.url.match(/https?:\/\/(?:www\.)?([a-z0-9-]+)\./i);
    const rawLabel = match?.[1];
    if (!rawLabel) return '';
    return `${rawLabel[0].toUpperCase()}${rawLabel.slice(1)}`;
  }, [task.url]);

  const confirmDelete = () => {
    Modal.confirm({
      centered: true,
      title: t('tasks.confirmDelete.title'),
      content: t('tasks.confirmDelete.content'),
      okText: t('tasks.confirmDelete.ok'),
      cancelText: t('tasks.confirmDelete.cancel'),
      okButtonProps: { danger: true },
      onOk: () => deleteTask(task.id, false),
    });
  };

  const menuItems: MenuProps['items'] = [
    {
      key: 'open',
      label: t('tasks.contextMenu.openLocation'),
      icon: <FolderOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: () => OpenTaskDir(task.id),
    },
    {
      key: 'details',
      label: t('tasks.contextMenu.details'),
      icon: <FileTextOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: () => onViewLog(task.id),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: t('tasks.contextMenu.delete'),
      icon: <DeleteOutlined className="w-4 h-4 mt-[-2px]" />,
      danger: true,
      onClick: confirmDelete,
    },
  ];

  return (
    <Badge.Ribbon text={siteLabel}>
      <Card
        hoverable
        variant="borderless"
        styles={{ body: { padding: '0' } }}
        className="overflow-hidden"
      >
        <Dropdown menu={{ items: menuItems }} trigger={['contextMenu']}>
          <div
            className="flex items-center gap-3 p-4 cursor-pointer hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
            onClick={() => setExpanded(!expanded)}
          >
            <div className="text-muted-foreground text-sm">
              {expanded ? <DownOutlined /> : <RightOutlined />}
            </div>

            <div className="flex-1 min-w-0">
              <div
                className="font-semibold text-[15px] truncate text-foreground mb-1"
                title={task.title || task.url}
              >
                {task.title || task.url}
              </div>
              <div className="text-xs text-muted-foreground flex items-center gap-1">
                <span>{t('tasks.playlist.statusPrefix')}</span>
                <span className="text-green-600 dark:text-green-400">
                  {t('tasks.playlist.progress', { completed: completedCount, total: totalCount })}
                </span>
                {failedCount > 0 && (
                  <>
                    <span>·</span>
                    <span className="text-red-500 dark:text-red-400">
                      {t('tasks.playlist.failed', { count: failedCount })}
                    </span>
                  </>
                )}
              </div>
            </div>
            <div className="w-8 h-8"></div>
          </div>
        </Dropdown>

        {expanded && (
          <div className="bg-gray-50/50 dark:bg-black/20 border-t border-gray-100 dark:border-white/5 p-2 space-y-2 pl-8">
            {childrenTasks.map((child) => (
              <TaskItem
                key={child.id}
                task={child}
                showSiteLabel={false}
                onViewLog={() => onViewLog(child.id)}
              />
            ))}
          </div>
        )}
      </Card>
    </Badge.Ribbon>
  );
}
