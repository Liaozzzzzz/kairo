import { create } from 'zustand';
import { Task } from '../types';
import { TaskStatus } from '@/data/variables';
import { DeleteTask as DeleteTaskWails } from '@root/wailsjs/go/main/App';

interface TaskState {
  tasks: Record<string, Task>;
  taskLogs: Record<string, string[]>;

  // Actions
  setTasks: (tasks: Record<string, Task>) => void;
  updateTask: (taskId: string, updates: Partial<Task>) => void;
  updateTaskProgress: (data: {
    id: string;
    progress: number;
    total_size?: string;
    speed?: string;
    eta?: string;
  }) => void;
  addTaskLog: (taskId: string, message: string, replace?: boolean) => void;
  setTaskLogs: (taskId: string, logs: string[]) => void;
  deleteTask: (taskId: string, purge?: boolean) => Promise<void>;
}

export const useTaskStore = create<TaskState>((set, get) => ({
  tasks: {},
  taskLogs: {},

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

  deleteTask: async (taskId, purge = false) => {
    const state = get();
    const task = state.tasks[taskId];
    if (!task) return;

    // Validate: Abort if the target task is in a protected state
    if (task.status === TaskStatus.Merging || task.status === TaskStatus.Trimming) {
      return;
    }

    // Backend returns the list of all deleted task IDs (including cascaded ones)
    const deletedIds = await DeleteTaskWails(taskId, purge);

    set((state) => {
      const newTasks = { ...state.tasks };
      const newTaskLogs = { ...state.taskLogs };

      deletedIds.forEach((id) => {
        delete newTasks[id];
        delete newTaskLogs[id];
      });

      return { tasks: newTasks, taskLogs: newTaskLogs };
    });
  },
}));
