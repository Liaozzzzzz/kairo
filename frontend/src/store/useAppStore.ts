import { create } from 'zustand';
import { Task, TaskStage } from '../types';

interface AppState {
  tasks: Record<string, Task>;
  taskLogs: Record<string, string[]>;
  defaultDir: string;

  // Actions
  setTasks: (tasks: Record<string, Task>) => void;
  updateTask: (taskId: string, updates: Partial<Task>) => void;
  updateTaskProgress: (data: {
    id: string;
    progress: number;
    total_size?: string;
    speed?: string;
    eta?: string;
    stages?: TaskStage[];
  }) => void;
  addTaskLog: (taskId: string, message: string, replace?: boolean) => void;
  setTaskLogs: (taskId: string, logs: string[]) => void;
  setDefaultDir: (dir: string) => void;
  deleteTask: (taskId: string) => void;
}

export const useAppStore = create<AppState>((set) => ({
  tasks: {},
  taskLogs: {},
  defaultDir: '',

  setTasks: (tasks) => set({ tasks }),

  updateTask: (taskId, updates) =>
    set((state) => {
      const task = state.tasks[taskId];
      if (!task) {
        // If task doesn't exist, treat updates as a new task
        // We assume updates contains the full task data when it's a new task
        return {
          tasks: {
            ...state.tasks,
            [taskId]: updates as Task,
          },
        };
      }
      return {
        tasks: {
          ...state.tasks,
          [taskId]: { ...task, ...updates },
        },
      };
    }),

  updateTaskProgress: (data) =>
    set((state) => {
      const task = state.tasks[data.id];
      if (!task) return state;
      return {
        tasks: {
          ...state.tasks,
          [data.id]: {
            ...task,
            progress: data.progress,
            total_size: data.total_size,
            speed: data.speed,
            eta: data.eta,
            stages: data.stages || task.stages,
          },
        },
      };
    }),

  addTaskLog: (taskId, message, replace) =>
    set((state) => {
      const currentLogs = state.taskLogs[taskId] || [];
      const newLogs = [...currentLogs];

      if (replace && newLogs.length > 0) {
        const lastLog = newLogs[newLogs.length - 1];
        // If the last log was also a progress update, replace it
        if (lastLog.startsWith('[download]') && lastLog.includes('%')) {
          newLogs[newLogs.length - 1] = message;
        } else {
          newLogs.push(message);
        }
      } else {
        newLogs.push(message);
      }

      return {
        taskLogs: {
          ...state.taskLogs,
          [taskId]: newLogs.slice(-500),
        },
      };
    }),

  setTaskLogs: (taskId, logs) =>
    set((state) => ({
      taskLogs: {
        ...state.taskLogs,
        [taskId]: logs,
      },
    })),

  setDefaultDir: (dir) => set({ defaultDir: dir }),

  deleteTask: (taskId) =>
    set((state) => {
      const newTasks = { ...state.tasks };
      delete newTasks[taskId];
      return { tasks: newTasks };
    }),
}));
