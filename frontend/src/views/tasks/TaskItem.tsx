import { useMemo } from 'react';
import { Card, Progress, Dropdown, MenuProps, Image, Badge, Modal } from 'antd';
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
  CloseOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Task } from '@/types';
import { TaskStatus } from '@/data/variables';
import { PauseTask, ResumeTask, OpenTaskDir, RetryTask } from '@root/wailsjs/go/main/App';
import { useTaskStore } from '@/store/useTaskStore';
import { ImageFallback } from '@/data/variables';

interface TaskItemProps {
  task: Task;
  onViewLog: () => void;
}

export function TaskItem({ task, onViewLog }: TaskItemProps) {
  const { t } = useTranslation();
  const deleteTask = useTaskStore((state) => state.deleteTask);

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
        return <CloseCircleOutlined className="w-4 h-4 text-red-500" />;
      case TaskStatus.Starting:
      case TaskStatus.Merging:
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
      case TaskStatus.Pending:
        return t('tasks.status.pending');
      case TaskStatus.Paused:
        return t('tasks.status.paused');
      default:
        return status;
    }
  };

  const displayProgress = task.progress;
  const displaySize = task.total_size;

  const confirmDelete = (purge: boolean) => {
    if (task.status === TaskStatus.Merging) {
      return;
    }
    const modalKey = purge ? 'tasks.confirmPurge' : 'tasks.confirmDelete';
    Modal.confirm({
      centered: true,
      title: t(`${modalKey}.title`),
      content: t(`${modalKey}.content`),
      okText: t(`${modalKey}.ok`),
      cancelText: t(`${modalKey}.cancel`),
      okButtonProps: { danger: true },
      onOk: () => deleteTask(task.id, purge),
    });
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
      onClick: () => confirmDelete(false),
    },
    ...(task.status !== TaskStatus.Merging
      ? [
        {
          key: 'purge',
          label: t('tasks.contextMenu.purge'),
          icon: <CloseOutlined className="w-4 h-4 mt-[-2px]" />,
          danger: true,
          onClick: () => confirmDelete(true),
        },
      ]
      : []),
  ];

  const getStatusTagClass = (status: string) => {
    switch (status) {
      case TaskStatus.Completed:
        return task.file_exists === false
          ? 'bg-gray-100 text-gray-400 border border-gray-200' // Disabled/Missing
          : 'bg-green-50 text-green-600 border border-green-200';
      case TaskStatus.Error:
        return 'bg-red-50 text-red-600 border border-red-200';
      case TaskStatus.Starting:
      case TaskStatus.Merging:
      case TaskStatus.Downloading:
        return 'bg-blue-50 text-blue-600 border border-blue-200';
      case TaskStatus.Paused:
        return 'bg-yellow-50 text-yellow-600 border border-yellow-200';
      default:
        return 'bg-gray-50 text-gray-500 border border-gray-200';
    }
  };

  const getProgressColor = () => {
    if (task.status === TaskStatus.Completed) return '#22c55e'; // green-500
    if (task.status === TaskStatus.Error) return '#ef4444'; // red-500
    if (task.status === TaskStatus.Paused) return '#eab308'; // yellow-500
    return undefined; // default primary
  };

  const isActive = task.status === TaskStatus.Starting || task.status === TaskStatus.Downloading;
  console.log(task)
  return (
    <Dropdown menu={{ items: menuItems }} trigger={['contextMenu']}>
      <div>
        <Badge.Ribbon text={siteLabel} color="cyan">
          <Card
            hoverable
            variant="borderless"
            className={`${task.status === TaskStatus.Completed && task.file_exists === false
              ? 'opacity-60 grayscale'
              : ''
              }`}
            styles={{ body: { padding: '16px' } }}
          >
            <div className="flex items-center gap-4">
              {/* Thumbnail Column */}
              <div className="flex-shrink-0 w-24 h-16 bg-gray-100 rounded-md overflow-hidden relative border border-gray-100">
                {task.thumbnail ? (
                  <Image
                    src={task.thumbnail}
                    referrerPolicy="no-referrer"
                    className="w-full h-full object-cover"
                    alt=""
                    width="100%"
                    height="100%"
                    fallback={ImageFallback}
                  />
                ) : (
                  <div className="flex items-center justify-center w-full h-full text-gray-300">
                    <PlayCircleOutlined className="w-4 h-4" />
                  </div>
                )}
                <div className="flex items-center justify-center absolute w-6 h-6 -bottom-1 -right-1 bg-white rounded-full p-0.5 shadow-sm border border-gray-100 scale-75 z-10">
                  {getStatusIcon(task.status)}
                </div>
              </div>

              {/* Main Info */}
              <div className="flex-1 min-w-0 flex flex-col gap-1.5">
                <div className="flex items-center justify-between gap-4 min-w-0">
                  <div
                    className="flex-1 min-w-0 font-semibold text-[15px] truncate text-gray-900"
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
                  railColor="#f3f4f6" // gray-100
                  className="m-0"
                />
                <div className="flex justify-between text-[11px] font-medium text-gray-400 items-center">
                  <div className="flex items-center gap-2">
                    <span>{displayProgress.toFixed(1)}%</span>
                    {displaySize && displaySize !== '~' && (
                      <>
                        <span className="text-gray-300">•</span>
                        <span>~{displaySize}</span>
                      </>
                    )}
                    {isActive && (
                      <>
                        {task.speed && task.speed !== '~' && (
                          <>
                            <span className="text-gray-300">•</span>
                            <span>{task.speed}</span>
                          </>
                        )}
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
                    className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-primary hover:bg-blue-50"
                  />
                )}
                {task.status === TaskStatus.Paused && (
                  <PlayCircleOutlined
                    title={t('tasks.resume')}
                    onClick={() => ResumeTask(task.id)}
                    className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-primary hover:bg-blue-50"
                  />
                )}
                {((task.status === TaskStatus.Completed && task.file_exists === false) ||
                  task.status === TaskStatus.Pending) && (
                    <DeleteOutlined
                      title={t('tasks.contextMenu.delete')}
                      onClick={() => confirmDelete(false)}
                      className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-red-500 hover:bg-red-50"
                    />
                  )}
                {((task.status === TaskStatus.Completed && task.file_exists === true) ||
                  task.status === TaskStatus.Error ||
                  task.status === TaskStatus.Merging) && (
                    <FileTextOutlined
                      title={t('tasks.viewLogs')}
                      onClick={onViewLog}
                      className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-primary hover:bg-blue-50"
                    />
                  )}
              </div>
            </div>
          </Card>
        </Badge.Ribbon>
      </div>
    </Dropdown>
  );
}
