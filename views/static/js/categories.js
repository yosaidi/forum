import { apiRequest } from './api.js';
import {  updateCategories } from './state.js';

export async function loadCategories() {
    try {
        const response = await apiRequest('/categories');
        
        if (response.success) {
            updateCategories(response.data);
            return { success: true };
        }
    } catch (error) {
        console.error('Failed to load categories:', error);
        updateCategories([]);
        return { success: false, error: error.message };
    }
}