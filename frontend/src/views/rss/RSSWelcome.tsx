import React from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card } from 'antd';
import { BrowserOpenURL } from '@root/wailsjs/runtime/runtime';
import { ReadOutlined, BookOutlined } from '@ant-design/icons';

const RSSWelcome: React.FC = () => {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col items-center justify-center h-full min-h-[500px] p-8">
      <div className="text-center max-w-lg">
        <div className="mb-8 flex justify-center">
          <div className="w-24 h-24 bg-primary/10 rounded-full flex items-center justify-center">
            <ReadOutlined className="text-4xl text-primary" />
          </div>
        </div>

        <h2 className="text-2xl font-bold text-slate-800 dark:text-slate-100 mb-3">
          {t('rss.welcome.title')}
        </h2>

        <p className="text-slate-500 dark:text-slate-400 mb-10 text-base leading-relaxed">
          {t('rss.welcome.subtitle')}
        </p>

        <Card
          className="border-amber-100 bg-amber-50 dark:border-amber-900/30 dark:bg-amber-900/10 shadow-sm"
          styles={{ body: { padding: '24px' } }}
        >
          <div className="flex flex-col gap-4">
            <div className="flex items-start gap-4">
              <div className="p-2.5 bg-amber-100 dark:bg-amber-900/30 rounded-lg shrink-0 mt-1">
                <BookOutlined className="text-xl text-amber-600 dark:text-amber-400" />
              </div>
              <div className="text-left">
                <h3 className="font-semibold text-slate-800 dark:text-slate-200 mb-1.5">
                  {t('rss.rsshub.title')}
                </h3>
                <p className="text-slate-600 dark:text-slate-400 text-sm leading-relaxed">
                  {t('rss.rsshub.desc')}
                </p>
              </div>
            </div>

            <div className="flex justify-end pt-2">
              <Button
                type="default"
                ghost
                className="!border-amber-200 !text-amber-700 hover:!text-amber-800 hover:!border-amber-300 dark:!border-amber-800 dark:!text-amber-400 dark:hover:!text-amber-300 dark:hover:!border-amber-700"
                onClick={() => BrowserOpenURL('https://docs.rsshub.app/')}
              >
                {t('rss.rsshub.openDocs')}
              </Button>
            </div>
          </div>
        </Card>
      </div>
    </div>
  );
};

export default RSSWelcome;
