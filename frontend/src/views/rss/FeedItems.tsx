import React, { useMemo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { useRSSStore } from '@/store/useRSSStore';
import { Button, Spin, message, Card, Empty } from 'antd';
import { ExportOutlined } from '@ant-design/icons';
import { cn } from '@/lib/utils';
import { BrowserOpenURL } from '@root/wailsjs/runtime/runtime';
import { AddRSSTask } from '@root/wailsjs/go/main/App';
import { models } from '@root/wailsjs/go/models';
import dayjs from 'dayjs';
import { RSSItem, RSSFeed, RSSItemStatus } from '@/types';
import { Grid, CellComponentProps, GridProps } from 'react-window';
import { AutoSizer } from 'react-virtualized-auto-sizer';

interface CellData {
  feedItems: RSSItem[];
  columnCount: number;
  handleDownload: (e: React.MouseEvent, item: RSSItem) => void;
  markItemRead: (id: string) => void;
  t: (key: string) => string;
  currentFeed: RSSFeed | undefined;
}

const ResizableGrid = Grid as unknown as React.ComponentType<
  GridProps<CellData> & { height: number; width: number }
>;

const Cell = (props: CellComponentProps<CellData>) => {
  const {
    columnIndex,
    rowIndex,
    style,
    feedItems,
    columnCount,
    handleDownload,
    markItemRead,
    t,
    currentFeed,
  } = props;
  const index = rowIndex * columnCount + columnIndex;

  if (index >= feedItems.length) {
    return null;
  }

  const item = feedItems[index];

  const getStatusInfo = (status: RSSItemStatus) => {
    switch (status) {
      case RSSItemStatus.New:
        return {
          text: t('rss.status.new'),
          className: 'bg-blue-500/90 text-white border-blue-400/50',
        };
      case RSSItemStatus.Queued:
        return {
          text: t('rss.status.queued'),
          className: 'bg-emerald-500/90 text-white border-emerald-400/50',
        };
      case RSSItemStatus.Failed:
        return {
          text: t('rss.status.failed'),
          className: 'bg-red-500/90 text-white border-red-400/50',
        };
      case RSSItemStatus.Downloaded:
        return {
          text: t('rss.status.downloaded'),
          className: 'bg-purple-500/90 text-white border-purple-400/50',
        };
      default:
        return null;
    }
  };

  const statusInfo = getStatusInfo(item.status);

  return (
    <div style={style} className="p-1.5">
      <div className="group flex flex-col h-full rounded-2xl bg-white dark:bg-neutral-900 border border-slate-200 dark:border-neutral-800 hover:shadow-xl hover:border-primary/30 transition-all duration-300 overflow-hidden">
        <div
          className="relative w-full aspect-video cursor-pointer bg-slate-100 dark:bg-neutral-800 shrink-0"
          onClick={(e) => handleDownload(e, item)}
        >
          {item.thumbnail ? (
            <img
              src={item.thumbnail}
              alt={item.title}
              className="w-full h-full object-cover transition-transform duration-700 group-hover:scale-105"
            />
          ) : (
            <div className="w-full h-full flex items-center justify-center text-slate-400 font-medium text-sm">
              {t('rss.noThumbnail')}
            </div>
          )}

          <div className="absolute top-2 left-2 flex items-center gap-1.5 bg-black/60 backdrop-blur-md rounded-full pl-1 pr-2.5 py-0.5 max-w-[70%] border border-white/10 opacity-0 group-hover:opacity-100 transition-opacity duration-300 pointer-events-none">
            {currentFeed?.thumbnail ? (
              <img
                src={currentFeed.thumbnail}
                alt={currentFeed.title}
                className="w-4 h-4 rounded-full object-cover shrink-0"
              />
            ) : (
              <div className="w-4 h-4 rounded-full bg-white/20 flex items-center justify-center text-[8px] font-bold shrink-0 text-white">
                {currentFeed?.title?.charAt(0).toUpperCase()}
              </div>
            )}
            <span className="text-[10px] text-white truncate font-medium drop-shadow-sm">
              {currentFeed?.title}
            </span>
          </div>

          <div className="absolute bottom-2 left-2 px-1.5 py-0.5 rounded bg-black/60 backdrop-blur-sm text-[10px] text-white font-medium border border-white/10">
            {dayjs(item.pub_date * 1000).format('MM-DD')}
          </div>

          {statusInfo && (
            <div
              className={cn(
                'absolute bottom-2 right-2 px-2 py-1 rounded-full text-[10px] leading-none font-bold backdrop-blur-md border shadow-sm tracking-wide uppercase',
                statusInfo.className
              )}
            >
              {statusInfo.text}
            </div>
          )}
        </div>

        <div className="flex items-start justify-between gap-1 p-2 flex-1">
          <h3
            className={cn(
              'font-bold text-sm leading-snug  line-clamp-2 cursor-pointer hover:text-primary transition-colors text-slate-900 dark:text-slate-100'
            )}
            onClick={() => {
              markItemRead(item.id);
              BrowserOpenURL(item.link);
            }}
            title={item.title}
          >
            {item.title}
          </h3>

          <Button
            type="text"
            size="small"
            className="shrink-0 text-slate-400 hover:text-primary hover:bg-primary/10 -mr-1"
            icon={<ExportOutlined />}
            onClick={(e) => {
              e.stopPropagation();
              markItemRead(item.id);
              BrowserOpenURL(item.link);
            }}
          />
        </div>
      </div>
    </div>
  );
};

const FeedItems: React.FC = () => {
  const { t } = useTranslation();
  const { feedItems, isItemsLoading, selectedFeedId, markItemRead, feeds, setRSSItemQueued } =
    useRSSStore();

  const currentFeed = useMemo(
    () => feeds.find((f) => f.id === selectedFeedId),
    [feeds, selectedFeedId]
  );

  const handleDownload = useCallback(
    async (e: React.MouseEvent, item: RSSItem) => {
      e.stopPropagation();
      try {
        if (item.status === RSSItemStatus.Queued || item.status === RSSItemStatus.Downloaded) {
          return;
        }

        await AddRSSTask(
          new models.AddRSSTaskInput({
            feed_url: currentFeed?.url || '',
            feed_title: currentFeed?.title || '',
            feed_thumbnail: currentFeed?.thumbnail || '',
            item_url: item.link,
            item_title: item.title,
            item_thumbnail: item.thumbnail,
            dir: currentFeed?.custom_dir || '',
          })
        );
        await markItemRead(item.id);
        await setRSSItemQueued(item.id, true);
        message.success(t('rss.taskAdded'));
      } catch (err) {
        console.error(err);
        message.error(t('rss.taskAddFailed'));
      }
    },
    [t, markItemRead, setRSSItemQueued, currentFeed]
  );

  const itemData = useMemo(
    () => ({
      feedItems,
      handleDownload,
      markItemRead,
      t,
      currentFeed,
    }),
    [feedItems, handleDownload, markItemRead, t, currentFeed]
  );

  if (isItemsLoading) {
    return (
      <div className="py-6 flex flex-col items-center justify-center gap-4">
        <Spin size="large" description="Loading"></Spin>
      </div>
    );
  }

  if (feedItems.length === 0) {
    return <Empty description={t('rss.noItems')} />;
  }

  return (
    <Card
      variant="borderless"
      className="h-full flex flex-col"
      size="small"
      styles={{
        body: { flex: 1, overflow: 'hidden', padding: '6px' },
      }}
    >
      <AutoSizer
        renderProp={({ height, width }) => {
          if (height === undefined || width === undefined) return null;
          const columnCount = width < 500 ? 1 : width < 680 ? 2 : 3;
          const columnWidth = width / columnCount;
          const rowHeight = (columnWidth - 12) * (9 / 16) + 70;
          const rowCount = Math.ceil(feedItems.length / columnCount);

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
    </Card>
  );
};

export default FeedItems;
