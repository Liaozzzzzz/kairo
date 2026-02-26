import { useState } from 'react';
import { Modal, Button, Tooltip, message } from 'antd';
import { EyeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { ReadFileContent } from '@root/wailsjs/go/main/App';

interface SubtitlesPreviewProps {
  filePath: string;
}

export default function SubtitlesPreview({ filePath }: SubtitlesPreviewProps) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [content, setContent] = useState('');
  const [loading, setLoading] = useState(false);

  const handlePreview = async () => {
    if (!filePath) return;
    try {
      setLoading(true);
      const fileContent = await ReadFileContent(filePath);
      setContent(fileContent);
      setOpen(true);
    } catch (error) {
      console.error('Failed to read file:', error);
      message.error(t('videos.subtitles.open.failed'));
    } finally {
      setLoading(false);
    }
  };

  const filename = filePath ? filePath.split(/[/\\]/).pop() : '';

  return (
    <>
      <Tooltip title={t('videos.subtitles.preview')}>
        <Button
          size="small"
          icon={<EyeOutlined />}
          onClick={handlePreview}
          disabled={!filePath}
          loading={loading}
        />
      </Tooltip>

      <Modal
        open={open}
        title={filename || t('videos.subtitles.preview')}
        onCancel={() => setOpen(false)}
        footer={null}
        width={800}
        destroyOnHidden
      >
        <div className="max-h-[60vh] overflow-y-auto p-4 bg-slate-50 dark:bg-slate-900 rounded-md font-mono text-sm whitespace-pre-wrap text-slate-700 dark:text-slate-300">
          {content}
        </div>
      </Modal>
    </>
  );
}
