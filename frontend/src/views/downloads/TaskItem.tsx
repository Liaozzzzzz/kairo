import { Card, Progress, Dropdown, MenuProps, Image } from 'antd';
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
import { useTranslation } from 'react-i18next';
import { Task } from '@/types';
import {
  PauseTask,
  ResumeTask,
  OpenTaskDir,
  RetryTask,
  DeleteTask as DeleteTaskWails,
} from '@root/wailsjs/go/main/App';
import { useAppStore } from '@/store/useAppStore';

interface TaskItemProps {
  task: Task;
  onViewLog: () => void;
}

export function TaskItem({ task, onViewLog }: TaskItemProps) {
  const { t } = useTranslation();
  const deleteTask = useAppStore((state) => state.deleteTask);

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircleOutlined className="w-4 h-4 text-green-500" />;
      case 'error':
        return <CloseCircleOutlined className="w-4 h-4 text-red-500" />;
      case 'downloading':
        return (
          <div className="animate-spin h-4 w-4 border-2 border-primary border-t-transparent rounded-full" />
        );
      case 'paused':
        return <PauseCircleOutlined className="w-4 h-4 text-yellow-500" />;
      default:
        return <PlayCircleOutlined className="w-4 h-4" />;
    }
  };

  const getStatusText = (status: string) => {
    switch (status) {
      case 'completed':
        return t('downloads.status.completed');
      case 'error':
        return t('downloads.status.error');
      case 'downloading':
        return t('downloads.status.downloading');
      case 'pending':
        return t('downloads.status.pending');
      case 'paused':
        return t('downloads.status.paused');
      default:
        return status;
    }
  };

  const getActiveStage = (task: Task) => {
    if ((task.status !== 'downloading' && task.status !== 'paused') || !task.stages) return null;
    const stage = task.stages.find((s) => s.status === 'downloading');
    if (!stage) return null;
    return stage;
  };

  const activeStage = getActiveStage(task);
  const showStage = (task.status === 'downloading' || task.status === 'paused') && activeStage;

  const displayProgress = showStage ? activeStage!.progress : task.progress;
  const displaySize =
    showStage && activeStage!.total_size ? activeStage!.total_size : task.total_size;
  const displayTag = showStage ? t(`downloads.stages.${activeStage!.name}`) : null;

  const menuItems: MenuProps['items'] = [
    {
      key: 'details',
      label: t('downloads.contextMenu.details'),
      icon: <FileTextOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: onViewLog,
    },
    {
      key: 'open',
      label: t('downloads.contextMenu.openLocation'),
      icon: <FolderOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: () => OpenTaskDir(task.id),
    },
    {
      key: 'copy',
      label: t('downloads.contextMenu.copyLink'),
      icon: <LinkOutlined className="w-4 h-4 mt-[-2px]" />,
      onClick: () => navigator.clipboard.writeText(task.url),
    },
    ...(task.status === 'error'
      ? [
          {
            key: 'retry',
            label: t('downloads.contextMenu.retry'),
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
      label: t('downloads.contextMenu.delete'),
      icon: <DeleteOutlined className="w-4 h-4 mt-[-2px]" />,
      danger: true,
      onClick: () => {
        DeleteTaskWails(task.id);
        deleteTask(task.id);
      },
    },
  ];

  const getStatusTagClass = (status: string) => {
    switch (status) {
      case 'completed':
        return 'bg-green-50 text-green-600 border border-green-200';
      case 'error':
        return 'bg-red-50 text-red-600 border border-red-200';
      case 'downloading':
        return 'bg-blue-50 text-blue-600 border border-blue-200';
      case 'paused':
        return 'bg-yellow-50 text-yellow-600 border border-yellow-200';
      default:
        return 'bg-gray-50 text-gray-500 border border-gray-200';
    }
  };

  const getProgressColor = () => {
    if (task.status === 'completed') return '#22c55e'; // green-500
    if (task.status === 'error') return '#ef4444'; // red-500
    if (task.status === 'paused') return '#eab308'; // yellow-500
    return undefined; // default primary
  };

  return (
    <Dropdown menu={{ items: menuItems }} trigger={['contextMenu']}>
      <Card
        hoverable
        variant="borderless"
        className="relative overflow-hidden"
        styles={{ body: { padding: '16px' } }}
      >
        <div className="flex items-center gap-4">
          {/* Thumbnail Column */}
          <div className="flex-shrink-0 w-24 h-16 bg-gray-100 rounded-md overflow-hidden relative border border-gray-100">
            {task.thumbnail ? (
              <Image
                src={task.thumbnail}
                className="w-full h-full object-cover"
                alt=""
                width="100%"
                height="100%"
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
                    <span>{displaySize}</span>
                  </>
                )}
                {task.status === 'downloading' && (
                  <>
                    {task.speed && task.speed !== '~' && (
                      <>
                        <span className="text-gray-300">•</span>
                        <span>{task.speed}</span>
                      </>
                    )}
                    {task.eta && task.eta !== '~' && (
                      <>
                        <span className="text-gray-300">•</span>
                        <span>ETA {task.eta}</span>
                      </>
                    )}
                  </>
                )}
                {displayTag && (
                  <span
                    className={`bg-blue-50 text-blue-600 px-1.5 py-0.5 rounded text-[10px] border border-blue-100 ml-1 ${task.status === 'downloading' ? 'animate-pulse' : ''}`}
                  >
                    {displayTag}
                  </span>
                )}
              </div>
              <span className="uppercase">{task.quality}</span>
            </div>
          </div>

          {/* Action */}
          <div className="shrink-0">
            {task.status === 'downloading' && (
              <PauseCircleOutlined
                title={t('downloads.pause')}
                onClick={() => PauseTask(task.id)}
                className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-primary hover:bg-blue-50"
              />
            )}
            {task.status === 'paused' && (
              <PlayCircleOutlined
                title={t('downloads.resume')}
                onClick={() => ResumeTask(task.id)}
                className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-primary hover:bg-blue-50"
              />
            )}
            {(task.status === 'completed' || task.status === 'error') && (
              <FileTextOutlined
                title={t('downloads.viewLogs')}
                onClick={onViewLog}
                className="flex items-center justify-center rounded-full w-8 h-8 text-gray-400 hover:text-primary hover:bg-blue-50"
              />
            )}
          </div>
        </div>
      </Card>
    </Dropdown>
  );
}
