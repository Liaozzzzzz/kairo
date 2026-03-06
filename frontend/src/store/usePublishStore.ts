import { create } from 'zustand';
import { schema } from '@root/wailsjs/go/models';
import {
  ListPublishPlatforms,
  ListPublishAccounts,
  ListPublishTasks,
} from '@root/wailsjs/go/main/App';

interface PublishState {
  platforms: schema.PublishPlatform[];
  accounts: schema.PublishAccount[];
  tasks: schema.PublishTask[];
  total: number;
  page: number;
  pageSize: number;
  loadingPlatforms: boolean;
  loadingAccounts: boolean;
  loadingTasks: boolean;
  fetchPlatforms: () => Promise<void>;
  fetchAccounts: (platformID?: string) => Promise<void>;
  fetchTasks: (
    status?: string,
    platformID?: string,
    page?: number,
    pageSize?: number
  ) => Promise<void>;
  setAccounts: (accounts: schema.PublishAccount[]) => void;
  setTasks: (tasks: schema.PublishTask[]) => void;
}

export const usePublishStore = create<PublishState>((set) => ({
  platforms: [],
  accounts: [],
  tasks: [],
  total: 0,
  page: 1,
  pageSize: 10,
  loadingPlatforms: false,
  loadingAccounts: false,
  loadingTasks: false,

  fetchPlatforms: async () => {
    set({ loadingPlatforms: true });
    try {
      const platforms = await ListPublishPlatforms();
      set({ platforms: platforms || [] });
    } catch (error) {
      console.error('Failed to fetch platforms:', error);
    } finally {
      set({ loadingPlatforms: false });
    }
  },

  fetchAccounts: async (platformID = 'all') => {
    set({ loadingAccounts: true });
    try {
      const accounts = await ListPublishAccounts(platformID);
      set({ accounts: accounts || [] });
    } catch (error) {
      console.error('Failed to fetch accounts:', error);
    } finally {
      set({ loadingAccounts: false });
    }
  },

  fetchTasks: async (status = 'all', platformID = 'all', page = 1, pageSize = 10) => {
    set({ loadingTasks: true, page, pageSize });
    try {
      const res = await ListPublishTasks(status, platformID, page, pageSize);

      if (res && Array.isArray(res.data)) {
        set({ tasks: res.data || [], total: res.total || 0 });
      } else if (Array.isArray(res)) {
        // Fallback for old API if something goes wrong or partial update
        set({ tasks: (res as schema.PublishTask[]) || [], total: res.length || 0 });
      } else {
        set({ tasks: [], total: 0 });
      }
    } catch (error) {
      console.error('Failed to fetch publish tasks:', error);
    } finally {
      set({ loadingTasks: false });
    }
  },

  setAccounts: (accounts) => set({ accounts }),
  setTasks: (tasks) => set({ tasks }),
}));
