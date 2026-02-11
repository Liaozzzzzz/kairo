import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Button,
  Select,
  Card,
  Slider,
  Input,
  Space,
  Typography,
  Segmented,
  Switch,
  Radio,
} from 'antd';
import { FolderOpenOutlined, FileTextOutlined } from '@ant-design/icons';
import { useShallow } from 'zustand/react/shallow';
import { ChooseDirectory, ChooseFile, GetPlatform } from '@root/wailsjs/go/main/App';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import { AppLanguage, useSettingStore, CookieConfig } from '@/store/useSettingStore';

const { Text } = Typography;

const Settings = () => {
  const { t, i18n } = useTranslation();
  const [platform, setPlatform] = useState('');

  useEffect(() => {
    GetPlatform().then(setPlatform);
  }, []);

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
    cookie,
    setCookie,
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
      cookie: state.cookie,
      setCookie: state.setCookie,
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

  const handleChooseCookiesFile = async (currentConfig: CookieConfig) => {
    try {
      const file = await ChooseFile();
      if (file) {
        const update = { ...currentConfig, file };
        setCookie(update);
      }
    } catch (e) {
      console.error(e);
    }
  };

  const maxSpeedSliderValue = maxDownloadSpeed === null ? 151 : maxDownloadSpeed;

  const renderCookieSettings = (config: CookieConfig, setConfig: (val: CookieConfig) => void) => {
    return (
      <>
        {/* Enable Switch */}
        <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
          <div className="md:col-span-4">
            <Text strong className="block text-[13px] text-gray-600 mb-0">
              {t('settings.site.enableAuth')}
            </Text>
          </div>
          <div className="md:col-span-8">
            <Switch
              checked={config.enabled}
              onChange={(checked) => setConfig({ ...config, enabled: checked })}
            />
          </div>
        </div>

        {config.enabled && (
          <>
            {/* Auth Mode */}
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-[13px] text-gray-600 mb-0">
                  {t('settings.site.authMode')}
                </Text>
              </div>
              <div className="md:col-span-8">
                <Radio.Group
                  value={config.source}
                  onChange={(e) => setConfig({ ...config, source: e.target.value })}
                >
                  <Radio value="browser">{t('settings.site.browser')}</Radio>
                  <Radio value="file">{t('settings.site.file')}</Radio>
                </Radio.Group>
              </div>
            </div>

            {/* Browser Select */}
            {config.source === 'browser' && (
              <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                <div className="md:col-span-4">
                  <Text strong className="block text-[13px] text-gray-600 mb-0">
                    {t('settings.network.cookies')}
                  </Text>
                </div>
                <div className="md:col-span-8">
                  <Select
                    value={config.browser || undefined}
                    onChange={(val) => setConfig({ ...config, browser: val })}
                    style={{ width: '100%' }}
                    placeholder={t('settings.network.cookiesPlaceholder')}
                    allowClear
                    options={[
                      { value: 'chrome', label: 'Google Chrome' },
                      { value: 'firefox', label: 'Mozilla Firefox' },
                      { value: 'edge', label: 'Microsoft Edge' },
                      { value: 'safari', label: 'Safari' },
                      { value: 'opera', label: 'Opera' },
                      { value: 'brave', label: 'Brave' },
                      { value: 'vivaldi', label: 'Vivaldi' },
                      { value: 'chromium', label: 'Chromium' },
                    ].filter((option) => platform !== 'windows' || option.value === 'firefox')}
                  />
                </div>
              </div>
            )}

            {/* File Select */}
            {config.source === 'file' && (
              <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
                <div className="md:col-span-4">
                  <Text strong className="block text-[13px] text-gray-600 mb-0">
                    {t('settings.network.cookiesFile')}
                  </Text>
                </div>
                <div className="md:col-span-8">
                  <Space.Compact style={{ width: '100%' }}>
                    <Input
                      value={config.file}
                      readOnly
                      placeholder={t('settings.network.cookiesFilePlaceholder')}
                      className="cursor-default bg-gray-50 hover:bg-gray-50 text-gray-700"
                      allowClear
                      onChange={(e) => {
                        if (!e.target.value) setConfig({ ...config, file: '' });
                      }}
                    />
                    <Button
                      icon={<FileTextOutlined />}
                      onClick={() => handleChooseCookiesFile(config)}
                      type="default"
                    >
                      {t('settings.network.chooseFile')}
                    </Button>
                  </Space.Compact>
                </div>
              </div>
            )}
          </>
        )}
      </>
    );
  };

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
        <Card
          variant="borderless"
          size="small"
          title={
            <span className="text-base font-semibold text-gray-800">
              {t('settings.tabs.downloads')}
            </span>
          }
        >
          <div className="space-y-5 px-2 py-0">
            {/* Directory */}
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-[13px] text-gray-600 mb-0">
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
                <Text strong className="block text-[13px] text-gray-600 mb-0">
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
                <Text strong className="block text-[13px] text-gray-600 mb-0">
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
        <Card
          variant="borderless"
          size="small"
          title={
            <span className="text-base font-semibold text-gray-800">
              {t('settings.tabs.network')}
            </span>
          }
        >
          <div className="px-2 py-0 space-y-5">
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-[13px] text-gray-600 mb-0">
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
            {renderCookieSettings(cookie, setCookie)}
          </div>
        </Card>

        {/* Language Settings */}
        <Card
          variant="borderless"
          size="small"
          title={
            <span className="text-base font-semibold text-gray-800">
              {t('settings.tabs.language')}
            </span>
          }
        >
          <div className="px-2 py-0">
            <div className="grid grid-cols-1 md:grid-cols-12 gap-2 items-center">
              <div className="md:col-span-4">
                <Text strong className="block text-[13px] text-gray-600 mb-0">
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
