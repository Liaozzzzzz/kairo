import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { DownloadOutlined, SettingOutlined, UnorderedListOutlined } from '@ant-design/icons';
import { ConfigProvider, Layout, Menu } from 'antd';
import { GetAppVersion, GetDefaultDownloadDir, GetTasks } from '@root/wailsjs/go/main/App';
import { EventsOn, WindowSetTitle } from '@root/wailsjs/runtime/runtime';
import { useAppStore } from '@/store/useAppStore';
import { Task } from '@/types';
import Tasks from '@/views/tasks';
import Downloads from '@/views/downloads';
import Settings from '@/views/settings';
import appIcon from '@/assets/images/icon-full.png';

const { Sider, Content } = Layout;

const TABS_CONFIG = [
  {
    id: 'downloads',
    icon: DownloadOutlined,
    labelKey: 'app.sidebar.downloads',
  },
  {
    id: 'tasks',
    icon: UnorderedListOutlined,
    labelKey: 'app.sidebar.tasks',
  },
  { id: 'settings', icon: SettingOutlined, labelKey: 'app.sidebar.settings' },
] as const;

function App() {
  const { t } = useTranslation();

  const TABS = TABS_CONFIG.map((tab) => ({
    key: tab.id,
    icon: <tab.icon style={{ fontSize: 20, marginTop: '-2px' }} />,
    label: t(tab.labelKey),
  }));

  const [activeTab, setActiveTab] = useState<(typeof TABS_CONFIG)[number]['id']>('downloads');
  const [version, setVersion] = useState<string>('');

  const { setDefaultDir, setTasks, updateTask, updateTaskProgress, addTaskLog } = useAppStore(
    useShallow((state) => ({
      setDefaultDir: state.setDefaultDir,
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
    // Set default dir
    GetDefaultDownloadDir()
      .then((d) => {
        setDefaultDir(d);
      })
      .catch(console.error);

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
  }, [setDefaultDir, setTasks, updateTask, updateTaskProgress, addTaskLog]);

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
            siderBg: '#f1f5f9', // slate-100
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
      <Layout style={{ height: '100vh', overflow: 'hidden' }}>
        <Sider width={200} theme="light" style={{ borderRight: '1px solid #e2e8f0' }}>
          <div className="flex flex-col h-full">
            <div className="p-4 border-b border-gray-200 mb-2 flex flex-col items-center justify-center">
              <img src={appIcon} alt="App Icon" className="w-16 h-16 mb-2 rounded-xl shadow-sm" />
              <h1 className="font-bold text-xl">{t('app.title')}</h1>
            </div>
            <Menu
              mode="inline"
              selectedKeys={[activeTab]}
              onClick={({ key }) => setActiveTab(key as (typeof TABS_CONFIG)[number]['id'])}
              items={TABS}
              style={{ borderRight: 0, background: 'transparent', flex: 1 }}
            />
            <div className="p-4 text-center text-xs text-gray-400">v{version}</div>
          </div>
        </Sider>
        <Content
          style={{
            display: 'flex',
            flexDirection: 'column',
            overflow: 'hidden',
            background: '#fff',
          }}
        >
          {activeTab === 'downloads' && <Downloads onAdded={() => setActiveTab('tasks')} />}
          {activeTab === 'tasks' && <Tasks />}
          {activeTab === 'settings' && <Settings />}
        </Content>
      </Layout>
    </ConfigProvider>
  );
}

export default App;
