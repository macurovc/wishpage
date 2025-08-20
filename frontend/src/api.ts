import axios from 'axios';
import { Item, FamilyMember } from './types';

// API_BASE_URL is now removed, using relative URLs instead

export const getItems = async (): Promise<Item[]> => {
  const response = await axios.get(`/api/items`); // Relative URL
  return response.data;
};

export const addItem = async (item: Omit<Item, 'id' | 'family_member_name' | 'reserved'>, password: string): Promise<Item> => {
    const response = await axios.post(`/api/items`, { ...item, password }); // Relative URL, remove category
    return response.data;
};

export const reserveItem = async (id: number, reserved: boolean, password: string): Promise<void> => {
  await axios.put(`/api/items/${id}/reserve`, { reserved, password }); // Relative URL
};

export const deleteItem = async (id: number, password: string): Promise<void> => {
  await axios.delete(`/api/items/${id}`, { data: { password }}); // Relative URL
};

export const getFamilyMembers = async (): Promise<FamilyMember[]> => {
  const response = await axios.get(`/api/family_members`); // Relative URL
  return response.data;
};

export const addFamilyMember = async (name: string, password: string): Promise<FamilyMember> => {
    const response = await axios.post(`/api/family_members`, { name, password }); // Relative URL
    return response.data;
};

export const deleteFamilyMember = async (id: number, password: string): Promise<void> => {
  await axios.delete(`/api/family_members/${id}`, { data: { password } }); // Relative URL
};

export const checkPassword = async (password: string): Promise<boolean> => {
    const response = await axios.post(`/api/check-password`, { password }); // Relative URL
    return response.data.valid;
};
