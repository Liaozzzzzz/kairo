import { create } from 'zustand';
import { DownloadOutlined, SettingOutlined, UnorderedListOutlined } from '@ant-design/icons';
import { MenuItemKey } from '@/data/variables';

interface TaskState {
  activeTab: MenuItemKey;

  menuItems: Array<{
    id: MenuItemKey;
    icon: React.FC<{ style?: React.CSSProperties }>;
    labelKey: string;
  }>;

  // Actions
  setActiveTab: (tab: MenuItemKey) => void;
}

export const useAppStore = create<TaskState>((set) => ({
  activeTab: MenuItemKey.Downloads,

  menuItems: [
    {
      id: MenuItemKey.Downloads,
      icon: DownloadOutlined,
      labelKey: 'app.sidebar.downloads',
    },
    {
      id: MenuItemKey.Tasks,
      icon: UnorderedListOutlined,
      labelKey: 'app.sidebar.tasks',
    },
    { id: MenuItemKey.Settings, icon: SettingOutlined, labelKey: 'app.sidebar.settings' },
  ],

  setActiveTab: (tab) => set({ activeTab: tab }),
}));
