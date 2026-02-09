import { TaskStatus } from '@/data/variables';

export interface Task {
  id: string;
  url: string;
  dir: string;
  quality: string;
  format: string;
  format_id?: string;
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
}
