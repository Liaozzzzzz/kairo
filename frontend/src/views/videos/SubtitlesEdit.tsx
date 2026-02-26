import { useState, useRef } from 'react';
import { Modal, Button, Tooltip, message, Input, Form } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { ReadFileContent, UpdateSubtitle } from '@root/wailsjs/go/main/App';
import { SubtitleStatus, VideoSubtitle } from '@/types';
import SubtitlesLanguageSelect, { SubtitlesLanguageSelectRef } from './SubtitlesLanguageSelect';

interface SubtitlesEditProps {
  subtitle: VideoSubtitle;
  onSuccess: () => void;
}

export default function SubtitlesEdit({ subtitle, onSuccess }: SubtitlesEditProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [content, setContent] = useState('');
  const [language, setLanguage] = useState<string | null>('');
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const languageSelectRef = useRef<SubtitlesLanguageSelectRef>(null);

  const handleOpen = async () => {
    if (!subtitle.file_path) return;
    try {
      setLoading(true);
      const fileContent = await ReadFileContent(subtitle.file_path);
      setContent(fileContent);
      setLanguage(subtitle.language || '');
      setOpen(true);

      // Add existing language to options if needed
      if (subtitle.language) {
        setTimeout(() => {
          languageSelectRef.current?.addLanguage(subtitle.language);
        }, 100);
      }
    } catch (error) {
      console.error('Failed to read file:', error);
      message.error(t('videos.open_failed'));
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async () => {
    if (!content) return;
    if (!language) {
      message.error(t('videos.subtitles.select.language_placeholder'));
      return;
    }
    try {
      setSaving(true);
      await UpdateSubtitle(subtitle.id, content, language);
      message.success(t('common.save_success'));
      setOpen(false);
      onSuccess();
    } catch (error) {
      console.error('Failed to save subtitle:', error);
      message.error(t('common.save_failed'));
    } finally {
      setSaving(false);
    }
  };

  return (
    <>
      <Tooltip title={t('common.edit')}>
        <Button
          size="small"
          icon={<EditOutlined />}
          onClick={handleOpen}
          disabled={!subtitle.file_path || subtitle.status !== SubtitleStatus.Success}
          loading={loading}
        />
      </Tooltip>

      <Modal
        centered
        open={open}
        title={t('videos.subtitles.edit')}
        onCancel={() => setOpen(false)}
        onOk={handleSave}
        confirmLoading={saving}
        width={800}
        destroyOnHidden
        okText={t('common.save')}
        cancelText={t('common.cancel')}
      >
        <Form layout="vertical">
          <Form.Item label={t('videos.subtitles.language')} required>
            <SubtitlesLanguageSelect
              ref={languageSelectRef}
              value={language}
              onChange={setLanguage}
            />
          </Form.Item>
          <Form.Item label={t('videos.subtitles.import.content')} required>
            <Input.TextArea
              value={content}
              onChange={(e) => setContent(e.target.value)}
              rows={16}
              className="font-mono text-sm"
              style={{ whiteSpace: 'pre', overflowX: 'auto' }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
