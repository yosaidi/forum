import { apiRequest } from './api.js';
import { state, updatePosts, updateCurrentPost } from './state.js';
import { escapeHtml, truncateText, formatDate } from './utils.js';

export async function loadPosts(page = 1) {
    try {
        document.getElementById('posts-loading').style.display = 'block';
        
        const params = new URLSearchParams({
            page: page,
            limit: 10,
            sort: state.currentSort
        });
        
        if (state.currentCategory) {
            params.append('category', state.currentCategory);
        }

        const response = await apiRequest(`/posts?${params}`);
        
        if (response.success) {
            updatePosts(response.data);
            state.currentPage = page;
            return { success: true, pagination: response.pagination };
        }
    } catch (error) {
        return { success: false, error: error.message };
    } finally {
        document.getElementById('posts-loading').style.display = 'none';
    }
}

export async function loadSinglePost(postId) {
    try {
        const response = await apiRequest(`/posts/${postId}`);
        
        if (response.success) {
            updateCurrentPost(response.data);
            return { success: true };
        }
    } catch (error) {
        return { success: false, error: error.message };
    }
}

export async function createPost(title, content, categoryId) {
    try {
        const response = await apiRequest('/posts', {
            method: 'POST',
            body: JSON.stringify({
                title,
                content,
                category_id: categoryId
            })
        });

        return response.success ? { success: true } : { success: false, error: 'Failed to create post' };
    } catch (error) {
        return { success: false, error: error.message };
    }
}

export async function votePost(postId, voteType) {
    try {
        const response = await apiRequest(`/posts/${postId}/vote`, {
            method: 'POST',
            body: JSON.stringify({ vote_type: voteType })
        });

        if (response.success) {
            return {
                success: true,
                data: {
                    like_count: response.data.like_count,
                    dislike_count: response.data.dislike_count,
                    user_vote: response.data.action === 'removed' ? null : voteType
                }
            };
        }
    } catch (error) {
        return { success: false, error: error.message };
    }
}