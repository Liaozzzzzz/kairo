import React, { useMemo } from 'react';
import { Tooltip, Empty, Spin, Modal, notification, Divider } from 'antd';
import {
  PlayCircleOutlined,
  DeleteOutlined,
  RobotOutlined,
  FileTextOutlined,
  ScissorOutlined,
} from '@ant-design/icons';
import { Video } from '@/types';
import { formatDuration } from '@/lib/utils';
import { useTranslation } from 'react-i18next';
import { DeleteVideo, OpenFile } from '@root/wailsjs/go/main/App';
import { Grid, GridProps } from 'react-window';
import { AutoSizer } from 'react-virtualized-auto-sizer';

interface VideoListProps {
  videos: Video[];
  loading: boolean;
  onSelect: (video: Video) => void;
  onRefresh: () => void;
  onHighlights: (video: Video) => void;
}

const ResizableGrid = Grid as unknown as React.ComponentType<
  GridProps<object> & { height: number; width: number }
>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const Cell = (props: any) => {
  const {
    columnIndex,
    rowIndex,
    style,
    videos,
    columnCount,
    onSelect,
    onRefresh,
    onHighlights,
    t,
  } = props;
  const index = rowIndex * columnCount + columnIndex;

  if (index >= videos.length) {
    return null;
  }

  const video = videos[index];

  const handleDelete = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation();
    Modal.confirm({
      centered: true,
      title: t('videos.confirmDelete.title'),
      content: t('videos.confirmDelete.content'),
      okText: t('videos.confirmDelete.ok'),
      cancelText: t('videos.confirmDelete.cancel'),
      okButtonProps: { danger: true },
      onOk: async () => {
        try {
          await DeleteVideo(id);
          onRefresh();
        } catch (error) {
          console.error('Failed to delete video:', error);
          notification.error({
            title: t('videos.confirmDelete.delete_failed'),
            description: (error as Error).message || (error as string),
          });
        }
      },
    });
  };

  const handlePlay = async () => {
    try {
      await OpenFile(video.file_path);
    } catch (error) {
      console.error('Failed to open file:', error);
      notification.error({
        title: t('videos.open_failed'),
        description: (error as Error).message || (error as string),
      });
    }
  };

  return (
    <div style={style} className="p-2">
      <div className="group relative flex flex-col bg-white dark:bg-slate-800 rounded-xl overflow-hidden border border-slate-200 dark:border-slate-700 shadow-sm hover:shadow-xl transition-all duration-300 hover:-translate-y-1 cursor-pointer h-full">
        {/* Thumbnail Section */}
        <div className="relative aspect-video bg-slate-100 dark:bg-slate-900 overflow-hidden shrink-0">
          <img
            alt={video.title}
            src={video.thumbnail || 'src/assets/images/icon.png'}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
            onError={(e) => {
              (e.target as HTMLImageElement).src = '';
              (e.target as HTMLImageElement).classList.add('hidden');
            }}
          />

          {/* Overlay Gradient */}
          <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300" />

          {/* Center Play Button */}
          <div
            onClick={handlePlay}
            className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity duration-300 z-10"
          >
            <PlayCircleOutlined className="text-5xl text-white drop-shadow-lg transform scale-90 group-hover:scale-100 transition-transform" />
          </div>

          {/* Status Badges (Top Left) */}
          <div className="absolute top-2 left-2 flex flex-col gap-1 z-20">
            {video.status === 'completed' && (
              <Tooltip title={t('videos.ai_completed')}>
                <div className="bg-emerald-500/90 text-white text-[10px] font-bold px-2 py-0.5 rounded-full backdrop-blur-sm shadow-sm flex items-center gap-1">
                  <RobotOutlined />
                  <span>AI</span>
                </div>
              </Tooltip>
            )}
            {video.status === 'processing' && (
              <Tooltip title={t('videos.analyzing')}>
                <div className="bg-blue-500/90 text-white text-[10px] font-bold px-2 py-0.5 rounded-full backdrop-blur-sm shadow-sm flex items-center gap-1 animate-pulse">
                  <Spin size="small" />
                  <span>AI</span>
                </div>
              </Tooltip>
            )}
          </div>

          {/* Duration (Bottom Right) */}
          {video.duration > 0 && (
            <div className="absolute bottom-2 right-2 bg-black/60 text-white text-[10px] px-1.5 py-0.5 rounded-md backdrop-blur-md font-medium tabular-nums z-10">
              {formatDuration(video.duration)}
            </div>
          )}
        </div>

        {/* Content Section */}
        <div className="p-3 flex flex-col flex-1">
          <Tooltip title={video.title} mouseEnterDelay={0.5}>
            <h3 className="font-medium text-slate-700 dark:text-slate-200 text-sm leading-tight line-clamp-2 min-h-[2.5em]">
              {video.title}
            </h3>
          </Tooltip>
          <Divider size="small" />
          <div className="flex items-center justify-around text-xs text-slate-500 dark:text-slate-400">
            <Tooltip title={t('common.view_details')}>
              <div
                className="w-7 h-7 rounded-full hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400 hover:text-blue-500 dark:hover:text-blue-400 flex items-center justify-center transition-colors"
                onClick={(e) => {
                  e.stopPropagation();
                  onSelect(video);
                }}
              >
                <FileTextOutlined />
              </div>
            </Tooltip>
            <Divider orientation="vertical" />
            <Tooltip title={t('videos.highlights')}>
              <div
                className="w-7 h-7 rounded-full hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400 hover:text-purple-500 dark:hover:text-purple-400 flex items-center justify-center transition-colors"
                onClick={(e) => {
                  e.stopPropagation();
                  onHighlights(video);
                }}
              >
                <ScissorOutlined />
              </div>
            </Tooltip>
            <Divider orientation="vertical" />
            <Tooltip title={t('common.delete')}>
              <div
                className="w-7 h-7 rounded-full hover:bg-red-50 dark:hover:bg-red-900/20 text-slate-500 dark:text-slate-400 hover:text-red-500 dark:hover:text-red-400 flex items-center justify-center transition-colors"
                onClick={(e) => handleDelete(e, video.id)}
              >
                <DeleteOutlined />
              </div>
            </Tooltip>
          </div>
        </div>
      </div>
    </div>
  );
};

