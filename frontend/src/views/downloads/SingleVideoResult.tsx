import { useEffect, useState } from 'react';
import { Select, Image, Button, Divider } from 'antd';
import { DownloadOutlined, PlayCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { models } from '@root/wailsjs/go/models';
import { ImageFallback } from '@/data/variables';
import { useSettingStore } from '@/store/useSettingStore';
import { useShallow } from 'zustand/react/shallow';
import DownloadDir from '@/components/DownloadDir';

interface SingleVideoResultProps {
  videoInfo: models.VideoInfo;
  onStartDownload: ({
    newDir,
    newQuality,
    newFormat,
  }: {
    newDir: string;
    newQuality: string;
    newFormat: string;
  }) => void;
}

const SingleVideoResult = ({ videoInfo, onStartDownload }: SingleVideoResultProps) => {
  const { t } = useTranslation();
  const defaultDir = useSettingStore(useShallow((state) => state.defaultDir));
  const [newDir, setNewDir] = useState(defaultDir);

  const [newQuality, setNewQuality] = useState('');
  const [newFormat, setNewFormat] = useState('original');

  useEffect(() => {
    if (videoInfo.qualities && videoInfo.qualities.length > 0) {
      setNewQuality(videoInfo.qualities[0].value);
    } else {
      setNewQuality('');
    }
  }, [videoInfo.qualities]);

  return (
    <>
      <div className="space-y-3">
        <div className="bg-slate-50 border border-slate-200 p-4 rounded-lg flex gap-4 shadow-sm animate-in fade-in slide-in-from-top-2 duration-300">
          <div className="flex-shrink-0 w-40 aspect-video rounded-md overflow-hidden border border-black/10 bg-gray-100">
            {videoInfo.thumbnail ? (
              <Image
                src={videoInfo.thumbnail}
                referrerPolicy="no-referrer"
                className="object-cover"
                alt=""
                width="100%"
                height="100%"
                fallback={ImageFallback}
              />
            ) : (
              <div className="w-full h-full flex items-center justify-center text-gray-300">
                <PlayCircleOutlined className="text-2xl" />
              </div>
            )}
          </div>
          <div className="flex-1 min-w-0 flex flex-col justify-center">
            <div className="font-medium text-lg truncate text-foreground" title={videoInfo.title}>
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
        <div className="grid grid-cols-2 gap-4 animate-in fade-in slide-in-from-top-2 duration-300 delay-75">
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('downloads.quality')}</label>
            <Select
              value={newQuality}
              onChange={setNewQuality}
              style={{ width: '100%' }}
              options={(videoInfo.qualities || []).map((q) => ({
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
              options={[
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
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('downloads.saveTo')}</label>
            <DownloadDir defaultDir={newDir} setNewDir={setNewDir} />
          </div>
        </div>
      </div>
      <Divider size="large" />
      <div className="flex justify-end">
        <Button
          type="primary"
          onClick={() => onStartDownload({ newDir, newQuality, newFormat })}
          disabled={!newDir || !videoInfo}
          icon={<DownloadOutlined />}
          className="px-8"
        >
          {t('downloads.start')}
        </Button>
      </div>
    </>
  );
};

export default SingleVideoResult;
