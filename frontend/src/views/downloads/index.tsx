import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { Input, Button, Select, Space, Image, Card } from 'antd';
import { FolderOutlined, DownloadOutlined } from '@ant-design/icons';
import { GetVideoInfo, AddTask as AddTaskGo, ChooseDirectory } from '@root/wailsjs/go/main/App';
import { main } from '@root/wailsjs/go/models';
import { useAppStore } from '@/store/useAppStore';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';

interface DownloadsProps {
  onAdded?: () => void;
}

export default function Downloads({ onAdded }: DownloadsProps) {
  const { t } = useTranslation();
  const defaultDir = useAppStore(useShallow((state) => state.defaultDir));

  const [newUrl, setNewUrl] = useState('');
  const [newDir, setNewDir] = useState('');
  const [newQuality, setNewQuality] = useState('best');
  const [newFormat, setNewFormat] = useState('webm');
  const [videoInfo, setVideoInfo] = useState<main.VideoInfo | null>(null);
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
        setNewQuality(info.qualities[0]);
      } else {
        setNewQuality('best');
      }
    } catch (e) {
      console.error(e);
      alert(t('downloads.modal.parseError') + e);
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
      await AddTaskGo(
        newUrl,
        newQuality,
        newFormat,
        newDir,
        videoInfo?.title || '',
        videoInfo?.thumbnail || ''
      );
      // Reset form
      setNewUrl('');
      setNewQuality('best');
      setNewFormat('webm');
      setVideoInfo(null);
      if (defaultDir) setNewDir(defaultDir);

      onAdded?.();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <PageContainer
      viewClass="px-10"
      header={
        <PageHeader title={t('downloads.modal.title')} subtitle={t('downloads.startDownloading')} />
      }
    >
      <Card variant="borderless" className="shadow-sm">
        <div className="space-y-6">
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('downloads.modal.videoUrl')}</label>
            <Input.Search
              value={newUrl}
              onChange={(e) => {
                setNewUrl(e.target.value);
                setVideoInfo(null);
              }}
              loading={isFetchingInfo}
              placeholder="https://www.youtube.com/watch?v=..."
              enterButton={t('downloads.modal.analyze')}
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
                    {t('downloads.modal.duration')}
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
                <label className="text-sm font-medium">{t('downloads.modal.quality')}</label>
                <Select
                  value={newQuality}
                  onChange={setNewQuality}
                  style={{ width: '100%' }}
                  size="large"
                  options={(videoInfo?.qualities?.length ? videoInfo.qualities : ['best']).map(
                    (q) => ({ label: q, value: q })
                  )}
                  placeholder={t('downloads.modal.bestQuality')}
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">{t('downloads.modal.format')}</label>
                <Select
                  value={newFormat}
                  onChange={setNewFormat}
                  style={{ width: '100%' }}
                  size="large"
                  options={[
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
                <label className="text-sm font-medium">{t('downloads.modal.saveTo')}</label>
                <Space.Compact style={{ width: '100%' }} size="large">
                  <Input value={newDir} readOnly />
                  <Button icon={<FolderOutlined />} onClick={handleChooseDir}>
                    {t('downloads.modal.chooseDir')}
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
                  {t('downloads.modal.start')}
                </Button>
              </div>
            </div>
          )}
        </div>
      </Card>
    </PageContainer>
  );
}
