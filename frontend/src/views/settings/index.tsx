import { useTranslation } from 'react-i18next';
import { Button, Select, Card, Slider, Input, Space, Typography, Segmented } from 'antd';
import { FolderOpenOutlined } from '@ant-design/icons';
import { useShallow } from 'zustand/react/shallow';
import { ChooseDirectory } from '@root/wailsjs/go/main/App';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import { AppLanguage, useSettingStore } from '@/store/useSettingStore';

const { Text } = Typography;

const Settings = () => {
  const { t, i18n } = useTranslation();

  const {
    defaultDir,
    setDefaultDir,
    downloadConcurrency,
    setDownloadConcurrency,
    maxDownloadSpeed,
    setMaxDownloadSpeed,
    language,
    setLanguage,
    proxyUrl,
    setProxyUrl,
  } = useSettingStore(
    useShallow((state) => ({
      defaultDir: state.defaultDir,
      setDefaultDir: state.setDefaultDir,
      downloadConcurrency: state.downloadConcurrency,
      setDownloadConcurrency: state.setDownloadConcurrency,
      maxDownloadSpeed: state.maxDownloadSpeed,
      setMaxDownloadSpeed: state.setMaxDownloadSpeed,
      language: state.language,
      setLanguage: state.setLanguage,
      proxyUrl: state.proxyUrl,
      setProxyUrl: state.setProxyUrl,
    }))
  );

  const handleChooseDir = async () => {
    try {
      const dir = await ChooseDirectory();
      if (dir) {
        setDefaultDir(dir);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const maxSpeedSliderValue = maxDownloadSpeed === null ? 151 : maxDownloadSpeed;

  return (
    <PageContainer
      viewClass="px-10"
      header={
        <div className="flex items-center justify-between pb-2">
          <PageHeader title={t('settings.title')} subtitle={t('settings.subtitle')} />
        </div>
      }
    >
      <div className="space-y-4 max-w-4xl mx-auto">
        {/* Downloads Settings */}
        <Card variant="borderless" size="small" title={<span>{t('settings.tabs.downloads')}</span>}>
          <div className="space-y-5 px-2 py-0">
            {/* Directory */}
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-sm mb-0">
                  {t('settings.downloads.dir')}
                </Text>
              </div>
              <div className="md:col-span-8">
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
              </div>
            </div>

            {/* Concurrent Downloads */}
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-sm mb-0">
                  {t('settings.downloads.concurrent')}
                </Text>
              </div>
              <div className="md:col-span-8">
                <Segmented
                  block
                  value={downloadConcurrency}
                  onChange={(val) => {
                    const v = Number(val);
                    setDownloadConcurrency(v);
                  }}
                  options={[1, 2, 3, 4, 5]}
                  className="bg-gray-100 font-medium"
                />
              </div>
            </div>

            {/* Max Speed */}
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-start">
              <div className="md:col-span-4 pt-1">
                <Text strong className="block text-sm mb-0">
                  {t('settings.downloads.maxSpeed')}
                </Text>
                <div className="text-xs text-gray-500 mt-0.5">
                  {maxDownloadSpeed === null
                    ? t('settings.downloads.speedUnlimited')
                    : `${maxDownloadSpeed} MB/s`}
                </div>
              </div>
              <div className="md:col-span-8">
                <Slider
                  min={0}
                  max={151}
                  value={maxSpeedSliderValue}
                  tooltip={{ formatter: (value) => (value === 151 ? '∞' : `${value} MB/s`) }}
                  marks={{
                    0: '0',
                    50: '50',
                    100: '100',
                    151: {
                      label: '∞',
                    },
                  }}
                  onChange={(value) => {
                    if (Array.isArray(value)) return;
                    setMaxDownloadSpeed(value >= 151 ? null : value);
                  }}
                  onChangeComplete={(value) => {
                    if (Array.isArray(value)) return;
                    const v = value >= 151 ? null : value;
                    setMaxDownloadSpeed(v);
                  }}
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Network Settings */}
        <Card variant="borderless" size="small" title={<span>{t('settings.tabs.network')}</span>}>
          <div className="px-2 py-0">
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-sm mb-0">
                  {t('settings.network.proxy')}
                </Text>
              </div>
              <div className="md:col-span-8">
                <Input
                  value={proxyUrl}
                  onChange={(e) => setProxyUrl(e.target.value)}
                  placeholder={t('settings.network.proxyPlaceholder')}
                  allowClear
                />
              </div>
            </div>
          </div>
        </Card>

        {/* Language Settings */}
        <Card variant="borderless" size="small" title={<span>{t('settings.tabs.language')}</span>}>
          <div className="px-2 py-0">
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-sm mb-0">
                  {t('settings.language')}
                </Text>
              </div>
              <div className="md:col-span-8">
                <Select
                  value={language}
                  onChange={(val: AppLanguage) => {
                    i18n.changeLanguage(val);
                    setLanguage(val);
                  }}
                  style={{ width: '100%' }}
                  options={[
                    { value: 'zh', label: '中文 (Chinese)' },
                    { value: 'en', label: 'English' },
                  ]}
                />
              </div>
            </div>
          </div>
        </Card>
      </div>
    </PageContainer>
  );
};

export default Settings;
