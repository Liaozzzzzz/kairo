import { useRef, useState } from 'react';
import { Button, Card, Tabs } from 'antd';
import { useTranslation } from 'react-i18next';
import PageContainer from '@/components/PageContainer';
import PageHeader from '@/components/PageHeader';
import AccountsPanel, { AccountsPanelRef } from './AccountsPanel';
import AutomationPanel, { AutomationPanelRef } from './AutomationPanel';
import PublishCenterPanel, { PublishCenterPanelRef } from './PublishCenterPanel';
import PlatformsPanel from './PlatformsPanel';

const PublishView = () => {
  const { t } = useTranslation();
  const [activeKey, setActiveKey] = useState('accounts');
  const accountsPanelRef = useRef<AccountsPanelRef>(null);
  const automationPanelRef = useRef<AutomationPanelRef>(null);
  const publishCenterPanelRef = useRef<PublishCenterPanelRef>(null);

  return (
    <PageContainer
      viewClass="px-10 py-4"
      header={<PageHeader title={t('publish.title')} subtitle={t('publish.subtitle')} />}
    >
      <Card size="small">
        <Tabs
          activeKey={activeKey}
          onChange={setActiveKey}
          tabBarExtraContent={
            <>
              {activeKey === 'accounts' && (
                <Button
                  type="primary"
                  size="middle"
                  onClick={() => {
                    accountsPanelRef.current?.addAccount?.();
                  }}
                >
                  {t('publish.addAccount')}
                </Button>
              )}
              {activeKey === 'automation' && (
                <Button
                  type="primary"
                  size="middle"
                  onClick={() => {
                    automationPanelRef.current?.addAutomation?.();
                  }}
                >
                  {t('publish.addAutomation')}
                </Button>
              )}
              {activeKey === 'center' && (
                <Button
                  type="primary"
                  size="middle"
                  onClick={() => {
                    publishCenterPanelRef.current?.addTask?.();
                  }}
                >
                  {t('publish.addTask')}
                </Button>
              )}
            </>
          }
          items={[
            {
              key: 'accounts',
              label: t('publish.tabs.accounts'),
              children: <AccountsPanel ref={accountsPanelRef} />,
            },
            {
              key: 'platforms',
              label: t('publish.tabs.platforms'),
              children: <PlatformsPanel />,
            },
            {
              key: 'automation',
              label: t('publish.tabs.automation'),
              children: <AutomationPanel ref={automationPanelRef} />,
            },
            {
              key: 'center',
              label: t('publish.tabs.center'),
              children: <PublishCenterPanel ref={publishCenterPanelRef} />,
            },
          ]}
        />
      </Card>
    </PageContainer>
  );
};

export default PublishView;
