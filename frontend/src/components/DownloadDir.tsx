import { FolderOpenOutlined } from '@ant-design/icons';
import { ChooseDirectory } from '@root/wailsjs/go/main/App';
import { Button, Input, Space } from 'antd';
import { useTranslation } from 'react-i18next';

export interface DownloadDirProps {
  defaultDir: string;
  setNewDir: (dir: string) => void;
}

const DownloadDir = ({ defaultDir, setNewDir }: DownloadDirProps) => {
  const { t } = useTranslation();

  const handleChooseDir = async () => {
    try {
      const d = await ChooseDirectory();
      if (d) setNewDir(d);
    } catch (e) {
      console.error(e);
    }
  };

  return (
    <Space.Compact style={{ width: '100%' }}>
      <Input
        value={defaultDir}
        readOnly
        className="cursor-default bg-gray-50 hover:bg-gray-50 text-gray-700"
      />
      <Button icon={<FolderOpenOutlined />} onClick={handleChooseDir} type="default">
        {t('settings.downloads.chooseDir')}
      </Button>
    </Space.Compact>
  );
};

export default DownloadDir;
