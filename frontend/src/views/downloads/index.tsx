import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { Input, Card, notification, message, Empty, Spin } from 'antd';
import { GetVideoInfo, AddTask as AddTaskGo, AddPlaylistTask } from '@root/wailsjs/go/main/App';
import { models } from '@root/wailsjs/go/models';
import { useAppStore } from '@/store/useAppStore';
import { useTaskStore } from '@/store/useTaskStore';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import bilibiliIcon from '@/assets/images/bilibili.png';
import youtubeIcon from '@/assets/images/Youtube.png';
import { MenuItemKey, TrimMode, SourceType } from '@/data/variables';
import SingleVideoResult from './SingleVideoResult';
import PlaylistResult from './PlaylistResult';

export default function Downloads() {
  const { t } = useTranslation();
  const tasks = useTaskStore(useShallow((state) => state.tasks));
  const [api, contextHolder] = notification.useNotification();
  const [messageApi, messageContextHolder] = message.useMessage();

  const [newUrl, setNewUrl] = useState('');

  const setActiveTab = useAppStore(useShallow((state) => state.setActiveTab));

  const [videoInfo, setVideoInfo] = useState<models.VideoInfo | null>(null);
  const [isFetchingInfo, setIsFetchingInfo] = useState(false);

  const playlistItems = videoInfo?.playlist_items || [];
  const isPlaylist = Boolean(
    videoInfo?.source_type === SourceType.Playlist && playlistItems.length
  );

  const fetchVideoInfo = async () => {
    if (!newUrl) return;

    // Check duplicate
    const isDuplicate = Object.values(tasks).some((t) => t.url === newUrl);
    if (isDuplicate) {
      messageApi.warning(t('downloads.duplicateTaskContent'));
      return;
    }

    setIsFetchingInfo(true);
    setVideoInfo(null);
    try {
      const info = await GetVideoInfo(newUrl);
      setVideoInfo(info);
    } catch (e) {
      console.error(e);
      api.error({
        title: t('downloads.parseError'),
        description: (e as Error).message || (e as string),
      });
    } finally {
      setIsFetchingInfo(false);
    }
  };

  const handleStartDownload = async ({
    newDir,
    newQuality,
    newFormat,
    trimStart,
    trimEnd,
    trimMode,
  }: {
    newDir: string;
    newQuality: string;
    newFormat: string;
    trimStart: string;
    trimEnd: string;
    trimMode: TrimMode;
  }) => {
    if (!newUrl || !newDir) return;
    try {
      const selectedQuality = videoInfo?.qualities?.find((q) => q.value === newQuality);
      const totalBytes = isPlaylist ? 0 : selectedQuality?.total_bytes || 0;
      const formatId = selectedQuality?.format_id || '';
      await AddTaskGo(
        new models.AddTaskInput({
          url: newUrl,
          quality: newQuality,
          format: newFormat,
          format_id: formatId,
          dir: newDir,
          title: videoInfo?.title || '',
          thumbnail: videoInfo?.thumbnail || '',
          total_bytes: totalBytes,
          trim_start: trimStart,
          trim_end: trimEnd,
          trim_mode: trimMode,
          source_type: SourceType.Single,
        })
      );
      setNewUrl('');
      setActiveTab(MenuItemKey.Tasks);
    } catch (e) {
      console.error(e);
    }
  };

  const handleStartPlaylistDownload = async ({
    newDir,
    playList = [],
  }: {
    newDir: string;
    playList?: number[];
  }) => {
    if (!newUrl || !newDir) return;
    try {
      const sortedItems = [...playList].sort((a, b) => a - b);
      const selectedItems: models.PlaylistItem[] = [];

      for (const index of sortedItems) {
        const item = playlistItems.find((playlistItem) => playlistItem.index === index);
        if (item) {
          selectedItems.push(item);
        }
      }

      if (selectedItems.length === 0) {
        return;
      }

      await AddPlaylistTask(
        new models.AddPlaylistTaskInput({
          url: newUrl,
          dir: newDir,
          title: videoInfo?.title || newUrl,
          thumbnail: videoInfo?.thumbnail || '',
          playlist_items: selectedItems,
        })
      );

      setNewUrl('');
      setActiveTab(MenuItemKey.Tasks);
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <PageContainer
      viewClass="px-10"
      header={
        <div className="flex items-end justify-between">
          <PageHeader title={t('downloads.title')} subtitle={t('downloads.subtitle')} />
          <div className="flex items-center gap-4">
            <span className="text-sm font-medium mt-1 text-muted-foreground">
              {t('downloads.supportedSites')}
            </span>
            <div className="flex items-center gap-3">
              <img src={bilibiliIcon} alt="Bilibili" title="Bilibili" className="w-10 h-4.5" />
              <img src={youtubeIcon} alt="YouTube" title="YouTube" className="w-14 h-6" />
            </div>
          </div>
        </div>
      }
    >
      {contextHolder}
      {messageContextHolder}
      <Card variant="borderless" className="shadow-sm">
        <div className="space-y-6">
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">{t('downloads.videoUrl')}</label>
            <Input.Search
              value={newUrl}
              onChange={(e) => {
                setNewUrl(e.target.value);
                setVideoInfo(null);
              }}
              loading={isFetchingInfo}
              placeholder="https://www.youtube.com/watch?v=..."
              enterButton={t('downloads.analyze')}
              onSearch={fetchVideoInfo}
            />
          </div>

          {isFetchingInfo && (
            <div className="flex flex-col items-center justify-center py-12 gap-3">
              <Spin size="large" />
              <span className="text-muted-foreground">{t('downloads.analyzing')}</span>
            </div>
          )}

          {!videoInfo && !isFetchingInfo && (
            <div className="flex flex-col items-center justify-center py-12">
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('downloads.emptyState')} />
            </div>
          )}

          {videoInfo && !isPlaylist && (
            <SingleVideoResult videoInfo={videoInfo} onStartDownload={handleStartDownload} />
          )}

          {videoInfo && isPlaylist && (
            <PlaylistResult videoInfo={videoInfo} onStartDownload={handleStartPlaylistDownload} />
          )}
        </div>
      </Card>
    </PageContainer>
  );
}
