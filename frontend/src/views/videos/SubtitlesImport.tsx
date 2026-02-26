import { Modal, Button, Input, message, Form } from 'antd';
import { UploadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useRef, useState } from 'react';
import { ChooseFile, ReadFileContent, SaveSubtitleContent } from '@root/wailsjs/go/main/App';

import SubtitlesLanguageSelect, { SubtitlesLanguageSelectRef } from './SubtitlesLanguageSelect';

interface SubtitlesImportProps {
  videoId: string;
  onSuccess: () => void;
  type?: 'link' | 'button';
}

export default function SubtitlesImport({
  videoId,
  onSuccess,
  type = 'button',
}: SubtitlesImportProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [content, setContent] = useState('');
  const [language, setLanguage] = useState<string | null>(null);
  const languageSelectRef = useRef<SubtitlesLanguageSelectRef>(null);

  const detectLanguage = (filePath: string) => {
    // Extract filename from path
    const filename = filePath.split(/[/\\]/).pop();
    if (!filename) return '';

    // Remove extension
    const nameWithoutExt = filename.replace(/\.(srt|vtt|ass|ssa)$/i, '');

    // Check for language code at the end (e.g., video.en, video.zh-CN)
    const parts = nameWithoutExt.split('.');
    if (parts.length > 1) {
      const lastPart = parts[parts.length - 1];
      // Simple heuristic: 2-3 chars, possibly with region (e.g., en, zh-Hans)
      if (/^[a-z]{2,3}(-[a-z]{2,4})?$/i.test(lastPart)) {
        return lastPart;
      }
    }
    return '';
  };

  const handleImport = async () => {
    try {
      const filePath = await ChooseFile([
        {
          displayName: 'Subtitle Files (*.srt;*.vtt;*.ass;*.ssa;*.json)',
          pattern: '*.srt;*.vtt;*.ass;*.ssa;*.json',
        },
        { displayName: 'All Files (*)', pattern: '*' },
      ]);
      if (!filePath) return;

      const fileContent = await ReadFileContent(filePath);
      setContent(fileContent);

      const detectedLang = detectLanguage(filePath);
      if (detectedLang) {
        setLanguage(detectedLang);
        languageSelectRef.current?.addLanguage(detectedLang);
      } else {
        setLanguage('');
      }

      setOpen(true);
    } catch (error) {
      console.error('Failed to import subtitle:', error);
      message.error(t('videos.subtitles.import.failed'));
    }
  };

  const handleSave = async () => {
    if (!videoId) return;
    if (!language) {
      message.error(t('videos.subtitles.select.language_placeholder'));
      return;
    }

    if (!content) {
      message.error(t('videos.subtitles.import.content_placeholder'));
      return;
    }

    setLoading(true);
    try {
      await SaveSubtitleContent(videoId, language, content);
      message.success(t('videos.subtitles.import.success'));
      setOpen(false);
      setContent('');
      setLanguage('');
      onSuccess();
    } catch (error) {
      console.error('Failed to save subtitle:', error);
      message.error(t('videos.subtitles.import.failed'));
    } finally {
      setLoading(false);
    }
  };

  return (
    <>
      {type === 'button' ? (
        <Button type="primary" onClick={handleImport} icon={<UploadOutlined />}>
          {t('videos.subtitles.import.title')}
        </Button>
      ) : (
        <Button type="link" onClick={handleImport}>
          {t('videos.subtitles.import.title')}
        </Button>
      )}

      <Modal
        open={open}
        title={t('videos.subtitles.import.title')}
        okText={t('common.save')}
        cancelText={t('common.cancel')}
        onCancel={() => setOpen(false)}
        onOk={handleSave}
        confirmLoading={loading}
        destroyOnHidden
      >
        <Form layout="vertical">
          <Form.Item label={t('videos.subtitles.import.language')} required>
            <SubtitlesLanguageSelect
              ref={languageSelectRef}
              value={language}
              onChange={setLanguage}
            />
          </Form.Item>
          <Form.Item label={t('videos.subtitles.import.content')} required>
            <Input.TextArea
              rows={10}
              value={content}
              onChange={(e) => setContent(e.target.value)}
              placeholder={t('videos.subtitles.import.content_placeholder')}
            />
          </Form.Item>
        </Form>
      </Modal>
    </>
  );
}
