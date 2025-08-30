import { votePost as apiVotePost } from './posts.js';
import { voteComment as apiVoteCommentDirect } from './comments.js';
import { state } from './state.js';
import { renderPosts, renderSinglePost, renderComments, showMessage } from './ui.js';

export async function handlePostVote(postId, voteType) {
    if (!state.user) {
        showMessage('Please login to vote', 'error');
        return;
    }

    const result = await apiVotePost(postId, voteType);
    
    if (result.success) {
        // Update the vote counts in the current view
        if (state.currentPost && state.currentPost.id === postId) {
            state.currentPost.like_count = result.data.like_count;
            state.currentPost.dislike_count = result.data.dislike_count;
            state.currentPost.user_vote = result.data.user_vote;
            renderSinglePost();
        } else {
            // Update in posts list
            const post = state.posts.find(p => p.id === postId);
            if (post) {
                post.like_count = result.data.like_count;
                post.dislike_count = result.data.dislike_count;
                post.user_vote = result.data.user_vote;
                renderPosts();
            }
        }
    } else {
        showMessage(result.error, 'error');
    }
}

export async function handleCommentVote(commentId, voteType) {
    if (!state.user) {
        showMessage('Please login to vote', 'error');
        return;
    }

    const result = await apiVoteCommentDirect(commentId, voteType);
    
    if (result.success) {
        const comment = state.comments.find(c => c.id === commentId);
        if (comment) {
            comment.likes = result.data.like_count;
            comment.dislikes = result.data.dislike_count;
            comment.user_vote = result.data.user_vote;
        }
        renderComments();
    } else {
        showMessage(result.error, 'error');
    }
}