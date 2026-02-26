import { useState } from 'react';
import { Modal, Button, Tooltip, message } from 'antd';
import { TranslationOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { TranslateSubtitle } from '@root/wailsjs/go/main/App';
import { SubtitleStatus, VideoSubtitle } from '@/types';

import SubtitlesLanguageSelect from './SubtitlesLanguageSelect';

interface SubtitlesTranslateProps {
  subtitle: VideoSubtitle;
  videoId: string;
  onSuccess: () => void;
}

export default function SubtitlesTranslate({
  subtitle,
  videoId,
  onSuccess,
}: SubtitlesTranslateProps) {
  const { t } = useTranslation();
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [targetLanguage, setTargetLanguage] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const handleOpen = () => {
    setTargetLanguage(null);
    setIsModalOpen(true);
  };

  const handleConfirm = async () => {
    if (!targetLanguage) {
      message.error(t('videos.subtitles.select.language_placeholder'));
      return;
    }

    if (targetLanguage.trim() === subtitle.language) {
      message.error(t('videos.subtitles.translate.same_language'));
      return;
    }

    setLoading(true);
    try {
      await TranslateSubtitle({
        subtitle_id: subtitle.id,
        video_id: videoId,
        target_language: targetLanguage.trim(),
      });
      message.success(t('videos.subtitles.translate.success'));
      setIsModalOpen(false);
      onSuccess();
    } catch (error) {
      console.error('Failed to translate subtitle:', error);
      message.error(t('videos.subtitles.translate.failed'));
    } finally {
      setLoading(false);
    }
  };

  const canTranslate = subtitle.status === SubtitleStatus.Success;

  return (
    <>
      <Tooltip title={t('videos.subtitles.translate.title')}>
        <Button
          size="small"
          icon={<TranslationOutlined />}
          onClick={handleOpen}
          disabled={!canTranslate}
        />
      </Tooltip>

      <Modal
        open={isModalOpen}
        title={t('videos.subtitles.translate.title')}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
        onCancel={() => setIsModalOpen(false)}
        onOk={handleConfirm}
        confirmLoading={loading}
        destroyOnHidden
      >
        <SubtitlesLanguageSelect value={targetLanguage} onChange={setTargetLanguage} />
      </Modal>
    </>
  );
}
