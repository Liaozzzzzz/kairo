import { Drawer, Button, Tag, Divider, Spin, Typography, message, Tooltip } from 'antd';
import {
  RobotOutlined,
  InfoCircleOutlined,
  ScissorOutlined,
  PlayCircleOutlined,
  FolderOpenOutlined,
} from '@ant-design/icons';
import { formatDuration, formatBytes } from '@/lib/utils';
import { useTranslation } from 'react-i18next';
import { useState, useEffect } from 'react';
import {
  AnalyzeVideo,
  ClipVideo,
  OpenFile,
  ShowInFolder,
  GetVideoHighlights,
} from '@root/wailsjs/go/main/App';
import { useVideoStore } from '@/store/useVideoStore';
import { useShallow } from 'zustand/react/shallow';
import dayjs from 'dayjs';
import { models } from '@root/wailsjs/go/models';

const { Title, Paragraph } = Typography;

interface VideoDetailModalProps {
  videoId: string;
  isOpen: boolean;
  onClose: () => void;
}

export default function VideoDetailModal({ videoId, isOpen, onClose }: VideoDetailModalProps) {
  const { t } = useTranslation();
  const [analyzing, setAnalyzing] = useState(false);
  const [clippingIndex, setClippingIndex] = useState<number | null>(null);
  const [highlights, setHighlights] = useState<models.AIHighlight[]>([]);

  // Use selector to get the specific video from store
  const video = useVideoStore(useShallow((state) => state.videos.find((v) => v.id === videoId)));

  // If video not found (e.g. deleted), close modal
  useEffect(() => {
    if (!video && isOpen) {
      onClose();
    }
  }, [video, isOpen, onClose]);

  const status = video?.status;

  // Reset analyzing state when status changes to completed or failed
  useEffect(() => {
    if (status === 'completed' || status === 'failed') {
      setAnalyzing(false);
    }
  }, [status]);

  // Fetch highlights when video is open and status is completed
  useEffect(() => {
    if (isOpen && videoId && status === 'completed') {
      GetVideoHighlights(videoId).then((data) => {
        setHighlights(data || []);
      });
    } else {
      setHighlights([]);
    }
  }, [isOpen, videoId, status]);

  // Listen for highlight updates via video store update (when clip happens)
  useEffect(() => {
    if (video?.highlights) {
      setHighlights(video.highlights);
    }
  }, [video?.highlights]);

  const handleAnalyze = async () => {
    if (!video) return;
    setAnalyzing(true);
    try {
      await AnalyzeVideo(video.id);
      // Status update will come via store -> selector
    } catch (error) {
      console.error('Analysis failed:', error);
      setAnalyzing(false);
    }
  };

  const handleClip = async (index: number, highlightID: string, start: string, end: string) => {
    if (!video) return;
    setClippingIndex(index);
    try {
      await ClipVideo(video.id, highlightID, start, end);
      message.success(t('videos.clip_success'));
    } catch (error) {
      console.error('Failed to clip:', error);
      message.error(t('videos.clip_failed'));
    } finally {
      setClippingIndex(null);
    }
  };

  const handlePlayClip = async (path: string) => {
    try {
      await OpenFile(path);
    } catch (error) {
      console.error('Failed to open file:', error);
      message.error(t('videos.open_failed'));
    }
  };

  const handleShowInFolder = async (path: string) => {
    try {
      await ShowInFolder(path);
    } catch (error) {
      console.error('Failed to show in folder:', error);
      message.error(t('videos.open_folder_failed'));
    }
  };

  if (!video) return null;

  return (
    <Drawer
      title={
        <div className="flex items-center gap-2">
          <span className="truncate max-w-md" title={video.title}>
            {video.title}
          </span>
        </div>
      }
      open={isOpen}
      onClose={onClose}
      destroyOnHidden
      size={800}
      footer={
        <div className="flex justify-end gap-2">
          <Button key="close" onClick={onClose}>
            {t('common.close')}
          </Button>
          <Button
            key="analyze"
            type="primary"
            icon={<RobotOutlined />}
            loading={analyzing}
            disabled={status === 'processing'}
            onClick={handleAnalyze}
          >
            {status === 'completed' ? t('videos.reanalyze') : t('videos.analyze')}
          </Button>
        </div>
      }
    >
      <div className="flex flex-col md:flex-row gap-6 h-full">
        <div className="w-full md:w-1/3 space-y-4 h-full overflow-hidden pr-2 flex flex-col">
          <div className="aspect-video bg-slate-100 rounded-lg overflow-hidden border border-slate-200 shrink-0">
            <img
              src={video.thumbnail || 'src/assets/images/icon.png'}
              alt={video.title}
              className="w-full h-full object-cover"
              onError={(e) => ((e.target as HTMLImageElement).src = '')}
            />
          </div>

          <div className="space-y-2 text-sm shrink-0">
            <div className="flex justify-between">
              <span className="text-slate-500">{t('common.duration')}:</span>
              <span className="font-medium">{formatDuration(video.duration)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">{t('common.size')}:</span>
              <span className="font-medium">{formatBytes(video.size)}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">{t('common.resolution')}:</span>
              <span className="font-medium">{video.resolution}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">{t('common.format')}:</span>
              <span className="font-medium">{video.format}</span>
            </div>
            <div className="flex justify-between">
              <span className="text-slate-500">{t('common.date')}:</span>
              <span className="font-medium">
                {dayjs(video.created_at * 1000).format('YYYY-MM-DD HH:mm')}
              </span>
            </div>
          </div>

          {video.subtitles && video.subtitles.length > 0 && (
            <div className="shrink-0">
              <Divider className="my-3" />
              <div className="text-slate-500 mb-2">{t('videos.subtitles')}</div>
              <div className="flex flex-wrap gap-1">
                {video.subtitles.map((sub, idx) => (
                  <Tag key={idx}>{sub}</Tag>
                ))}
              </div>
            </div>
          )}

          {video.description && (
            <div className="mt-4 flex-1 overflow-y-auto custom-scrollbar">
              <Title level={5} className="text-sm sticky top-0 bg-white z-10 py-1">
                {t('common.description')}
              </Title>
              <Paragraph className="text-slate-600 text-sm whitespace-pre-wrap">
                {video.description}
              </Paragraph>
            </div>
          )}
        </div>

        <div className="w-full md:w-2/3 h-full overflow-y-auto pl-2 custom-scrollbar relative">
          <Title
            level={5}
            className="flex items-center gap-2 !mt-0 sticky top-0 bg-white z-10 py-2 border-b border-slate-100"
          >
            <RobotOutlined className="text-blue-500" />
            {t('videos.ai_analysis')}
          </Title>

          <div className="bg-slate-50 p-4 rounded-lg border border-slate-200 min-h-[300px]">
            {status === 'completed' ? (
              <div className="space-y-4">
                {video.tags && video.tags.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {video.tags.map((tag, idx) => (
                      <Tag color="blue" key={idx}>
                        #{tag}
                      </Tag>
                    ))}
                  </div>
                )}

                {video.evaluation && (
                  <div className="bg-white p-3 rounded border border-blue-100">
                    <div className="text-xs text-slate-400 uppercase tracking-wider mb-1">
                      {t('videos.evaluation')}
                    </div>
                    <div className="font-medium text-blue-800">{video.evaluation}</div>
                  </div>
                )}

                {highlights && highlights.length > 0 && (
                  <div>
                    <div className="text-xs text-slate-400 uppercase tracking-wider mb-1">
                      {t('videos.highlights')}
                    </div>
                    <div className="grid gap-2">
                      {highlights.map((highlight, idx) => (
                        <div
                          key={idx}
                          className="bg-white p-3 rounded border border-slate-200 flex justify-between items-center gap-4"
                        >
                          <div>
                            <div className="font-medium text-slate-700 text-sm">
                              {highlight.description}
                            </div>
                            <div className="text-xs text-slate-500 font-mono mt-1">
                              {highlight.start} - {highlight.end}
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            {highlight.file_path ? (
                              <>
                                <Tooltip title={t('videos.play_clip')}>
                                  <Button
                                    size="small"
                                    icon={<PlayCircleOutlined />}
                                    onClick={() => handlePlayClip(highlight.file_path!)}
                                  />
                                </Tooltip>
                                <Tooltip title={t('videos.show_in_folder')}>
                                  <Button
                                    size="small"
                                    icon={<FolderOpenOutlined />}
                                    onClick={() => handleShowInFolder(highlight.file_path!)}
                                  />
                                </Tooltip>
                              </>
                            ) : (
                              <Button
                                size="small"
                                icon={<ScissorOutlined />}
                                loading={clippingIndex === idx}
                                onClick={() =>
                                  handleClip(idx, highlight.id, highlight.start, highlight.end)
                                }
                              >
                                {t('videos.clip')}
                              </Button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                <div>
                  <div className="text-xs text-slate-400 uppercase tracking-wider mb-1">
                    {t('videos.summary')}
                  </div>
                  <Paragraph className="text-slate-700 leading-relaxed whitespace-pre-wrap">
                    {video.summary}
                  </Paragraph>
                </div>
              </div>
            ) : (
              <div className="h-full flex flex-col items-center justify-center text-slate-400 space-y-3 pt-10">
                {analyzing || status === 'processing' ? (
                  <>
                    <Spin size="large" />
                    <div className="text-center">
                      <div>{t('videos.analyzing')}</div>
                      <div className="text-xs text-slate-400 mt-1">{t('videos.wait_moment')}</div>
                    </div>
                  </>
                ) : (
                  <>
                    <InfoCircleOutlined className="text-4xl opacity-50" />
                    <div className="text-center">
                      {status === 'failed' ? (
                        <div className="text-red-500">
                          <div>{t('videos.analysis_failed')}</div>
                          <div className="text-xs mt-1">{t('videos.click_retry')}</div>
                        </div>
                      ) : (
                        <div>
                          <div>{t('videos.no_analysis')}</div>
                          <div className="text-xs mt-1">{t('videos.click_analyze')}</div>
                        </div>
                      )}
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        </div>
      </div>
    </Drawer>
  );
}
