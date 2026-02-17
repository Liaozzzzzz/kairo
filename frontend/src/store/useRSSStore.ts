import { create } from 'zustand';
import { RSSFeed, RSSItem, RSSItemStatus } from '@/types';

import {
  AddRSSFeed,
  GetRSSFeeds,
  DeleteRSSFeed,
  RefreshRSSFeed,
  MarkRSSItemRead,
  GetRSSFeedItems,
  SetRSSFeedEnabled,
  UpdateRSSFeed,
  SetRSSItemQueued,
} from '@root/wailsjs/go/main/App';

interface AddFeedInput {
  url: string;
  custom_dir: string;
  download_latest: boolean;
  filters: string;
  tags: string;
  filename_template: string;
}

interface UpdateFeedInput {
  id: string;
  custom_dir: string;
  download_latest: boolean;
  filters: string;
  tags: string;
  filename_template: string;
}

interface RSSState {
  feeds: RSSFeed[];
  selectedFeedId: string | null;
  feedItems: RSSItem[];
  isLoading: boolean;
  isItemsLoading: boolean;

  fetchFeeds: () => Promise<void>;
  selectFeed: (id: string | null) => Promise<void>;
  addFeed: (input: AddFeedInput) => Promise<void>;
  updateFeed: (input: UpdateFeedInput) => Promise<void>;
  deleteFeed: (id: string) => Promise<void>;
  refreshFeed: (id: string) => Promise<void>;
  markItemRead: (itemId: string) => Promise<void>;
  setRSSItemQueued: (itemId: string, queued: boolean) => Promise<void>;
  toggleFeedEnabled: (feedId: string, enabled: boolean) => Promise<void>;
}

export const useRSSStore = create<RSSState>((set, get) => ({
  feeds: [],
  selectedFeedId: null,
  feedItems: [],
  isLoading: false,
  isItemsLoading: false,

  fetchFeeds: async () => {
    set({ isLoading: true });
    try {
      const feeds = await GetRSSFeeds();
      const values = { feeds: feeds || [] };
      if (!get().selectedFeedId) {
        get().selectFeed(values.feeds[0]?.id || null);
      }
      set(values);
    } catch (error) {
      console.error('Failed to fetch feeds:', error);
    } finally {
      set({ isLoading: false });
    }
  },

  selectFeed: async (id: string | null) => {
    set({ selectedFeedId: id, feedItems: [] });
    if (id) {
      set({ isItemsLoading: true });
      try {
        const items = await GetRSSFeedItems(id);
        set({ feedItems: items || [] });
      } catch (error) {
        console.error('Failed to fetch feed items:', error);
      } finally {
        set({ isItemsLoading: false });
      }
    }
  },

  addFeed: async (input: AddFeedInput) => {
    set({ isLoading: true });
    try {
      const feed = await AddRSSFeed(input);
      set((state) => ({ feeds: [...state.feeds, feed] }));
    } finally {
      set({ isLoading: false });
    }
  },

  updateFeed: async (input: UpdateFeedInput) => {
    try {
      // Find existing feed to get other properties
      const existingFeed = get().feeds.find((f) => f.id === input.id);
      if (!existingFeed) return;

      const updatedFeed: RSSFeed = {
        ...existingFeed,
        custom_dir: input.custom_dir,
        download_latest: input.download_latest,
        filters: input.filters,
        tags: input.tags,
        filename_template: input.filename_template,
      };

      await UpdateRSSFeed(updatedFeed);
      set((state) => ({
        feeds: state.feeds.map((f) => (f.id === input.id ? updatedFeed : f)),
      }));
    } catch (error) {
      console.error('Failed to update feed:', error);
      throw error;
    }
  },

  deleteFeed: async (id: string) => {
    try {
      await DeleteRSSFeed(id);
      set((state) => ({
        feeds: state.feeds.filter((f) => f.id !== id),
        selectedFeedId: state.selectedFeedId === id ? null : state.selectedFeedId,
        feedItems: state.selectedFeedId === id ? [] : state.feedItems,
      }));
    } catch (error) {
      console.error('Failed to delete feed:', error);
    }
  },

  refreshFeed: async (id: string) => {
    try {
      await RefreshRSSFeed(id);
      await get().fetchFeeds();
      if (get().selectedFeedId === id) {
        const items = await GetRSSFeedItems(id);
        set({ feedItems: items || [] });
      }
    } catch (error) {
      console.error('Failed to refresh feed:', error);
    }
  },

  markItemRead: async (itemId: string) => {
    // Optimistic update
    set((state) => ({
      feedItems: state.feedItems.map((item) =>
        item.id === itemId && item.status === RSSItemStatus.New
          ? { ...item, status: RSSItemStatus.Read }
          : item
      ),
      // Update unread count in feeds list locally to avoid full refetch
      feeds: state.feeds.map((feed) => {
        if (feed.id === state.selectedFeedId) {
          // Check if item was actually unread before decrementing
          const item = state.feedItems.find((i) => i.id === itemId);
          if (item && item.status === RSSItemStatus.New) {
            return { ...feed, unread_count: Math.max(0, feed.unread_count - 1) };
          }
        }
        return feed;
      }),
    }));

    try {
      await MarkRSSItemRead(itemId);
    } catch (error) {
      console.error('Failed to mark item read:', error);
      // Revert optimistic update if failed (optional, but good practice)
      // For read status, it's often okay to just leave it as read or fetch fresh state later
    }
  },

  setRSSItemQueued: async (itemId: string, queued: boolean) => {
    // Optimistic update
    const newStatus = queued ? RSSItemStatus.Queued : RSSItemStatus.Read; // 2: Queued, 1: Read (fallback)
    set((state) => ({
      feedItems: state.feedItems.map((item) =>
        item.id === itemId ? { ...item, status: newStatus } : item
      ),
    }));

    try {
      await SetRSSItemQueued(itemId, queued);
    } catch (error) {
      console.error('Failed to set item queued:', error);
      // Revert optimistic update
      // We don't know the previous status easily without storing it, but we can refetch
      const items = await GetRSSFeedItems(get().selectedFeedId!);
      set({ feedItems: items || [] });
    }
  },

  toggleFeedEnabled: async (feedId: string, enabled: boolean) => {
    try {
      await SetRSSFeedEnabled(feedId, enabled);
      set((state) => ({
        feeds: state.feeds.map((feed) => (feed.id === feedId ? { ...feed, enabled } : feed)),
      }));
    } catch (error) {
      console.error('Failed to toggle feed enabled:', error);
    }
  },
}));
