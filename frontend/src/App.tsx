import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useShallow } from 'zustand/react/shallow';
import { ConfigProvider, Layout, Menu, notification } from 'antd';
import { GetAppVersion, GetTasks, GetPlatform } from '@root/wailsjs/go/main/App';
import {
  EventsOn,
  WindowSetTitle,
  WindowToggleMaximise,
  WindowMinimise,
  Quit,
} from '@root/wailsjs/runtime/runtime';
import { useSettingStore } from '@/store/useSettingStore';
import { useTaskStore } from '@/store/useTaskStore';
import { useRSSStore } from '@/store/useRSSStore';
import { useAppStore } from '@/store/useAppStore';
import { useTheme } from '@/hooks/useTheme';
import { generateDarkPalette, generateLightPalette } from '@/lib/theme';
import ThemeSync from '@/components/ThemeSync';
import { Task } from '@/types';
import Tasks from '@/views/tasks';
import Downloads from '@/views/downloads';
import Settings from '@/views/settings';
import RSSView from '@/views/rss';
import appIcon from '@/assets/images/icon-full.png';
import { MenuItemKey, TaskStatus, SourceType } from './data/variables';

import { getThemeColor } from '@/data/themeColors';

const { Sider, Content } = Layout;

function App() {
  const [version, setVersion] = useState<string>('');
  const [platform, setPlatform] = useState<string>('');
  const [api, contextHolder] = notification.useNotification();
  const { antAlgorithm, isDark } = useTheme();

  const {
    language,
    themeColor,
    setDefaultDir,
    setDownloadConcurrency,
    setMaxDownloadSpeed,
    loadSettings,
  } = useSettingStore(
    useShallow((state) => ({
      defaultDir: state.defaultDir,
      language: state.language,
      themeColor: state.themeColor,
      setDefaultDir: state.setDefaultDir,
      setDownloadConcurrency: state.setDownloadConcurrency,
      setMaxDownloadSpeed: state.setMaxDownloadSpeed,
      loadSettings: state.loadSettings,
    }))
  );

  const brandColor = useMemo(() => getThemeColor(themeColor), [themeColor]);

  const darkPalette = useMemo(() => {
    return generateDarkPalette(brandColor);
  }, [brandColor]);

  const lightPalette = useMemo(() => {
    return generateLightPalette(brandColor);
  }, [brandColor]);

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

    GetPlatform().then(setPlatform).catch(console.error);

    const cleanupUpdate = EventsOn('task:update', (task: Task) => {
      // Check for status change
      const currentTasks = useTaskStore.getState().tasks;
      const prevTask = currentTasks[task.id];

      // The backend sends the full task object on update
      updateTask(task.id, task);

      if (task.status === TaskStatus.Completed && task.source_type === SourceType.RSS) {
        useRSSStore.getState().setRSSItemDownloaded(task.url);
      }

      // Skip notification for playlist parent tasks
      if (
        task.source_type === SourceType.Playlist ||
        task.source_type === SourceType.RSS ||
        (prevTask && prevTask.status === task.status)
      ) {
        return;
      }

      if (task.status === TaskStatus.Completed) {
        api.success({
          message: t('tasks.notification.completed'),
          description: task.title || task.url,
          placement: 'topRight',
        });
      } else if (task.status === TaskStatus.Error) {
        api.error({
          message: t('tasks.notification.failed'),
          description: task.title || task.url,
          placement: 'topRight',
        });
      }
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
    t,
  ]);

  useEffect(() => {
    if (language && language !== i18n.language) {
      i18n.changeLanguage(language);
    }
  }, [language, i18n]);

  const currentPalette = isDark ? darkPalette : lightPalette;

  return (
    <ConfigProvider
      theme={{
        algorithm: antAlgorithm,
        token: {
          colorPrimary: brandColor,
          borderRadius: 8,
          fontFamily: 'Nunito, sans-serif',
          colorBgContainer: isDark ? darkPalette.containerBg : lightPalette.containerBg,
          colorBgElevated: isDark ? darkPalette.elevatedBg : lightPalette.elevatedBg,
          colorText: isDark ? darkPalette.text : lightPalette.text,
          colorBorder: isDark ? darkPalette.border : lightPalette.border,
        },
        components: {
          Layout: {
            bodyBg: isDark ? darkPalette.bodyBg : lightPalette.bodyBg,
            siderBg: isDark ? darkPalette.siderBg : lightPalette.siderBg,
          },
          Menu: {
            itemBg: 'transparent',
            itemSelectedBg: isDark
              ? darkPalette.itemSelectedBg + '1a'
              : lightPalette.itemSelectedBg, // Light mode uses solid color from palette
            itemSelectedColor: brandColor,
            itemColor: isDark ? darkPalette.textSecondary : lightPalette.text,
            itemBorderRadius: 8,
            itemMarginInline: 16,
          },
          Segmented: {
            itemSelectedBg: isDark ? darkPalette.elevatedBg : lightPalette.elevatedBg,
            itemSelectedColor: isDark ? darkPalette.text : lightPalette.text,
            trackBg: isDark ? darkPalette.containerBg : 'rgba(0, 0, 0, 0.04)',
            trackPadding: 2,
            borderRadius: 8,
            controlHeightLG: 32,
          },
          Input: {
            colorBgContainer: isDark ? darkPalette.containerBg : lightPalette.containerBg,
            colorBorder: isDark ? darkPalette.border : lightPalette.border,
            colorText: isDark ? darkPalette.text : lightPalette.text,
            colorTextPlaceholder: isDark ? darkPalette.textSecondary : 'rgba(0, 0, 0, 0.25)',
          },
          Card: {
            colorBgContainer: isDark ? darkPalette.containerBg : lightPalette.containerBg,
            colorBorderSecondary: isDark ? darkPalette.border : '#f0f0f0',
          },
        },
      }}
    >
      {contextHolder}
      <ThemeSync palette={currentPalette} />
      <Layout className="h-screen overflow-hidden bg-background">
        <Sider width={200} theme={isDark ? 'dark' : 'light'} className="border-r border-border">
          <div className="flex flex-col h-full">
            <div
              style={{ height: 40, '--wails-draggable': 'drag' } as React.CSSProperties}
              className="flex-shrink-0"
              onDoubleClick={WindowToggleMaximise}
            />
            <div className="p-4 border-b border-border mb-2 ml-4 flex items-center gap-3">
              <img src={appIcon} alt="App Icon" className="w-8 h-8 shadow-sm" />
              <h1 className="font-extrabold text-2xl mt-0.5 select-text text-foreground">
                {t('app.title')}
              </h1>
            </div>
            <Menu
              mode="inline"
              selectedKeys={[activeTab]}
              onClick={({ key }) => setActiveTab(key as MenuItemKey)}
              items={TABS}
              className="border-r-0 bg-transparent flex-1"
            />
            <div className="border-t border-border p-3 text-center text-xs text-muted-foreground">
              v{version}
            </div>
          </div>
        </Sider>
        <div className="flex-1 flex flex-col relative min-w-0">
          <div
            style={
              {
                position: 'absolute',
                top: 0,
                left: 0,
                right: 0,
                height: 40,
                zIndex: 9999,
                '--wails-draggable': 'drag',
              } as React.CSSProperties
            }
            onDoubleClick={WindowToggleMaximise}
          >
            {platform === 'windows' && (
              <div className="absolute top-0 right-0 h-full flex items-center">
                <div
                  className="h-full w-12 flex items-center justify-center hover:bg-gray-200 cursor-default transition-colors text-gray-600"
                  onClick={WindowMinimise}
                  style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                >
                  <svg width="10" height="1" viewBox="0 0 10 1">
                    <rect width="10" height="1" fill="currentColor" />
                  </svg>
                </div>
                <div
                  className="h-full w-12 flex items-center justify-center hover:bg-gray-200 cursor-default transition-colors text-gray-600"
                  onClick={WindowToggleMaximise}
                  style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                >
                  <svg width="10" height="10" viewBox="0 0 10 10">
                    <path
                      d="M1,1 L9,1 L9,9 L1,9 L1,1 M0,0 L0,10 L10,10 L10,0 L0,0"
                      fill="currentColor"
                    />
                  </svg>
                </div>
                <div
                  className="h-full w-12 flex items-center justify-center hover:bg-[#E81123] hover:text-white cursor-default transition-colors text-gray-600"
                  onClick={Quit}
                  style={{ '--wails-draggable': 'no-drag' } as React.CSSProperties}
                >
                  <svg width="10" height="10" viewBox="0 0 10 10">
                    <path
                      d="M1.0,0.0 L5.0,4.0 L9.0,0.0 L10.0,1.0 L6.0,5.0 L10.0,9.0 L9.0,10.0 L5.0,6.0 L1.0,10.0 L0.0,9.0 L4.0,5.0 L0.0,1.0 L1.0,0.0"
                      fill="currentColor"
                    />
                  </svg>
                </div>
              </div>
            )}
          </div>
          <Content
            style={{
              display: 'flex',
              flexDirection: 'column',
              overflowY: 'auto',
              height: '100vh',
            }}
          >
            {activeTab === 'downloads' && <Downloads />}
            {activeTab === 'tasks' && <Tasks />}
            {activeTab === 'rss' && <RSSView />}
            {activeTab === 'settings' && <Settings />}
          </Content>
        </div>
      </Layout>
    </ConfigProvider>
  );
}

export default App;
