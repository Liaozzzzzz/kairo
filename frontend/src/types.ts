import { TaskStatus, RSSItemStatus } from '@/data/variables';

export interface TaskFile {
  path: string;
  size?: string;
  size_bytes?: number;
  progress?: number;
}

export interface Task {
  id: string;
  url: string;
  dir: string;
  quality: string;
  format: string;
  format_id?: string;
  parent_id?: string;
  source_type?: number;
  status: TaskStatus;
  progress: number;
  title: string;
  thumbnail: string;
  total_size?: string;
  total_bytes?: number;
  speed?: string;
  eta?: string;
  log_path?: string;
  file_exists?: boolean;
  files?: TaskFile[];
  current_item?: number;
  total_items?: number;
  created_at?: number;
}

export interface RSSFeed {
  id: string;
  url: string;
  title: string;
  description: string;
  thumbnail: string;
  last_updated: number;
  unread_count: number;
  enabled: boolean;
  custom_dir: string;
  download_latest: boolean;
  filters: string;
  tags: string;
  filename_template: string;
}

export interface RSSItem {
  id: string;
  feed_id: string;
  title: string;
  link: string;
  description: string;
  pub_date: number;
  status: RSSItemStatus;
  thumbnail: string;
}

export interface Video {
  id: string;
  task_id: string;
  title: string;
  file_path: string;
  thumbnail: string;
  duration: number;
  size: number;
  format: string;
  resolution: string;
  created_at: number;
  description: string;
  uploader: string;
  subtitles: string[];
  summary: string;
  tags: string[];
  evaluation: string;
  highlights?: {
    id: string;
    video_id: string;
    start: string;
    end: string;
    description: string;
    file_path?: string;
  }[];
  status: string;
}
