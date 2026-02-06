import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Select, Card, Typography } from 'antd';

const { Title } = Typography;

const Settings = () => {
  const { t, i18n } = useTranslation();
  const [lang, setLang] = useState(i18n.language.startsWith('en') ? 'en' : 'zh');

  // Update local state if external change happens (though unlikely in this simple app)
  useEffect(() => {
    setLang(i18n.language.startsWith('en') ? 'en' : 'zh');
  }, [i18n.language]);

  const handleSave = () => {
    i18n.changeLanguage(lang);
  };

  return (
    <div className="h-full w-full bg-background text-foreground overflow-hiddenmax-w-5xl py-10 px-10">
      <Title level={2} style={{ marginBottom: 24 }}>
        {t('settings.title')}
      </Title>
      <Card variant="borderless" className="shadow-sm">
        <div className="space-y-6">
          <div className="space-y-2">
            <label className="text-sm font-medium">{t('settings.language')}</label>
            <Select
              value={lang}
              onChange={setLang}
              style={{ width: '100%' }}
              options={[
                { value: 'zh', label: '中文' },
                { value: 'en', label: 'English' },
              ]}
            />
          </div>
          <div className="flex justify-end">
            <Button type="primary" onClick={handleSave}>
              {t('settings.save')}
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
};

export default Settings;
