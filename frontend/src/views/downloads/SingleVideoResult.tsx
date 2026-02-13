import { useEffect, useState } from 'react';
import { Select, Image, Button, Divider, Radio, Slider } from 'antd';
import {
  DownloadOutlined,
  PlayCircleOutlined,
  ClockCircleOutlined,
  YoutubeOutlined,
  CustomerServiceOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { models } from '@root/wailsjs/go/models';
import { ImageFallback } from '@/data/variables';
import { useSettingStore } from '@/store/useSettingStore';
import { useShallow } from 'zustand/react/shallow';
import DownloadDir from '@/components/DownloadDir';

import { TrimMode } from '@/data/variables';

interface SingleVideoResultProps {
  videoInfo: models.VideoInfo;
  onStartDownload: ({
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
  }) => void;
}

const SingleVideoResult = ({ videoInfo, onStartDownload }: SingleVideoResultProps) => {
  const { t } = useTranslation();
  const defaultDir = useSettingStore(useShallow((state) => state.defaultDir));
  const [newDir, setNewDir] = useState(defaultDir);

  const [newQuality, setNewQuality] = useState('');
  const [newFormat, setNewFormat] = useState('original');
  const [trimRange, setTrimRange] = useState<[number, number]>([0, 0]);
  const [trimMode, setTrimMode] = useState<TrimMode>(TrimMode.None);

  useEffect(() => {
    if (videoInfo.qualities && videoInfo.qualities.length > 0) {
      setNewQuality(videoInfo.qualities[0].value);
    } else {
      setNewQuality('');
    }
    setTrimRange([0, Math.floor(videoInfo.duration || 0)]);
  }, [videoInfo]);

  const formatSeconds = (seconds: number) => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = Math.floor(seconds % 60);
    return `${h.toString().padStart(2, '0')}:${m.toString().padStart(2, '0')}:${s.toString().padStart(2, '0')}`;
  };

  return (
    <>
      <div className="space-y-3">
        <div className="bg-muted/30 border border-border p-4 rounded-lg flex gap-4 shadow-sm animate-in fade-in slide-in-from-top-2 duration-300">
          <div className="flex-shrink-0 w-40 aspect-video rounded-md overflow-hidden border border-black/10 dark:border-white/10 bg-gray-100 dark:bg-black/20">
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
              <div className="w-full h-full flex items-center justify-center text-gray-300 dark:text-muted-foreground/50">
                <PlayCircleOutlined className="text-2xl" />
              </div>
            )}
          </div>
          <div className="flex-1 min-w-0 flex flex-col justify-between py-1">
            <div
              className="font-semibold text-lg leading-tight truncate text-foreground mb-1"
              title={videoInfo.title}
            >
              {videoInfo.title}
            </div>

            <div className="flex items-center flex-wrap gap-2">
              {/* Duration Tag */}
              <span className="bg-primary/10 dark:bg-primary/20 px-2.5 py-1 rounded-md border border-primary/20 text-xs text-primary flex items-center gap-1.5 font-medium transition-colors hover:bg-primary/15">
                <ClockCircleOutlined />
                <span>
                  {Math.floor(videoInfo.duration / 60)}:
                  {String(Math.floor(videoInfo.duration % 60)).padStart(2, '0')}
                </span>
              </span>

              {/* Type Tag */}
              <span className="bg-orange-50 dark:bg-orange-500/10 px-2.5 py-1 rounded-md border border-orange-200 dark:border-orange-500/20 text-xs text-orange-600 dark:text-orange-400 flex items-center gap-1.5 font-medium">
                {videoInfo.qualities?.some((q) => q.audio_bytes > 0 && q.video_bytes === 0) ? (
                  <CustomerServiceOutlined />
                ) : (
                  <YoutubeOutlined />
                )}
                <span>
                  {videoInfo.qualities?.some((q) => q.audio_bytes > 0 && q.video_bytes === 0)
                    ? 'Audio'
                    : 'Video'}
                </span>
              </span>

              {/* Best Quality Tag */}
              {videoInfo.qualities && videoInfo.qualities.length > 0 && (
                <span className="bg-blue-50 dark:bg-blue-500/10 px-2.5 py-1 rounded-md border border-blue-200 dark:border-blue-500/20 text-xs text-blue-600 dark:text-blue-400 flex items-center gap-1.5 font-medium">
                  <span className="font-bold">HD</span>
                  <span>{videoInfo.qualities[0].label.split(' ')[0]}</span>
                </span>
              )}
            </div>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-4 animate-in fade-in slide-in-from-top-2 duration-300 delay-75">
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">{t('downloads.quality')}</label>
            <Select
              value={newQuality}
              onChange={setNewQuality}
              style={{ width: '100%' }}
              options={(videoInfo.qualities || []).map((q) => ({
                label: (
                  <div className="flex justify-between items-center w-full gap-4">
                    <span>{q.label}</span>
                    <span className="text-gray-400 dark:text-muted-foreground text-xs font-normal">
                      {q.total_size}
                    </span>
                  </div>
                ),
                value: q.value,
              }))}
              placeholder={t('downloads.bestQuality')}
            />
          </div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">{t('downloads.format')}</label>
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
            <label className="text-sm font-medium text-foreground">{t('downloads.saveTo')}</label>
            <DownloadDir defaultDir={newDir} setNewDir={setNewDir} />
          </div>
          <div></div>
          <div className="space-y-2">
            <label className="text-sm font-medium text-foreground">
              {t('downloads.trimVideo')}
            </label>
            <Radio.Group
              className="flex items-center h-8"
              value={trimMode}
              onChange={(e) => setTrimMode(e.target.value)}
            >
              <Radio value={TrimMode.None}>{t('downloads.noTrim')}</Radio>
              <Radio value={TrimMode.Overwrite}>{t('downloads.trimOverwrite')}</Radio>
              <Radio value={TrimMode.Keep}>{t('downloads.trimKeep')}</Radio>
            </Radio.Group>
          </div>
          {trimMode !== TrimMode.None && (
            <div className="space-y-2">
              <label className="text-sm font-medium text-foreground">
                {t('downloads.trimRange')}
              </label>
              <div className="w-full flex items-center h-8">
                <Slider
                  range
                  className="w-full"
                  min={0}
                  max={Math.floor(videoInfo.duration || 0)}
                  value={trimRange}
                  onChange={(val) => setTrimRange(val as [number, number])}
                  tooltip={{ formatter: (val) => formatSeconds(val || 0) }}
                />
              </div>
            </div>
          )}
        </div>
      </div>
      <Divider size="large" />
      <div className="flex justify-end">
        <Button
          type="primary"
          onClick={() =>
            onStartDownload({
              newDir,
              newQuality,
              newFormat,
              trimStart: formatSeconds(trimRange[0]),
              trimEnd: formatSeconds(trimRange[1]),
              trimMode,
            })
          }
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
