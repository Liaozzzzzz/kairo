import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Select, Card } from 'antd';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';

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
    <PageContainer className="px-10" header={<PageHeader title={t('settings.title')} />}>
      <div className="pb-10">
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
    </PageContainer>
  );
};

export default Settings;
