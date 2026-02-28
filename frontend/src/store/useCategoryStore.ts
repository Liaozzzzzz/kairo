import { create } from 'zustand';
import { Category, CategorySource } from '@/types';
import { schema } from '@root/wailsjs/go/models';
import {
  GetCategories,
  CreateCategory,
  UpdateCategory,
  DeleteCategory,
} from '@root/wailsjs/go/main/App';

interface CategoryState {
  categories: Category[];
  isLoading: boolean;
  fetchCategories: () => Promise<void>;
  createCategory: (name: string, prompt: string) => Promise<Category | null>;
  updateCategory: (id: string, name: string, prompt: string) => Promise<Category | null>;
  deleteCategory: (id: string) => Promise<void>;
}

const normalizeCategory = (category: schema.Category): Category => ({
  id: category.id,
  name: category.name,
  prompt: category.prompt,
  source:
    category.source === CategorySource.Builtin ? CategorySource.Builtin : CategorySource.Custom,
  created_at: category.created_at,
  updated_at: category.updated_at,
});

export const useCategoryStore = create<CategoryState>((set) => ({
  categories: [],
  isLoading: false,

  fetchCategories: async () => {
    set({ isLoading: true });
    try {
      const categories = await GetCategories();
      set({ categories: Array.isArray(categories) ? categories.map(normalizeCategory) : [] });
    } catch (error) {
      console.error('Failed to fetch categories:', error);
    } finally {
      set({ isLoading: false });
    }
  },

  createCategory: async (name: string, prompt: string) => {
    try {
      const created = await CreateCategory(name, prompt);
      if (created) {
        const normalized = normalizeCategory(created);
        set((state) => ({ categories: [...state.categories, normalized] }));
        return normalized;
      }
      return null;
    } catch (error) {
      console.error('Failed to create category:', error);
      return null;
    }
  },

  updateCategory: async (id: string, name: string, prompt: string) => {
    try {
      const updated = await UpdateCategory(id, name, prompt);
      if (updated) {
        const normalized = normalizeCategory(updated);
        set((state) => ({
          categories: state.categories.map((item) => (item.id === id ? normalized : item)),
        }));
        return normalized;
      }
      return null;
    } catch (error) {
      console.error('Failed to update category:', error);
      return null;
    }
  },

  deleteCategory: async (id: string) => {
    try {
      await DeleteCategory(id);
      set((state) => ({ categories: state.categories.filter((item) => item.id !== id) }));
    } catch (error) {
      console.error('Failed to delete category:', error);
    }
  },
}));
