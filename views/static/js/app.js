import { checkAuthStatus, login, register, logout } from "./auth.js";
import { loadPosts, loadSinglePost, createPost, deletePost } from "./posts.js";
import { loadComments, submitComment, deleteComment } from "./comments.js";
import { loadCategories } from "./categories.js";
import { handlePostVote, handleCommentVote } from "./voting.js";
import {
  showSection,
  renderPosts,
  renderSinglePost,
  renderComments,
  renderCategories,
  renderPagination,
  showMessage,
} from "./ui.js";
import { state } from "./state.js";
import { profile } from "./profile.js";

// Main Application Controller
export const app = {
  // Navigation
  showSection,

  showHome() {
    this.showSection("home-section");
    this.loadPosts();
  },

  showLogin() {
    this.showSection("login-section");
  },

  showRegister() {
    this.showSection("register-section");
  },

  showCreatePost() {
    if (!state.user) {
      showMessage("Please login to create a post", "error");
      this.showLogin();
      return;
    }
    this.showSection("create-post-section");
    this.loadCategoriesForForm();
  },

  showPost(postId) {
    this.showSection("post-section");
    this.loadSinglePost(postId);
  },

  // Authentication handlers
  async login(event) {
    event.preventDefault();
    const username = document.getElementById("login-username").value;
    const password = document.getElementById("login-password").value;

    const result = await login(username, password);

    if (result.success) {
      showMessage("Login successful!", "success");
      this.showSorting();
      this.showHome();
    } else {
      showMessage(result.error, "error");
    }
  },

  async register(event) {
    event.preventDefault();
    const username = document.getElementById("register-username").value;
    const email = document.getElementById("register-email").value;
    const password = document.getElementById("register-password").value;

    const result = await register(username, email, password);

    if (result.success) {
      showMessage("Registration successful!", "success");
      this.showHome();
    } else {
      showMessage(result.error, "error");
    }
  },

  async logout() {
    const result = await logout();

    if (result.success) {
      showMessage("Logged out successfully", "success");
      this.showSorting();
      this.showHome();
    } else {
      showMessage(result.error, "error");
    }
  },

  // Posts handlers
  async loadPosts(page = 1) {
    const result = await loadPosts(page);

    if (result.success) {
      renderPosts();
      renderPagination(result.pagination);
    } else {
      showMessage(result.error, "error");
    }
  },

  async loadSinglePost(postId) {
    const result = await loadSinglePost(postId);

    if (result.success) {
      renderSinglePost();
      this.loadComments(postId);
    } else {
      showMessage(result.error, "error");
    }
  },

  async createPost(event) {
    event.preventDefault();

    if (!state.user) {
      showMessage("Please login to create a post", "error");
      return;
    }

    const title = document.getElementById("post-title").value;
    const content = document.getElementById("post-content").value;
    const categoryId = parseInt(document.getElementById("post-category").value);

    const result = await createPost(title, content, categoryId);

    if (result.success) {
      showMessage("Post created successfully!", "success");
      document.getElementById("post-title").value = "";
      document.getElementById("post-content").value = "";
      document.getElementById("post-category").value = "";
      this.showHome();
    } else {
      showMessage(result.error, "error");
    }
  },

  async handleDeletePost(postId) {
    if (!state.user) {
      showMessage("Please login to delete a post", "error");
      return;
    }

    if (!confirm("Are you sure you want to delete this post?")) {
      return;
    }

    const result = await deletePost(postId);

    if (result.success) {
      showMessage("Post deleted successfully!", "success");
      this.showHome();
    } else {
      showMessage(result.error, "error");
    }
  },

  // Voting handlers
  async votePost(postId, voteType) {
    await handlePostVote(postId, voteType);
  },

  async voteComment(commentId, voteType) {
    await handleCommentVote(commentId, voteType);
  },

  // Comments handlers
  async loadComments(postId) {
    const result = await loadComments(postId);

    if (result.success) {
      renderComments();
    } else {
      showMessage(result.error, "error");
    }
  },

  async handleDeleteComment(commentId) {
    if (!state.user) {
      showMessage("Please login to delete a comment", "error");
      return;
    }

    if (!confirm("Are you sure you want to delete this comment?")) {
      return;
    }

    const result = await deleteComment(commentId);

    if (result.success) {
      showMessage("Comment deleted successfully!", "success");
      this.loadComments(state.currentPost.id);
    } else {
      showMessage(result.error, "error");
    }
  },

  async submitComment(event) {
    event.preventDefault();

    if (!state.user) {
      showMessage("Please login to comment", "error");
      return;
    }

    const content = document.getElementById("comment-content").value;

    if (!content.trim()) {
      showMessage("Comment cannot be empty", "error");
      return;
    }

    const result = await submitComment(content, state.currentPost.id);

    if (result.success) {
      document.getElementById("comment-content").value = "";
      this.loadComments(state.currentPost.id);
      showMessage("Comment added successfully!", "success");
    } else {
      showMessage(result.error, "error");
    }
  },

  // Categories
  async loadCategories() {
    const result = await loadCategories();

    if (result.success) {
      renderCategories();
    }
  },

  async loadCategoriesForForm() {
    try {
      const select = document.getElementById("post-category");
      select.innerHTML = '<option value="">Select a category</option>';

      const result = await loadCategories();
      if (result.success) {
        state.categories.forEach((category) => {
          const option = document.createElement("option");
          option.value = category.id;
          option.textContent = category.name;
          select.appendChild(option);
        });
      }
    } catch (error) {
      console.error("Failed to load categories for form:", error);
    }
  },

  // Filtering and Sorting
  filterByCategory(categoryId) {
    state.currentCategory = categoryId;
    state.currentPage = 1;

    document.querySelectorAll(".category-list button").forEach((btn) => {
      btn.classList.remove("active");
    });

    if (event && event.target) {
      event.target.classList.add("active");
    }

    this.loadPosts(1);
    this.showHome();
  },

  changeSorting(sortBy) {
    state.currentSort = sortBy;
    state.currentPage = 1;
    this.loadPosts(1);
  },
  showSorting() {
    const sortSelect = document.getElementById("sort-select");
    if (state.user) {
      sortSelect.innerHTML = `
        <option value="newest">Newest</option>
        <option value="oldest">Oldest</option>
        <option value="popular">Popular</option>
        <option value="my_posts">My Posts</option>
        <option value="my_likes">My Likes</option>
        <option value="my_dislikes">My Dislikes</option>
      `;
    } else {
      sortSelect.innerHTML = `
        <option value="newest">Newest</option>
        <option value="oldest">Oldest</option>
        <option value="popular">Popular</option>
      `;
    }
    sortSelect.value = state.currentSort;
  },
  // Initialize Application
  async init() {
    await checkAuthStatus();
    this.showSorting();
    this.loadPosts();
    this.loadCategories();

    window.addEventListener("popstate", () => {
      this.showHome();
    });
  },
};

// Make profile globally available for HTML buttons
window.profile = profile;
