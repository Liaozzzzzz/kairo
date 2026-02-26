import { Modal, Button, Tag, message, Tooltip, Space, Table, Empty } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DeleteOutlined, SyncOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useState, useEffect } from 'react';
import {
  GetVideoSubtitles,
  DeleteSubtitle,
  RegenerateSubtitle,
  FetchSubtitles,
} from '@root/wailsjs/go/main/App';
import { useVideoStore } from '@/store/useVideoStore';
import { useShallow } from 'zustand/react/shallow';
import dayjs from 'dayjs';
import { SubtitleSource, SubtitleStatus, VideoSubtitle } from '@/types';

import SubtitlesImport from './SubtitlesImport';
import SubtitlesTranslate from './SubtitlesTranslate';
import SubtitlesPreview from './SubtitlesPreview';
import SubtitlesEdit from './SubtitlesEdit';

interface VideoDetailModalProps {
  videoId: string;
  isOpen: boolean;
  onClose: () => void;
}

export default function SubtitlesModal({ videoId, isOpen, onClose }: VideoDetailModalProps) {
  const { t } = useTranslation();
  const [subtitles, setSubtitles] = useState<VideoSubtitle[]>([]);
  const [subtitlesLoading, setSubtitlesLoading] = useState(false);

  const video = useVideoStore(useShallow((state) => state.videos.find((v) => v.id === videoId)));

  useEffect(() => {
    if (!video && isOpen) {
      onClose();
    }
  }, [video, isOpen, onClose]);

  const loadSubtitles = async () => {
    if (!videoId) return;
    setSubtitlesLoading(true);
    try {
      const data = await GetVideoSubtitles(videoId);
      setSubtitles(data || []);
    } catch (error) {
      console.error('Failed to load subtitles:', error);
    } finally {
      setSubtitlesLoading(false);
    }
  };

  useEffect(() => {
    if (isOpen && videoId) {
      loadSubtitles();
    } else {
      setSubtitles([]);
    }
  }, [isOpen, videoId]);

  const renderSubtitleStatus = (statusValue: SubtitleStatus) => {
    if (statusValue === SubtitleStatus.Generating) {
      return <Tag color="processing">{t('videos.subtitles.status.generating')}</Tag>;
    }
    if (statusValue === SubtitleStatus.Pending) {
      return <Tag color="default">{t('videos.subtitles.status.pending')}</Tag>;
    }
    if (statusValue === SubtitleStatus.Success) {
      return <Tag color="success">{t('videos.subtitles.status.success')}</Tag>;
    }
    if (statusValue === SubtitleStatus.Failed) {
      return <Tag color="error">{t('videos.subtitles.status.failed')}</Tag>;
    }
    return <Tag>{statusValue}</Tag>;
  };

  const renderSubtitleSource = (sourceValue: SubtitleSource) => {
    if (sourceValue === SubtitleSource.Builtin) {
      return <Tag>{t('videos.subtitles.source.builtin')}</Tag>;
    }
    if (sourceValue === SubtitleSource.ASR) {
      return <Tag color="blue">{t('videos.subtitles.source.asr')}</Tag>;
    }
    if (sourceValue === SubtitleSource.Manual) {
      return <Tag color="purple">{t('videos.subtitles.source.manual')}</Tag>;
    }
    if (sourceValue === SubtitleSource.Translation) {
      return <Tag color="orange">{t('videos.subtitles.source.translation')}</Tag>;
    }
    return <Tag>{sourceValue}</Tag>;
  };

  const handleDeleteSubtitle = (subtitle: VideoSubtitle) => {
    Modal.confirm({
      title: t('videos.subtitles.delete.title'),
      content: t('videos.subtitles.delete.confirm'),
      okText: t('common.delete'),
      okButtonProps: { danger: true },
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await DeleteSubtitle(subtitle.id);
          message.success(t('videos.subtitles.delete.success'));
          await loadSubtitles();
        } catch (error) {
          console.error('Failed to delete subtitle:', error);
          message.error(t('videos.subtitles.delete.failed'));
        }
      },
    });
  };

  const handleAutoDetectLanguage = async () => {
    try {
      setSubtitlesLoading(true);
      await FetchSubtitles(videoId);
      message.success(t('videos.subtitles.auto_detect.success'));
      await loadSubtitles();
    } catch (error) {
      console.error('Failed to auto detect subtitle language:', error);
      message.error(t('videos.subtitles.auto_detect.failed'));
    } finally {
      setSubtitlesLoading(false);
    }
  };

  const handleRegenerateSubtitle = async (subtitle: VideoSubtitle) => {
    try {
      await RegenerateSubtitle(subtitle.id);
      message.success(t('videos.subtitles.regenerate.success'));
      await loadSubtitles();
    } catch (error) {
      console.error('Failed to regenerate subtitle:', error);
      message.error(t('videos.subtitles.regenerate.failed'));
    }
  };

  if (!video) return null;

  const columns: ColumnsType<VideoSubtitle> = [
    {
      title: t('videos.subtitles.language'),
      dataIndex: 'language',
      key: 'language',
      render: (_: string, record: VideoSubtitle) =>
        record.language?.trim() ? record.language : t('videos.subtitle_language_detecting'),
    },
    {
      title: t('videos.subtitles.source.title'),
      dataIndex: 'source',
      key: 'source',
      render: (value: SubtitleSource) => renderSubtitleSource(value),
    },
    {
      title: t('videos.subtitles.status.title'),
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (value: SubtitleStatus) => renderSubtitleStatus(value),
    },
    {
      title: t('videos.subtitles.created_at'),
      dataIndex: 'created_at',
      key: 'created_at',
      width: 160,
      render: (value: number) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: t('videos.subtitles.updated_at'),
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 160,
      render: (value: number) => (value ? dayjs(value).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: t('videos.subtitles.actions'),
      key: 'actions',
      width: 160,
      render: (_: unknown, record: VideoSubtitle) => {
        const isGenerating = record.status === SubtitleStatus.Generating;
        const canRegenerate =
          [SubtitleSource.ASR, SubtitleSource.Translation].includes(record.source) &&
          [SubtitleStatus.Success, SubtitleStatus.Failed].includes(record.status);
        return (
          <Space size="small">
            <SubtitlesPreview filePath={record.file_path} />
            <SubtitlesEdit subtitle={record} onSuccess={loadSubtitles} />
            <SubtitlesTranslate subtitle={record} videoId={video.id} onSuccess={loadSubtitles} />
            <Tooltip title={t('videos.subtitles.regenerate.title')}>
              <Button
                size="small"
                icon={<SyncOutlined />}
                onClick={() => handleRegenerateSubtitle(record)}
                disabled={!canRegenerate}
              />
            </Tooltip>
            <Tooltip title={t('common.delete')}>
              <Button
                size="small"
                danger
                icon={<DeleteOutlined />}
                onClick={() => handleDeleteSubtitle(record)}
                disabled={isGenerating}
              />
            </Tooltip>
          </Space>
        );
      },
    },
  ];

  return (
    <Modal
      title={`${t('videos.subtitles.title')} - ${video.title}`}
      open={isOpen}
      onCancel={onClose}
      destroyOnHidden
      width={860}
      footer={<Button onClick={onClose}>{t('common.close')}</Button>}
    >
      <div className="space-y-3 mt-4">
        <Space size="small">
          <SubtitlesImport videoId={video.id} onSuccess={loadSubtitles} />
        </Space>
        <Table
          rowKey="id"
          size="middle"
          scroll={{ y: 60 * 5 }}
          dataSource={subtitles}
          columns={columns}
          loading={subtitlesLoading}
          pagination={false}
          locale={{
            emptyText: (
              <Empty
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={
                  <>
                    <div>{t('videos.subtitles.empty')}</div>
                    <div className="flex items-center justify-center">
                      <SubtitlesImport type="link" videoId={video.id} onSuccess={loadSubtitles} />
                      <div>{t('common.or')}</div>
                      <Button type="link" onClick={handleAutoDetectLanguage}>
                        {t('videos.subtitles.auto_detect.title')}
                      </Button>
                    </div>
                  </>
                }
              />
            ),
          }}
        />
      </div>
    </Modal>
  );
}
