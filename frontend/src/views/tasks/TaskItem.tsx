import { ReactNode, useMemo } from 'react';
import { Card, Progress, Dropdown, MenuProps, Badge, Modal, message, Tag, Checkbox } from 'antd';
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  FileTextOutlined,
  FolderOutlined,
  LinkOutlined,
  ReloadOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { Task } from '@/types';
import { TaskStatus } from '@/data/variables';
import {
  PauseTask,
  ResumeTask,
  OpenTaskDir,
  RetryTask,
  AddVideoToLibrary,
} from '@root/wailsjs/go/main/App';
import { useTaskStore } from '@/store/useTaskStore';
import { formatBytes } from '@/lib/utils';
import { useTheme } from '@/hooks/useTheme';
import { ThumbnailImage } from '@/components/ThumbnailImage';
import { useCategoryStore } from '@/store/useCategoryStore';
import { useSettingStore } from '@/store/useSettingStore';
import { useStore } from 'zustand';

interface TaskItemProps {
  task: Task;
  showSiteLabel?: boolean;
  showCategoryTag?: boolean;
  onViewLog: () => void;
}

export function TaskItem({
  task,
  showSiteLabel = true,
  showCategoryTag = true,
  onViewLog,
}: TaskItemProps) {
  const { t } = useTranslation();
  const deleteTask = useTaskStore((state) => state.deleteTask);
  const { isDark } = useTheme();
  const categories = useStore(useCategoryStore, (state) => state.categories);
  const themeColor = useStore(useSettingStore, (state) => state.themeColor);

  const category = useMemo(() => {
    if (!task.category_id || !showCategoryTag) return null;
    return categories?.find((c) => c.id === task.category_id);
  }, [task.category_id, categories, showCategoryTag]);

  const siteLabel = useMemo(() => {
    const match = task.url.match(/https?:\/\/(?:www\.)?([a-z0-9-]+)\./i);
    const rawLabel = match?.[1];
    if (!rawLabel) return '';
    return `${rawLabel[0].toUpperCase()}${rawLabel.slice(1)}`;
  }, [task.url]);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case TaskStatus.Completed:
        return <CheckCircleOutlined className="w-4 h-4 text-green-500" />;
      case TaskStatus.Error:
      case TaskStatus.TrimFailed:
        return <CloseCircleOutlined className="w-4 h-4 text-red-500" />;
      case TaskStatus.Starting:
      case TaskStatus.Merging:
      case TaskStatus.Trimming:
      case TaskStatus.Downloading:
        return (
          <div className="animate-spin h-4 w-4 border-2 border-primary border-t-transparent rounded-full" />
        );
      case TaskStatus.Paused:
        return <PauseCircleOutlined className="w-4 h-4 text-yellow-500" />;
      default:
        return <PlayCircleOutlined className="w-4 h-4" />;
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case TaskStatus.Completed:
        return t('tasks.status.completed');
      case TaskStatus.Error:
        return t('tasks.status.error');
      case TaskStatus.Starting:
        return t('tasks.status.starting');
      case TaskStatus.Downloading:
        return t('tasks.status.downloading');
      case TaskStatus.Merging:
        return t('tasks.status.merging');
      case TaskStatus.Trimming:
        return t('tasks.status.trimming');
      case TaskStatus.TrimFailed:
        return t('tasks.status.trim_failed');
      case TaskStatus.Pending:
        return t('tasks.status.pending');
      case TaskStatus.Paused:
        return t('tasks.status.paused');
      default:
        return status;
    }
  };

  const displayProgress = task.progress;
  const displaySize = task.total_bytes ? formatBytes(task.total_bytes) : undefined;

  const confirmDelete = () => {
    if (task.status === TaskStatus.Merging || task.status === TaskStatus.Trimming) {
      return;
    }
    // 下载成功的任务不显示删除文件选项
    const showDeleteFileOption = task.status !== TaskStatus.Completed;
    let deleteFile = false;

    Modal.confirm({
      centered: true,
      title: t('tasks.confirmDelete.title'),
      content: showDeleteFileOption ? (
        <div className="mt-2">
          <div className="mb-2">{t('tasks.confirmDelete.content')}</div>
          <Checkbox
            onChange={(e) => {
              deleteFile = e.target.checked;
            }}
          >
            {t('tasks.confirmDelete.deleteFile')}
          </Checkbox>
        </div>
      ) : (
        t('tasks.confirmDelete.content')
      ),
      okText: t('tasks.confirmDelete.ok'),
      cancelText: t('tasks.confirmDelete.cancel'),
      okButtonProps: { danger: true },
      onOk: () => deleteTask(task.id, deleteFile),
    });
  };

  const handleAddToLibrary = async () => {
    try {
      const added = await AddVideoToLibrary(task.id);
      if (added) {
        message.success(t('tasks.libraryAdded'));
      } else {
        message.warning(t('tasks.libraryExists'));
      }
    } catch (error) {
      console.error('Failed to add video to library:', error);
      message.error(t('tasks.libraryAddFailed'));
    }
  };

  const menuItems: MenuProps['items'] = [
    {
      key: 'details',
      label: t('tasks.contextMenu.details'),
      icon: <FileTextOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: onViewLog,
    },
    {
      key: 'open',
      label: t('tasks.contextMenu.openLocation'),
      icon: <FolderOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: () => OpenTaskDir(task.id),
    },
    {
      key: 'copy',
      label: t('tasks.contextMenu.copyLink'),
      icon: <LinkOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: () => navigator.clipboard.writeText(task.url),
    },
    ...(task.status === TaskStatus.Completed && task.file_exists
      ? [
          {
            key: 'addToLibrary',
            label: t('tasks.contextMenu.addToLibrary'),
            icon: <CheckCircleOutlined className="w-4 h-4 mt-[-2px]" />,
            onClick: handleAddToLibrary,
          },
        ]
      : []),
    ...(task.status === 'error' || (task.status === 'completed' && !task.file_exists)
      ? [
          {
            key: 'retry',
            label: t('tasks.contextMenu.retry'),
            icon: <ReloadOutlined className="w-4 h-4 mt-[-2px]" />,
            onClick: () => RetryTask(task.id),
          },
        ]
      : []),
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: t('tasks.contextMenu.delete'),
      icon: <DeleteOutlined className="w-4 h-4 mt-[-2px]" />,
      danger: true,
      onClick: () => confirmDelete(),
    },
  ];

  const getStatusTagClass = (status: string) => {
    switch (status) {
      case TaskStatus.Completed:
        return task.file_exists === false
          ? 'bg-gray-100 dark:bg-white/5 text-gray-400 dark:text-muted-foreground border border-gray-200 dark:border-white/10' // Disabled/Missing
          : 'bg-green-50 dark:bg-green-500/10 text-green-600 dark:text-green-400 border border-green-200 dark:border-green-500/20';
      case TaskStatus.Error:
        return 'bg-red-50 dark:bg-red-500/10 text-red-600 dark:text-red-400 border border-red-200 dark:border-red-500/20';
      case TaskStatus.Starting:
      case TaskStatus.Merging:
      case TaskStatus.Trimming:
      case TaskStatus.Downloading:
        return 'bg-primary/10 text-primary border border-primary/20';
      case TaskStatus.Paused:
        return 'bg-yellow-50 dark:bg-yellow-500/10 text-yellow-600 dark:text-yellow-400 border border-yellow-200 dark:border-yellow-500/20';
      default:
        return 'bg-gray-50 dark:bg-white/5 text-gray-500 dark:text-muted-foreground border border-gray-200 dark:border-white/10';
    }
  };

  const getProgressColor = () => {
    if (task.status === TaskStatus.Completed) return '#22c55e'; // green-500
    if (task.status === TaskStatus.Error) return '#ef4444'; // red-500
    if (task.status === TaskStatus.Paused) return '#eab308'; // yellow-500
    return undefined; // default primary
  };

  const isActive = task.status === TaskStatus.Starting || task.status === TaskStatus.Downloading;

  const RibbonWrap = (children: ReactNode) => {
    return showSiteLabel ? (
      <Badge.Ribbon text={showSiteLabel ? siteLabel : ''}>{children}</Badge.Ribbon>
    ) : (
      <>{children}</>
    );
  };

  return (
    <Dropdown menu={{ items: menuItems }} trigger={['contextMenu']}>
      <div>
        {RibbonWrap(
          <Card
            hoverable
            variant="borderless"
            className={`${
              task.status === TaskStatus.Completed && task.file_exists === false
                ? 'opacity-60 grayscale'
                : ''
            }`}
            styles={{ body: { padding: '16px' } }}
          >
            <div className="flex items-center gap-4">
              {/* Thumbnail Column */}
              <div className="flex-shrink-0 w-24 h-16 bg-gray-100 dark:bg-white/5 rounded-md overflow-hidden relative border border-gray-100 dark:border-white/5">
                <ThumbnailImage
                  src={task.thumbnail}
                  referrerPolicy="no-referrer"
                  alt=""
                  width="100%"
                  height="100%"
                />
                <div className="flex items-center justify-center absolute w-6 h-6 -bottom-1 -right-1 bg-white dark:bg-card rounded-full p-0.5 shadow-sm border border-gray-100 dark:border-border scale-75 z-10">
                  {getStatusIcon(task.status)}
                </div>
              </div>

              {/* Main Info */}
              <div className="flex-1 min-w-0 flex flex-col gap-1.5">
                <div className="flex items-center gap-2 min-w-0">
                  {category && (
                    <Tag color={themeColor} className="py-0">
                      {category.name}
                    </Tag>
                  )}
                  <div
                    className="flex-1 min-w-0 font-semibold text-[15px] truncate text-foreground"
                    title={task.title || task.url}
                  >
                    {task.title || task.url}
                  </div>
                  <div
                    className={`shrink-0 text-[11px] font-medium px-2.5 py-0.5 rounded-full uppercase tracking-wide whitespace-nowrap ${getStatusTagClass(
                      task.status
                    )}`}
                  >
                    {getStatusText(task.status)}
                  </div>
                </div>

                <Progress
                  percent={displayProgress}
                  showInfo={false}
                  size="small"
                  strokeColor={getProgressColor()}
                  railColor={isDark ? '#333333' : '#f3f4f6'}
                  className="m-0"
                />
                <div className="flex justify-between text-[11px] font-medium text-muted-foreground items-center">
                  <div className="flex items-center gap-2">
                    <span>{displayProgress.toFixed(1)}%</span>
                    {displaySize && displaySize !== '~' && (
                      <>
                        <span className="text-gray-300 dark:text-muted-foreground/50">•</span>
                        <span>~{displaySize}</span>
                      </>
                    )}
                    {isActive && (
                      <>
                        {task.speed && task.speed !== '~' && (
                          <>
                            <span className="text-gray-300 dark:text-muted-foreground/50">•</span>
                            <span>{task.speed}</span>
                          </>
                        )}
                      </>
                    )}
                    {task.created_at && (
                      <>
                        <span className="text-gray-300 dark:text-muted-foreground/50">•</span>
                        <span>{dayjs.unix(task.created_at).format('YYYY-MM-DD HH:mm')}</span>
                      </>
                    )}
                  </div>
                  <span className="uppercase">{task.quality}</span>
                </div>
              </div>

              {/* Action */}
              <div className="shrink-0">
                {isActive && (
                  <PauseCircleOutlined
                    title={t('tasks.pause')}
                    onClick={() => PauseTask(task.id)}
                    className="flex items-center justify-center rounded-full w-8 h-8 text-muted-foreground hover:text-primary hover:bg-blue-50 dark:hover:bg-blue-500/20"
                  />
                )}
                {task.status === TaskStatus.Paused && (
                  <PlayCircleOutlined
                    title={t('tasks.resume')}
                    onClick={() => ResumeTask(task.id)}
                    className="flex items-center justify-center rounded-full w-8 h-8 text-muted-foreground hover:text-primary hover:bg-blue-50 dark:hover:bg-blue-500/20"
                  />
                )}
                {((task.status === TaskStatus.Completed && task.file_exists === false) ||
                  task.status === TaskStatus.Pending) && (
                  <DeleteOutlined
                    title={t('tasks.contextMenu.delete')}
                    onClick={() => confirmDelete()}
                    className="flex items-center justify-center rounded-full w-8 h-8 text-muted-foreground hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-500/20"
                  />
                )}
                {((task.status === TaskStatus.Completed && task.file_exists === true) ||
                  task.status === TaskStatus.Error ||
                  task.status === TaskStatus.Merging) && (
                  <FileTextOutlined
                    title={t('tasks.viewLogs')}
                    onClick={onViewLog}
                    className="flex items-center justify-center rounded-full w-8 h-8 text-muted-foreground hover:text-primary hover:bg-blue-50 dark:hover:bg-blue-500/20"
                  />
                )}
              </div>
            </div>
          </Card>
        )}
      </div>
    </Dropdown>
  );
}