export default function VideoList({
  videos,
  loading,
  onSelect,
  onRefresh,
  onHighlights,
}: VideoListProps) {
  const { t } = useTranslation();

  const itemData = useMemo(
    () => ({
      videos,
      onSelect,
      onRefresh,
      onHighlights,
      t,
    }),
    [videos, onSelect, onRefresh, onHighlights, t]
  );

  if (!loading && videos.length === 0) {
    return <Empty description={t('videos.empty')} />;
  }

  return (
    <div className="h-full w-full relative">
      {loading && (
        <div className="absolute inset-0 bg-white/50 dark:bg-black/50 z-50 flex items-center justify-center backdrop-blur-sm">
          <Spin size="large" />
        </div>
      )}
      <AutoSizer
        renderProp={({ height, width }) => {
          if (height === undefined || width === undefined) return null;
          const columnCount = width < 500 ? 1 : width < 680 ? 2 : 3;
          const columnWidth = width / columnCount;
          const rowHeight = (columnWidth - 12) * (9 / 16) + 110;
          const rowCount = Math.ceil(videos.length / columnCount);

          return (
            <ResizableGrid
              columnCount={columnCount}
              columnWidth={columnWidth}
              height={height}
              rowCount={rowCount}
              rowHeight={rowHeight}
              width={width}
              cellProps={{ ...itemData, columnCount }}
              cellComponent={Cell}
            />
          );
        }}
      />
    </div>
  );
}
