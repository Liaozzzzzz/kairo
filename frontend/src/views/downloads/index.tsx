import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { Input, Button, Select, Space, Image, Card, notification } from 'antd';
import { FolderOutlined, DownloadOutlined } from '@ant-design/icons';
import { GetVideoInfo, AddTask as AddTaskGo, ChooseDirectory } from '@root/wailsjs/go/main/App';
import { models } from '@root/wailsjs/go/models';
import { useSettingStore } from '@/store/useSettingStore';
import { useAppStore } from '@/store/useAppStore';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import bilibiliIcon from '@/assets/images/bilibili.png';
import youtubeIcon from '@/assets/images/Youtube.png';
import { ImageFallback, MenuItemKey } from '@/data/variables';

export default function Downloads() {
  const { t } = useTranslation();
  const [api, contextHolder] = notification.useNotification();

  const defaultDir = useSettingStore(useShallow((state) => state.defaultDir));
  const setActiveTab = useAppStore(useShallow((state) => state.setActiveTab));

  const [newUrl, setNewUrl] = useState('');
  const [newDir, setNewDir] = useState('');
  const [newQuality, setNewQuality] = useState('best');
  const [newFormat, setNewFormat] = useState('original');
  const [videoInfo, setVideoInfo] = useState<models.VideoInfo | null>(null);
  const [isFetchingInfo, setIsFetchingInfo] = useState(false);

  // Initialize directory when defaultDir is loaded
  useEffect(() => {
    if (defaultDir && !newDir) {
      setNewDir(defaultDir);
    }
  }, [defaultDir, newDir]);

  const fetchVideoInfo = async () => {
    if (!newUrl) return;

    setIsFetchingInfo(true);
    setVideoInfo(null);
    try {
      const info = await GetVideoInfo(newUrl);
      setVideoInfo(info);
      if (info.qualities && info.qualities.length > 0) {
        setNewQuality(info.qualities[0].value);
      } else {
        setNewQuality('best');
      }
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

  const handleChooseDir = async () => {
    try {
      const d = await ChooseDirectory();
      if (d) setNewDir(d);
    } catch (e) {
      console.error(e);
    }
  };

  const handleStartDownload = async () => {
    if (!newUrl || !newDir) return;
    try {
      const selectedQuality = videoInfo?.qualities?.find((q) => q.value === newQuality);
      const totalBytes = selectedQuality?.total_bytes || 0;

      await AddTaskGo({
        url: newUrl,
        quality: newQuality,
        format: newFormat,
        dir: newDir,
        title: videoInfo?.title || '',
        thumbnail: videoInfo?.thumbnail || '',
        total_bytes: totalBytes,
      });
      // Reset form
      setNewUrl('');
      setNewQuality('best');
      setNewFormat('original');
      setVideoInfo(null);
      if (defaultDir) setNewDir(defaultDir);

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
            <span className="text-sm ont-medium mt-1">{t('downloads.supportedSites')}</span>
            <div className="flex items-center gap-3">
              <img src={bilibiliIcon} alt="Bilibili" title="Bilibili" className="w-10 h-4.5" />
              <img src={youtubeIcon} alt="YouTube" title="YouTube" className="w-14 h-6" />
            </div>
          </div>
        </div>
      }
    >
      {contextHolder}
      <Card variant="borderless" className="shadow-sm">
        <div className="space-y-6">
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('downloads.videoUrl')}</label>
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
              size="large"
            />
          </div>

          {videoInfo && (
            <div className="bg-slate-50 border border-slate-200 p-4 rounded-lg flex gap-4 shadow-sm animate-in fade-in slide-in-from-top-2 duration-300">
              <div className="flex-shrink-0 w-40 aspect-video rounded-md overflow-hidden border border-black/10">
                <Image
                  src={videoInfo.thumbnail}
                  className="w-full h-full object-cover"
                  alt=""
                  width="100%"
                  height="100%"
                  fallback={ImageFallback}
                />
              </div>
              <div className="flex-1 min-w-0 flex flex-col justify-center">
                <div
                  className="font-medium text-lg truncate text-foreground"
                  title={videoInfo.title}
                >
                  {videoInfo.title}
                </div>
                <div className="text-sm text-muted-foreground mt-2 flex items-center gap-2">
                  <span className="bg-white px-2 py-0.5 rounded border border-slate-200 text-xs">
                    {t('downloads.duration')}
                    {Math.floor(videoInfo.duration / 60)}:
                    {String(Math.floor(videoInfo.duration % 60)).padStart(2, '0')}
                  </span>
                </div>
              </div>
            </div>
          )}

          {videoInfo && (
            <div className="grid grid-cols-2 gap-6 animate-in fade-in slide-in-from-top-2 duration-300 delay-75">
              <div className="space-y-2">
                <label className="text-sm font-medium">{t('downloads.quality')}</label>
                <Select
                  value={newQuality}
                  onChange={setNewQuality}
                  style={{ width: '100%' }}
                  size="large"
                  options={(videoInfo?.qualities || []).map((q) => ({
                    label: (
                      <div className="flex justify-between items-center w-full gap-4">
                        <span>{q.label}</span>
                        <span className="text-gray-400 text-xs font-normal">{q.total_size}</span>
                      </div>
                    ),
                    value: q.value,
                  }))}
                  placeholder={t('downloads.bestQuality')}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t('downloads.format')}</label>
                <Select
                  value={newFormat}
                  onChange={setNewFormat}
                  style={{ width: '100%' }}
                  size="large"
                  options={[
                    // 不转码
                    { label: 'ORIGINAL', value: 'original' },
                    { label: 'WEBM', value: 'webm' },
                    { label: 'MP4', value: 'mp4' },
                    { label: 'MKV', value: 'mkv' },
                    { label: 'AVI', value: 'avi' },
                    { label: 'FLV', value: 'flv' },
                    { label: 'MOV', value: 'mov' },
                  ]}
                />
              </div>
              <div className="space-y-2 col-span-2">
                <label className="text-sm font-medium">{t('downloads.saveTo')}</label>
                <Space.Compact style={{ width: '100%' }} size="large">
                  <Input value={newDir} readOnly />
                  <Button icon={<FolderOutlined />} onClick={handleChooseDir}>
                    {t('downloads.chooseDir')}
                  </Button>
                </Space.Compact>
              </div>

              <div className="col-span-2 pt-4 flex justify-end">
                <Button
                  type="primary"
                  size="large"
                  onClick={handleStartDownload}
                  disabled={!newUrl || !newDir || !videoInfo}
                  icon={<DownloadOutlined />}
                  className="px-8"
                >
                  {t('downloads.start')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </Card>
    </PageContainer>
  );
}
