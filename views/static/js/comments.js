import { apiRequest } from './api.js';
import { updateComments } from './state.js';

export async function loadComments(postId) {
    try {
        const response = await apiRequest(`/posts/${postId}/comments`);
        
        if (response.success) {
            updateComments(response.data);
            return { success: true };
        }
    } catch (error) {
        return { success: false, error: error.message };
    }
}

export async function submitComment(content, postId) {
    try {
        const response = await apiRequest('/comments', {
            method: 'POST',
            body: JSON.stringify({
                content: content,
                post_id: postId
            })
        });

        return response.success ? { success: true } : { success: false, error: 'Failed to submit comment' };
    } catch (error) {
        return { success: false, error: error.message };
    }
}

export async function voteComment(commentId, voteType) {
    try {
        const response = await apiRequest(`/comments/${commentId}/vote`, {
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