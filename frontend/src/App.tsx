import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { ConfigProvider, Layout, Menu } from 'antd';
import { GetAppVersion, GetDefaultDownloadDir, GetTasks } from '@root/wailsjs/go/main/App';
import { EventsOn, WindowSetTitle } from '@root/wailsjs/runtime/runtime';
import { useSettingStore } from '@/store/useSettingStore';
import { useTaskStore } from '@/store/useTaskStore';
import { useAppStore } from '@/store/useAppStore';
import { Task } from '@/types';
import Tasks from '@/views/tasks';
import Downloads from '@/views/downloads';
import Settings from '@/views/settings';
import appIcon from '@/assets/images/icon-full.png';
import { MenuItemKey } from './data/variables';

const { Sider, Content } = Layout;

function App() {
  const [version, setVersion] = useState<string>('');

  const { t, i18n } = useTranslation();

  const { activeTab, menuItems, setActiveTab } = useAppStore(
    useShallow((state) => ({
      activeTab: state.activeTab,
      menuItems: state.menuItems,
      setActiveTab: state.setActiveTab,
    }))
  );

  const TABS = useMemo(() => {
    return menuItems.map((tab) => ({
      key: tab.id,
      icon: <tab.icon style={{ fontSize: 16, marginTop: '-2px' }} />,
      label: <span className="text-sm font-bold">{t(tab.labelKey)}</span>,
    }));
  }, [menuItems, t]);

  const {
    defaultDir,
    language,
    setDefaultDir,
    setDownloadConcurrency,
    setMaxDownloadSpeed,
    loadSettings,
  } = useSettingStore(
    useShallow((state) => ({
      defaultDir: state.defaultDir,
      language: state.language,
      setDefaultDir: state.setDefaultDir,
      setDownloadConcurrency: state.setDownloadConcurrency,
      setMaxDownloadSpeed: state.setMaxDownloadSpeed,
      loadSettings: state.loadSettings,
    }))
  );

  const { setTasks, updateTask, updateTaskProgress, addTaskLog } = useTaskStore(
    useShallow((state) => ({
      setTasks: state.setTasks,
      updateTask: state.updateTask,
      updateTaskProgress: state.updateTaskProgress,
      addTaskLog: state.addTaskLog,
    }))
  );

  useEffect(() => {
    // Update window title when translation changes
    WindowSetTitle(t('app.title'));
  }, [t]);

  useEffect(() => {
    loadSettings();
    if (language && language !== i18n.language) {
      i18n.changeLanguage(language);
    }

    if (!defaultDir) {
      GetDefaultDownloadDir()
        .then((d) => {
          if (d) {
            setDefaultDir(d);
          }
        })
        .catch(console.error);
    }

    GetTasks()
      .then((t) => {
        setTasks(t || {});
      })
      .catch(console.error);

    GetAppVersion()
      .then((v) => {
        setVersion(v);
      })
      .catch(console.error);

    const cleanupUpdate = EventsOn('task:update', (task: Task) => {
      // The backend sends the full task object on update
      updateTask(task.id, task);
    });

    const cleanupProgress = EventsOn(
      'task:progress',
      (data: {
        id: string;
        progress: number;
        total_size?: string;
        speed?: string;
        eta?: string;
      }) => {
        updateTaskProgress(data);
      }
    );

    const cleanupLog = EventsOn(
      'task:log',
      (data: { id: string; message: string; replace?: boolean }) => {
        addTaskLog(data.id, data.message, data.replace);
      }
    );

    const cleanupDebugNotify = EventsOn('debug:notify', (message: string) => {
      console.log('[debug:notify]', message);
    });

    return () => {
      cleanupUpdate();
      cleanupProgress();
      cleanupLog();
      cleanupDebugNotify();
    };
  }, [
    setDefaultDir,
    setTasks,
    updateTask,
    updateTaskProgress,
    addTaskLog,
    setDownloadConcurrency,
    setMaxDownloadSpeed,
    i18n,
  ]);

  return (
    <ConfigProvider
      theme={{
        token: {
          colorPrimary: '#007AFF',
          borderRadius: 8,
          fontFamily: 'Nunito, sans-serif',
        },
        components: {
          Layout: {
            bodyBg: '#ffffff',
            siderBg: undefined,
          },
          Menu: {
            itemBg: 'transparent',
            itemSelectedBg: '#ffffff',
            itemSelectedColor: '#007AFF',
            itemBorderRadius: 8,
            itemMarginInline: 16,
          },
          Segmented: {
            itemSelectedBg: '#ffffff',
            itemSelectedColor: '#000000',
            trackBg: '#7676801f', // Apple style gray with opacity
            trackPadding: 2,
            borderRadius: 8,
            controlHeightLG: 32, // Match macOS standard height
          },
        },
      }}
    >
      <Layout
        style={{
          height: '100vh',
          overflow: 'hidden',
          background:
            'linear-gradient(to right, #f1f5f9 0, #f1f5f9 200px, #eef2f7 220px, #f8fafc 400px, #ffffff 420px, #ffffff 100%)',
        }}
      >
        <Sider
          width={200}
          theme="light"
          style={{ borderRight: '1px solid #e2e8f0', background: 'transparent' }}
        >
          <div className="flex flex-col h-full">
            <div className="p-4 border-b border-gray-300 mb-2 ml-4 flex items-center gap-3">
              <img src={appIcon} alt="App Icon" className="w-8 h-8 shadow-sm" />
              <h1 className="font-extrabold text-2xl mt-0.5 select-text">{t('app.title')}</h1>
            </div>
            <Menu
              mode="inline"
              selectedKeys={[activeTab]}
              onClick={({ key }) => setActiveTab(key as MenuItemKey)}
              items={TABS}
              style={{ borderRight: 0, background: 'transparent', flex: 1 }}
            />
            <div className="border-t border-gray-300 p-3 text-center text-xs text-gray-400">
              v{version}
            </div>
          </div>
        </Sider>
        <Content
          style={{
            display: 'flex',
            flexDirection: 'column',
            overflowY: 'auto',
            minHeight: '100vh',
          }}
        >
          {activeTab === 'downloads' && <Downloads />}
          {activeTab === 'tasks' && <Tasks />}
          {activeTab === 'settings' && <Settings />}
        </Content>
      </Layout>
    </ConfigProvider>
  );
}

export default App;
