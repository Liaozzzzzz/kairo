import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { Modal, Input, Button, Select, Space, Image } from 'antd';
import { FolderOutlined, DownloadOutlined } from '@ant-design/icons';
import { GetVideoInfo, AddTask, ChooseDirectory } from '@root/wailsjs/go/main/App';
import { main } from '@root/wailsjs/go/models';
import { useAppStore } from '@/store/useAppStore';

interface AddTaskModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

export function AddTaskModal({ isOpen, onClose, onSuccess }: AddTaskModalProps) {
  const { t } = useTranslation();
  const defaultDir = useAppStore(useShallow((state) => state.defaultDir));

  const [newUrl, setNewUrl] = useState('');
  const [newDir, setNewDir] = useState('');
  const [newQuality, setNewQuality] = useState('best');
  const [videoInfo, setVideoInfo] = useState<main.VideoInfo | null>(null);
  const [isFetchingInfo, setIsFetchingInfo] = useState(false);

  // Initialize directory when defaultDir is loaded or modal opens
  useEffect(() => {
    if (defaultDir && !newDir) {
      setNewDir(defaultDir);
    }
  }, [defaultDir, newDir]);

  // Reset when modal opens/closes
  useEffect(() => {
    if (!isOpen) {
      // Delay reset slightly to avoid flicker during close animation if any
      const timer = setTimeout(() => {
        setNewUrl('');
        setNewQuality('best');
        setVideoInfo(null);
        if (defaultDir) setNewDir(defaultDir);
      }, 300);
      return () => clearTimeout(timer);
    } else {
      // When opening, ensure dir is set
      if (defaultDir && !newDir) setNewDir(defaultDir);
    }
  }, [isOpen, defaultDir, newDir]);

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

  const handleAddTask = async () => {
    if (!newUrl || !newDir) return;
    try {
      await AddTask(newUrl, newQuality, newDir, videoInfo?.title || '', videoInfo?.thumbnail || '');
      onSuccess?.();
      onClose();
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <Modal
      open={isOpen}
      onCancel={onClose}
      title={t('downloads.modal.title')}
      footer={[
        <Button key="cancel" onClick={onClose}>
          {t('downloads.modal.cancel')}
        </Button>,
        <Button
          key="submit"
          type="primary"
          onClick={handleAddTask}
          disabled={!newUrl || !newDir || !videoInfo}
          icon={<DownloadOutlined className="w-4 h-4" />}
        >
          {t('downloads.modal.start')}
        </Button>,
      ]}
    >
      <div className="space-y-4 pt-4">
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
            onSearch={fetchVideoInfo}
            className="flex-1"
          />
        </div>

        {videoInfo && (
          <div className="bg-slate-50 border border-slate-200 p-3 rounded-lg flex gap-3 shadow-sm">
            <div className="flex-shrink-0 w-24 h-16 rounded-md overflow-hidden border border-black/10">
              <Image
                src={videoInfo.thumbnail}
                className="w-full h-full object-cover"
                alt=""
                width="100%"
                height="100%"
              />
            </div>
            <div className="flex-1 min-w-0">
              <div className="font-medium truncate text-foreground" title={videoInfo.title}>
                {videoInfo.title}
              </div>
              <div className="text-xs text-muted-foreground mt-1 flex items-center gap-1">
                <span>{t('downloads.modal.duration')}</span>
                <span>
                  {Math.floor(videoInfo.duration / 60)}:
                  {String(Math.floor(videoInfo.duration % 60)).padStart(2, '0')}
                </span>
              </div>
            </div>
          </div>
        )}

        {videoInfo && (
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('downloads.modal.quality')}</label>
              <Select
                value={newQuality}
                onChange={setNewQuality}
                style={{ width: '100%' }}
                options={(videoInfo?.qualities?.length ? videoInfo.qualities : ['best']).map(
                  (q) => ({ label: q, value: q })
                )}
                placeholder={t('downloads.modal.bestQuality')}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">{t('downloads.modal.saveTo')}</label>
              <div className="flex gap-2">
                <Space.Compact style={{ width: '100%' }}>
                  <Input value={newDir} readOnly />
                  <Button
                    size="middle"
                    icon={<FolderOutlined className="w-4 h-4" />}
                    onClick={handleChooseDir}
                    title={t('downloads.modal.chooseDir')}
                  />
                </Space.Compact>
              </div>
            </div>
          </div>
        )}
      </div>
    </Modal>
  );
}
