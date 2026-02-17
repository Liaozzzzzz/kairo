import { FolderOpenOutlined } from '@ant-design/icons';
import { ChooseDirectory } from '@root/wailsjs/go/main/App';
import { Button, Input, Space } from 'antd';
import { useTranslation } from 'react-i18next';

export interface DownloadDirProps {
  defaultDir?: string;
  setNewDir?: (dir: string) => void;
  value?: string;
  onChange?: (dir: string) => void;
  className?: string;
}

const DownloadDir = ({ defaultDir, setNewDir, value, onChange, className }: DownloadDirProps) => {
  const { t } = useTranslation();

  const currentDir = value ?? defaultDir;

  const handleChooseDir = async () => {
    try {
      const d = await ChooseDirectory();
      if (d) {
        setNewDir?.(d);
        onChange?.(d);
      }
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <Space.Compact className={`w-full ${className}`}>
      <Input
        value={currentDir}
        readOnly
        className="cursor-default bg-gray-50 hover:bg-gray-50 text-gray-700 dark:bg-gray-800 dark:hover:bg-gray-800 dark:text-gray-300 dark:border-gray-700"
      />
      <Button icon={<FolderOpenOutlined />} onClick={handleChooseDir} type="default">
        {t('settings.downloads.chooseDir')}
      </Button>
    </Space.Compact>
  );
};

export default DownloadDir;
