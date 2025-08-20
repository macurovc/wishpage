import { create } from 'zustand';
import { Item, FamilyMember } from './types';
import {
    getItems,
    addItem as addItemAPI,
    reserveItem as reserveItemAPI,
    deleteItem as deleteItemAPI,
    getFamilyMembers,
    addFamilyMember as addFamilyMemberAPI,
    deleteFamilyMember as deleteFamilyMemberAPI,
    checkPassword as checkPasswordAPI,
} from './api';
import { StoreApi, UseBoundStore } from 'zustand';

interface ConfirmationState {
    isOpen: boolean;
    title: string;
    message: string;
    confirmText: string;
    onConfirm: (() => void) | null;
}

interface AppState {
    items: Item[];
    familyMembers: FamilyMember[];
    isEditMode: boolean;
    password: string;
    error: string | null;
    passwordError: string | null;
    theme: 'light' | 'dark';
    selectedFamilyMemberName: string | null;
    confirmation: ConfirmationState;
    isPasswordDialogOpen: boolean;
    showConfirmation: (config: Omit<ConfirmationState, 'isOpen'>) => void;
    hideConfirmation: () => void;
    showPasswordDialog: () => void;
    hidePasswordDialog: () => void;
    fetchItems: () => Promise<void>;
    fetchFamilyMembers: () => Promise<void>;
    addItem: (newItem: Omit<Item, 'id' | 'family_member_name' | 'reserved'>) => Promise<void>;
    reserveItem: (id: number, reserved: boolean) => Promise<void>;
    deleteItem: (id: number) => Promise<void>;
    addFamilyMember: (name: string) => Promise<void>;
    deleteFamilyMember: (id: number) => Promise<void>;
    toggleEditMode: (password: string) => Promise<void>;
    setTheme: (theme: 'light' | 'dark') => void;
    selectFamilyMember: (familyName: string | null) => void;
}

export const useStore: UseBoundStore<StoreApi<AppState>> = create<AppState>((set, get) => ({
    items: [],
    familyMembers: [],
    isEditMode: false,
    password: '',
    error: null,
    passwordError: null,
    theme: 'light',
    selectedFamilyMemberName: null,
    isPasswordDialogOpen: false,
    confirmation: {
        isOpen: false,
        title: '',
        message: '',
        confirmText: '',
        onConfirm: null,
    },
    showConfirmation: (config) => set({ confirmation: { ...config, isOpen: true } }),
    hideConfirmation: () => set(state => ({ confirmation: { ...state.confirmation, isOpen: false } })),
    showPasswordDialog: () => set({ isPasswordDialogOpen: true, passwordError: null }),
    hidePasswordDialog: () => set({ isPasswordDialogOpen: false, passwordError: null }),
    fetchItems: async () => {
        try {
            const data = await getItems();
            set({ items: data, error: null });
        } catch (e) {
            set({ error: e instanceof Error ? e.message : 'An unexpected error occurred.' });
        }
    },
    fetchFamilyMembers: async () => {
        try {
            const data = await getFamilyMembers();
            set({ familyMembers: data, error: null });
        } catch (e) {
            set({ error: e instanceof Error ? e.message : 'An unexpected error occurred fetching family members.' });
        }
    },
    addItem: async (newItem: Omit<Item, 'id' | 'family_member_name' | 'reserved'>) => {
        await addItemAPI(newItem, get().password);
        await get().fetchItems();
    },
    reserveItem: async (id: number, reserved: boolean) => {
        try {
            await reserveItemAPI(id, reserved, get().password);
            set((state) => ({
                items: state.items.map((item) =>
                    item.id === id ? { ...item, reserved } : item
                ),
                error: null,
            }));
        } catch (e) {
            set({ error: e instanceof Error ? e.message : 'Failed to reserve item.' });
        }
    },
    deleteItem: async (id: number) => {
        try {
            await deleteItemAPI(id, get().password);
            await get().fetchItems();
        } catch (e) {
            set({ error: e instanceof Error ? e.message : 'Failed to delete item.' });
        }
    },
    addFamilyMember: async (name: string) => {
        try {
            await addFamilyMemberAPI(name, get().password);
            await get().fetchFamilyMembers();
        } catch {
            set({ error: 'Failed to add family member.' });
        }
    },
    deleteFamilyMember: async (id: number) => {
        try {
            await deleteFamilyMemberAPI(id, get().password);
            await get().fetchFamilyMembers();
        } catch {
            set({ error: 'Failed to delete family member.' });
        }
    },
    toggleEditMode: async (inputPassword: string) => {
        if (!get().isEditMode) {
            const isValidPassword = await checkPasswordAPI(inputPassword);
            if (isValidPassword) {
                set({ isEditMode: true, password: inputPassword, error: null, isPasswordDialogOpen: false });
            } else {
                set({ passwordError: 'Incorrect password.' });
            }
        } else {
            set({ isEditMode: false, password: '' });
        }
    },
    setTheme: (theme: 'light' | 'dark') => {
        set({ theme });
        document.documentElement.setAttribute('data-theme', theme);
        localStorage.setItem('theme', theme);
    },
    selectFamilyMember: (familyName: string | null) => {
        set({ selectedFamilyMemberName: familyName });
    },
}));
