import { TaskStatus } from '@/data/variables';

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
  playlist_items?: number[];
  parent_id?: string;
  is_playlist?: boolean;
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
