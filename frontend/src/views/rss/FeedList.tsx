import React from 'react';
import { useTranslation } from 'react-i18next';
import { useRSSStore } from '@/store/useRSSStore';
import { Dropdown, MenuProps } from 'antd';
import {
  DeleteOutlined,
  ReloadOutlined,
  StopOutlined,
  CheckCircleOutlined,
  EditOutlined,
} from '@ant-design/icons';
import { cn } from '@/lib/utils';
import { RSSFeed } from '@/types';

interface FeedListProps {
  onEdit: (feed: RSSFeed) => void;
}

const FeedList: React.FC<FeedListProps> = ({ onEdit }) => {
  const { t } = useTranslation();
  const {
    feeds,
    selectedFeedId,
    selectFeed,
    deleteFeed,
    refreshFeed,
    toggleFeedEnabled,
    isItemsLoading,
  } = useRSSStore();

  const getMenuItems = (feed: RSSFeed): MenuProps['items'] => [
    {
      key: 'edit',
      label: t('rss.edit'),
      icon: <EditOutlined />,
      onClick: ({ domEvent }) => {
        domEvent.stopPropagation();
        onEdit(feed);
      },
    },
    {
      key: 'toggleEnabled',
      label: feed.enabled ? t('rss.disable') : t('rss.enable'),
      icon: feed.enabled ? <StopOutlined /> : <CheckCircleOutlined />,
      onClick: ({ domEvent }) => {
        domEvent.stopPropagation();
        toggleFeedEnabled(feed.id, !feed.enabled);
      },
    },
    {
      key: 'refresh',
      label: t('rss.refresh'),
      icon: <ReloadOutlined spin={selectedFeedId === feed.id && isItemsLoading} />,
      disabled: selectedFeedId === feed.id && isItemsLoading,
      onClick: ({ domEvent }) => {
        domEvent.stopPropagation();
        refreshFeed(feed.id);
      },
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: t('rss.delete'),
      icon: <DeleteOutlined />,
      danger: true,
      onClick: ({ domEvent }) => {
        domEvent.stopPropagation();
        deleteFeed(feed.id);
      },
    },
  ];

  return (
    <div className="flex items-start py-1 gap-3 overflow-x-auto px-1 scrollbar-none">
      {feeds.map((feed) => (
        <Dropdown key={feed.id} menu={{ items: getMenuItems(feed) }} trigger={['contextMenu']}>
          <div
            className="flex flex-col items-center gap-1.5 cursor-pointer group"
            onClick={() => {
              if (selectedFeedId !== feed.id) {
                selectFeed(feed.id);
              }
            }}
          >
            <div
              className={cn(
                'relative shrink-0 w-14 h-14 rounded-full transition-all',
                selectedFeedId === feed.id
                  ? 'ring-2 ring-primary ring-offset-2 ring-offset-white dark:ring-offset-neutral-950'
                  : 'ring-1 ring-slate-200 dark:ring-neutral-800 group-hover:ring-slate-300 dark:group-hover:ring-neutral-700'
              )}
            >
              <div
                className={cn(
                  'w-full h-full rounded-full overflow-hidden',
                  !feed.enabled && 'opacity-50 grayscale'
                )}
              >
                {feed.thumbnail ? (
                  <img
                    src={feed.thumbnail}
                    alt={feed.title}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="w-full h-full bg-slate-100 dark:bg-neutral-800 flex items-center justify-center text-lg font-bold text-slate-500 dark:text-slate-400">
                    {feed.title.charAt(0).toUpperCase()}
                  </div>
                )}
              </div>

              {!feed.enabled && (
                <div className="absolute inset-0 z-10 flex items-center justify-center bg-black/10 rounded-full">
                  <StopOutlined className="text-white text-xl drop-shadow-md" />
                </div>
              )}

              {feed.unread_count > 0 && (
                <div
                  className={cn(
                    'absolute -top-0.5 -right-0.5 z-20 min-w-[18px] h-[18px] px-1 rounded-full border-[1.5px] border-white dark:border-neutral-900 flex items-center justify-center shadow-sm',
                    !feed.enabled ? 'bg-slate-400' : 'bg-red-500'
                  )}
                >
                  <span className="text-[10px] font-bold text-white leading-none">
                    {feed.unread_count > 99 ? '99+' : feed.unread_count}
                  </span>
                </div>
              )}
            </div>
            <span
              className={cn(
                'text-[11px] w-16 truncate text-center transition-colors',
                selectedFeedId === feed.id
                  ? 'font-medium text-primary'
                  : 'text-slate-500 dark:text-slate-400 group-hover:text-slate-700 dark:group-hover:text-slate-300'
              )}
            >
              {feed.title}
            </span>
          </div>
        </Dropdown>
      ))}
    </div>
  );
};

export default FeedList;
