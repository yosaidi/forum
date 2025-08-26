    // API Configuration
        const API_BASE = '/api';

        // Application State
        const state = {
            user: null,
            posts: [],
            currentPost: null,
            categories: [],
            comments: [],
            currentPage: 1,
            currentCategory: null,
            currentSort: 'newest'
        };

        // API Helper Functions
        async function apiRequest(endpoint, options = {}) {
            const url = `${API_BASE}${endpoint}`;
            
            const config = {
                headers: {
                    'Content-Type': 'application/json',
                    ...options.headers
                },
                credentials: 'include',
                ...options
            };

            try {
                const response = await fetch(url, config);
                const data = await response.json();
                
                if (!response.ok) {
                    throw new Error(data.error || data.message || 'Request failed');
                }
                
                return data;
            } catch (error) {
                console.error('API Error:', error);
                throw error;
            }
        }

        // Authentication Functions
        async function checkAuthStatus() {
            try {
                const response = await apiRequest('/auth/me');
                if (response.success) {
                    state.user = response.data;
                    updateNavigation();
                    return true;
                }
            } catch (error) {
                state.user = null;
            }
            updateNavigation();
            return false;
        }

        function updateNavigation() {
            const guestNav = document.getElementById('guest-nav');
            const userNav = document.getElementById('user-nav');
            const username = document.getElementById('username');
            const avatar = document.getElementById('user-avatar');

            if (state.user) {
                guestNav.style.display = 'none';
                userNav.style.display = 'flex';
                username.textContent = state.user.username;
                avatar.textContent = state.user.username.charAt(0).toUpperCase();
            } else {
                guestNav.style.display = 'flex';
                userNav.style.display = 'none';
            }
        }

        // Main Application Object
        const app = {
            // Navigation
            showSection(sectionId) {
                document.querySelectorAll('.section').forEach(section => {
                    section.classList.remove('active');
                });
                document.getElementById(sectionId).classList.add('active');
            },

            showHome() {
                this.showSection('home-section');
                this.loadPosts();
            },

            showLogin() {
                this.showSection('login-section');
            },

            showRegister() {
                this.showSection('register-section');
            },

            showCreatePost() {
                if (!state.user) {
                    this.showMessage('Please login to create a post', 'error');
                    this.showLogin();
                    return;
                }
                this.showSection('create-post-section');
                this.loadCategoriesForForm();
            },

            showPost(postId) {
                this.showSection('post-section');
                this.loadSinglePost(postId);
            },

            // Authentication
            async login(event) {
                event.preventDefault();
                const username = document.getElementById('login-username').value;
                const password = document.getElementById('login-password').value;

                try {
                    const response = await apiRequest('/auth/login', {
                        method: 'POST',
                        body: JSON.stringify({ username, password })
                    });

                    if (response.success) {
                        state.user = response.data.user;
                        this.showMessage('Login successful!', 'success');
                        updateNavigation();
                        this.showHome();
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            async register(event) {
                event.preventDefault();
                const username = document.getElementById('register-username').value;
                const email = document.getElementById('register-email').value;
                const password = document.getElementById('register-password').value;

                try {
                    const response = await apiRequest('/auth/register', {
                        method: 'POST',
                        body: JSON.stringify({ username, email, password })
                    });

                    if (response.success) {
                        state.user = response.data.user;
                        this.showMessage('Registration successful!', 'success');
                        updateNavigation();
                        this.showHome();
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            async logout() {
                try {
                    await apiRequest('/auth/logout', { method: 'POST' });
                    state.user = null;
                    updateNavigation();
                    this.showMessage('Logged out successfully', 'success');
                    this.showHome();
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            // Posts
            async loadPosts(page = 1) {
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
                        state.posts = response.data;
                        state.currentPage = page;
                        this.renderPosts();
                        this.renderPagination(response.pagination);
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                } finally {
                    document.getElementById('posts-loading').style.display = 'none';
                }
            },

            renderPosts() {
                const container = document.getElementById('posts-list');
                
                if (!state.posts || state.posts.length === 0) {
                    container.innerHTML = '<p>No posts found.</p>';
                    return;
                }

                container.innerHTML = state.posts.map(post => `
                    <div class="post-card" onclick="app.showPost(${post.id})">
                        <div class="post-header">
                            <div>
                                <h3 class="post-title">${this.escapeHtml(post.title)}</h3>
                                <div class="post-meta">
                                    <span>by ${this.escapeHtml(post.author.username)}</span>
                                    <span>•</span>
                                    <span>${post.category}</span>
                                    <span>•</span>
                                    <span>${this.formatDate(post.created_at)}</span>
                                </div>
                            </div>
                        </div>
                        <div class="post-content">
                            ${this.truncateText(this.escapeHtml(post.content), 200)}
                        </div>
                        <div class="post-actions" onclick="event.stopPropagation()">
                            <div class="vote-buttons">
                                <button class="vote-btn ${post.user_vote === 'like' ? 'active like' : ''}" 
                                        onclick="app.votePost(${post.id}, 'like')">
                                    👍 ${post.like_count}
                                </button>
                                <button class="vote-btn ${post.user_vote === 'dislike' ? 'active dislike' : ''}" 
                                        onclick="app.votePost(${post.id}, 'dislike')">
                                    👎 ${post.dislike_count}
                                </button>
                            </div>
                            <span>💬 ${post.comment_count} comments</span>
                        </div>
                    </div>
                `).join('');
            },

            async loadSinglePost(postId) {
                try {
                    const response = await apiRequest(`/posts/${postId}`);
                    
                    if (response.success) {
                        state.currentPost = response.data;
                        this.renderSinglePost();
                        this.loadComments(postId);
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            renderSinglePost() {
                const container = document.getElementById('single-post');
                const post = state.currentPost;
                
                container.innerHTML = `
                    <div class="post-card">
                        <h1 class="post-title">${this.escapeHtml(post.title)}</h1>
                        <div class="post-meta">
                            <span>by ${this.escapeHtml(post.author.username)}</span>
                            <span>•</span>
                            <span>${post.category}</span>
                            <span>•</span>
                            <span>${this.formatDate(post.created_at)}</span>
                        </div>
                        <div class="post-content" style="margin: 2rem 0; white-space: pre-wrap;">
                            ${this.escapeHtml(post.content)}
                        </div>
                        <div class="post-actions">
                            <div class="vote-buttons">
                                <button class="vote-btn ${post.user_vote === 'like' ? 'active like' : ''}" 
                                        onclick="app.votePost(${post.id}, 'like')">
                                    👍 ${post.like_count}
                                </button>
                                <button class="vote-btn ${post.user_vote === 'dislike' ? 'active dislike' : ''}" 
                                        onclick="app.votePost(${post.id}, 'dislike')">
                                    👎 ${post.dislike_count}
                                </button>
                            </div>
                        </div>
                    </div>
                `;

                // Show comment form if logged in
                const commentForm = document.getElementById('comment-form');
                if (state.user) {
                    commentForm.style.display = 'block';
                } else {
                    commentForm.style.display = 'none';
                }
            },

            async createPost(event) {
                event.preventDefault();
                
                if (!state.user) {
                    this.showMessage('Please login to create a post', 'error');
                    return;
                }

                const title = document.getElementById('post-title').value;
                const content = document.getElementById('post-content').value;
                const categoryId = parseInt(document.getElementById('post-category').value);

                try {
                    const response = await apiRequest('/posts', {
                        method: 'POST',
                        body: JSON.stringify({
                            title,
                            content,
                            category_id: categoryId
                        })
                    });

                    if (response.success) {
                        this.showMessage('Post created successfully!', 'success');
                        document.getElementById('post-title').value = '';
                        document.getElementById('post-content').value = '';
                        document.getElementById('post-category').value = '';
                        this.showHome();
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            // Voting
            async votePost(postId, voteType) {

                if (!state.user) {
                    this.showMessage('Please login to vote', 'error');
                    this.showLogin();
                    return;
                }

                try {
                    const response = await apiRequest(`/posts/${postId}/vote`, {
                        method: 'POST',
                        body: JSON.stringify({ vote_type: voteType })
                    });

                    if (response.success) {
                        // Update the vote counts in the current view
                        if (state.currentPost && state.currentPost.id === postId) {
                            state.currentPost.like_count = response.data.like_count;
                            state.currentPost.dislike_count = response.data.dislike_count;
                            state.currentPost.user_vote = response.data.action === 'removed' ? null : voteType;
                            this.renderSinglePost();
                        } else {
                            // Update in posts list
                            const post = state.posts.find(p => p.id === postId);
                            if (post) {
                                post.like_count = response.data.like_count;
                                post.dislike_count = response.data.dislike_count;
                                post.user_vote = response.data.action === 'removed' ? null : voteType;
                                this.renderPosts();
                            }
                        }
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            async voteComment(commentId, voteType) {
                if (!state.user) {
                    this.showMessage('Please login to vote', 'error');
                    return;
                }

                try {
                    const response = await apiRequest(`/comments/${commentId}/vote`, {
                        method: 'POST',
                        body: JSON.stringify({ vote_type: voteType })
                    });

                    if (response.success) {
                        // Reload comments to update vote counts
                        this.loadComments(state.currentPost.id);
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            // Comments
            async loadComments(postId) {
                try {
                    const response = await apiRequest(`/posts/${postId}/comments`);
                    
                    if (response.success) {
                        state.comments = response.data;
                        this.renderComments();
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            renderComments() {
                const container = document.getElementById('comments-list');
                
                if (!state.comments || state.comments.length === 0) {
                    container.innerHTML = '<p>No comments yet. Be the first to comment!</p>';
                    return;
                }

                container.innerHTML = state.comments.map(comment => `
                    <div class="comment">
                        <div class="comment-header">
                            <span class="comment-author">${this.escapeHtml(comment.author.username)}</span>
                            <span class="comment-date">${this.formatDate(comment.created_at)}</span>
                        </div>
                        <div class="comment-content">${this.escapeHtml(comment.content)}</div>
                        <div class="post-actions">
                            <div class="vote-buttons">
                                <button class="vote-btn ${comment.user_vote === 'like' ? 'active like' : ''}" 
                                        onclick="app.voteComment(${comment.id}, 'like')">
                                    👍 ${comment.likes || 0}
                                </button>
                                <button class="vote-btn ${comment.user_vote === 'dislike' ? 'active dislike' : ''}" 
                                        onclick="app.voteComment(${comment.id}, 'dislike')">
                                    👎 ${comment.dislikes || 0}
                                </button>
                            </div>
                        </div>
                    </div>
                `).join('');
            },

            async submitComment(event) {
                event.preventDefault();
                
                if (!state.user) {
                    this.showMessage('Please login to comment', 'error');
                    return;
                }

                const content = document.getElementById('comment-content').value;
                
                if (!content.trim()) {
                    this.showMessage('Comment cannot be empty', 'error');
                    return;
                }

                try {
                    const response = await apiRequest('/comments', {
                        method: 'POST',
                        body: JSON.stringify({
                            content: content,
                            post_id: state.currentPost.id
                        })
                    });

                    if (response.success) {
                        document.getElementById('comment-content').value = '';
                        this.loadComments(state.currentPost.id);
                        this.showMessage('Comment added successfully!', 'success');
                    }
                } catch (error) {
                    this.showMessage(error.message, 'error');
                }
            },

            // Categories

            async loadCategories() {
                try {
                    const response = await apiRequest('/categories');
                    
                    if (response.success) {
                        state.categories = response.data;
                        this.renderCategories();
                    }
                } catch (error) {
                    console.error('Failed to load categories:', error);
                    // Fallback to empty categories
                    state.categories = [];
                    this.renderCategories();
                }
            },



            renderCategories() {
                const container = document.getElementById('categories');
                
                // Clear existing categories
                container.innerHTML = '';
                
                // Add "All Posts" button first
                const allLi = document.createElement('li');
                allLi.innerHTML = '<button class="active" onclick="app.filterByCategory(null)">All Posts</button>';
                container.appendChild(allLi);

                // Add category buttons
                state.categories.forEach(category => {
                    const li = document.createElement('li');
                    li.innerHTML = `<button onclick="app.filterByCategory(${category.id})">${this.escapeHtml(category.name)}</button>`;
                    container.appendChild(li);
                });
            },


            async loadCategoriesForForm() {
                try {
                    const select = document.getElementById('post-category');
                    select.innerHTML = '<option value="">Select a category</option>';
                    
                    // Fetch categories from backend
                    const response = await apiRequest('/categories');
                    if (response.success) {
                        response.data.forEach(category => {
                            const option = document.createElement('option');
                            option.value = category.id;
                            option.textContent = category.name;
                            select.appendChild(option);
                        });
                    }   
                } catch (error) {
                    console.error('Failed to load categories for form:', error);
                }
            },

            // Filtering and Sorting
            filterByCategory(categoryId) {
                state.currentCategory = categoryId;
                state.currentPage = 1;
                
                // Update active category button
                document.querySelectorAll('.category-list button').forEach(btn => {
                    btn.classList.remove('active');
                });
                
                // Add active class to clicked button
                if (event && event.target) {
                    event.target.classList.add('active');
                }
                
                this.loadPosts(1);
                this.showHome();
            },

            changeSorting(sortBy) {
                state.currentSort = sortBy;
                state.currentPage = 1;
                this.loadPosts(1);
            },

            // Pagination
            renderPagination(pagination) {
                const container = document.getElementById('pagination');
                
                if (!pagination || pagination.total_pages <= 1) {
                    container.innerHTML = '';
                    return;
                }

                let html = '';
                
                // Previous button
                html += `<button ${!pagination.has_prev ? 'disabled' : ''} 
                               onclick="app.loadPosts(${pagination.current_page - 1})">Previous</button>`;
                
                // Page numbers
                const startPage = Math.max(1, pagination.current_page - 2);
                const endPage = Math.min(pagination.total_pages, pagination.current_page + 2);
                
                if (startPage > 1) {
                    html += `<button onclick="app.loadPosts(1)">1</button>`;
                    if (startPage > 2) html += '<span>...</span>';
                }
                
                for (let i = startPage; i <= endPage; i++) {
                    html += `<button class="${i === pagination.current_page ? 'active' : ''}" 
                                   onclick="app.loadPosts(${i})">${i}</button>`;
                }
                
                if (endPage < pagination.total_pages) {
                    if (endPage < pagination.total_pages - 1) html += '<span>...</span>';
                    html += `<button onclick="app.loadPosts(${pagination.total_pages})">${pagination.total_pages}</button>`;
                }
                
                // Next button
                html += `<button ${!pagination.has_next ? 'disabled' : ''} 
                               onclick="app.loadPosts(${pagination.current_page + 1})">Next</button>`;
                
                container.innerHTML = html;
            },

            // Utility Functions
            showMessage(message, type = 'info') {
                const messageArea = document.getElementById('message-area');
                const messageDiv = document.createElement('div');
                messageDiv.className = type;
                messageDiv.textContent = message;
                messageDiv.style.cssText = `
                    position: fixed;
                    top: 20px;
                    right: 20px;
                    padding: 1rem 2rem;
                    border-radius: 4px;
                    z-index: 3000;
                    max-width: 400px;
                    box-shadow: 0 4px 8px rgba(0,0,0,0.2);
                `;
                
                messageArea.appendChild(messageDiv);
                
                // Auto-remove after 5 seconds
                setTimeout(() => {
                    if (messageDiv.parentNode) {
                        messageDiv.parentNode.removeChild(messageDiv);
                    }
                }, 5000);
            },

            escapeHtml(text) {
                const map = {
                    '&': '&amp;',
                    '<': '&lt;',
                    '>': '&gt;',
                    '"': '&quot;',
                    "'": '&#039;'
                };
                return text.replace(/[&<>"']/g, m => map[m]);
            },

            truncateText(text, maxLength) {
                if (text.length <= maxLength) return text;
                return text.substr(0, maxLength) + '...';
            },

            formatDate(dateString) {
                const date = new Date(dateString);
                const now = new Date();
                const diffInHours = (now - date) / (1000 * 60 * 60);
                
                if (diffInHours < 1) {
                    return `${Math.floor(diffInHours * 60)} minutes ago`;
                } else if (diffInHours < 24) {
                    return `${Math.floor(diffInHours)} hours ago`;
                } else if (diffInHours < 24 * 7) {
                    return `${Math.floor(diffInHours / 24)} days ago`;
                } else {
                    return date.toLocaleDateString();
                }
            },

            // Initialize Application
            async init() {
                // Check authentication status
                await checkAuthStatus();
                
                // Load initial data
                this.loadPosts();
                this.loadCategories();
                
                // Set up event listeners
                window.addEventListener('popstate', () => {
                    // Handle browser back/forward buttons
                    this.showHome();
                });
            }
        };

        // Initialize the application when DOM is loaded
        document.addEventListener('DOMContentLoaded', () => {
            app.init();
        });

        // Handle keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.ctrlKey || e.metaKey) {
                switch (e.key) {
                    case 'k':
                        e.preventDefault();
                        document.querySelector('#login-username, #register-username')?.focus();
                        break;
                    case 'n':
                        e.preventDefault();
                        if (state.user) {
                            app.showCreatePost();
                        }
                        break;
                }
            }
        });

        // Export app for debugging in console
        window.app = app;