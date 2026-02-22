import { create } from 'zustand';
import { Video } from '@/types';
import { GetVideos } from '@root/wailsjs/go/main/App';

interface VideoState {
  videos: Video[];
  loading: boolean;
  setVideos: (videos: Video[]) => void;
  fetchVideos: (status?: string, query?: string) => Promise<void>;
  updateVideoStatus: (
    id: string,
    status: string,
    summary?: string,
    evaluation?: string,
    tags?: string[]
  ) => void;
  removeVideo: (id: string) => void;
}

export const useVideoStore = create<VideoState>((set) => ({
  videos: [],
  loading: false,

  setVideos: (videos) => set({ videos }),

  fetchVideos: async (status = 'all', query = '') => {
    set({ loading: true });
    try {
      const result = await GetVideos({ status, query });
      set({ videos: result || [] });
    } catch (error) {
      console.error('Failed to fetch videos:', error);
    } finally {
      set({ loading: false });
    }
  },

  updateVideoStatus: (id, status, summary, evaluation, tags) =>
    set((state) => ({
      videos: state.videos.map((v) =>
        v.id === id
          ? {
              ...v,
              status,
              summary: summary || v.summary,
              evaluation: evaluation || v.evaluation,
              tags: tags || v.tags,
            }
          : v
      ),
    })),

  removeVideo: (id) =>
    set((state) => ({
      videos: state.videos.filter((v) => v.id !== id),
    })),
}));
