export interface TaskStage {
  name: string;
  status: string;
  progress: number;
  total_size?: string;
}

export interface Task {
  id: string;
  url: string;
  dir: string;
  quality: string;
  format: string;
  status: string;
  progress: number;
  title: string;
  thumbnail: string;
  stages?: TaskStage[];
  total_size?: string;
  speed?: string;
  eta?: string;
  log_path?: string;
}
