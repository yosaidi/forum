import { state } from "./state.js";
import { escapeHtml, truncateText, formatDate } from "./utils.js";
import { CONSTANTS } from "./config.js";

export function showSection(sectionId) {
  document.querySelectorAll(".section").forEach((section) => {
    section.classList.remove("active");
  });
  document.getElementById(sectionId).classList.add("active");
}

export function renderPosts() {
  const container = document.getElementById("posts-list");

  if (!state.posts || state.posts.length === 0) {
    container.innerHTML = "<p>No posts found.</p>";
    return;
  }

  container.innerHTML = state.posts
    .map(
      (post) => `
        <div class="post-card" onclick="app.showPost(${post.id})">
            <div class="post-header">
                <div>
                    <h3 class="post-title">${escapeHtml(post.title)}</h3>
                    <div class="post-meta">
                        <span>by ${escapeHtml(post.author.username)}</span>
                        <span>•</span>
                        <span>${formatDate(post.created_at)}</span>
                    </div>
                </div>
            </div>
            <div class="post-content">
                ${truncateText(
                  escapeHtml(post.content),
                  CONSTANTS.TRUNCATE_LENGTH
                )}
            </div>
            <div class="post-actions" onclick="event.stopPropagation()">
                <div class="vote-buttons">
                    <button class="vote-btn ${
                      post.user_vote === "like" ? "active like" : ""
                    }" 
                            onclick="app.votePost(${post.id}, 'like')">
                        👍 ${post.like_count}
                    </button>
                    <button class="vote-btn ${
                      post.user_vote === "dislike" ? "active dislike" : ""
                    }" 
                            onclick="app.votePost(${post.id}, 'dislike')">
                        👎 ${post.dislike_count}
                    </button>
                </div>
                <span>💬 ${post.comment_count} comments</span>
            </div>
        </div>
    `
    )
    .join("");
}

export function renderSinglePost() {
  const container = document.getElementById("single-post");
  const post = state.currentPost;
  const deleteBtn =
    state.user && state.user.id === post.author.id
      ? `
        <button class="btn btn-secondary btn-small" onclick="app.handleDeletePost(${post.id})">Delete Post</button>
    `
      : "";

  if (!post) {
    container.innerHTML = "<p>Post not found.</p>";
    return;
  }

  container.innerHTML = `
        <div class="post-card">
            <h1 class="post-title">${escapeHtml(post.title)}</h1>
            ${deleteBtn}
            <div class="post-meta">
                <span>by ${escapeHtml(post.author.username)}</span>
                <span>•</span>
                <span>${formatDate(post.created_at)}</span>
            </div>
            <div class="post-content" style="margin: 2rem 0; white-space: pre-wrap;">
                ${escapeHtml(post.content)}
            </div>
            <div class="post-actions">
                <div class="vote-buttons">
                    <button class="vote-btn ${
                      post.user_vote === "like" ? "active like" : ""
                    }" 
                            onclick="app.votePost(${post.id}, 'like')">
                        👍 ${post.like_count}
                    </button>
                    <button class="vote-btn ${
                      post.user_vote === "dislike" ? "active dislike" : ""
                    }" 
                            onclick="app.votePost(${post.id}, 'dislike')">
                        👎 ${post.dislike_count}
                    </button>
                </div>
            </div>
        </div>
    `;

  const commentForm = document.getElementById("comment-form");
  if (state.user) {
    commentForm.style.display = "block";
  } else {
    commentForm.style.display = "none";
  }
}

export function renderComments() {
  const container = document.getElementById("comments-list");

  if (!state.comments || state.comments.length === 0) {
    container.innerHTML = "<p>No comments yet. Be the first to comment!</p>";
    return;
  }

  container.innerHTML = state.comments
    .map((comment) => {
      const isCommentAuthor =
        state.user && comment.author && state.user.id === comment.author.id;

      const deleteBtn = isCommentAuthor
        ? `<button class="btn btn-secondary btn-small" 
                        onclick="app.handleDeleteComment(${comment.id})">Delete</button>`
        : "";

      return `
        <div class="comment">
            <div class="comment-header">
                <span class="comment-author">${escapeHtml(
                  comment.author.username
                )}</span>
                <span class="comment-date">${formatDate(
                  comment.created_at
                )}</span>
            </div>
            <div class="comment-content">${escapeHtml(comment.content)}</div>
            <div class="post-actions">
                <div class="vote-buttons">
                    <button class="vote-btn ${
                      comment.user_vote === "like" ? "active like" : ""
                    }" 
                            onclick="app.voteComment(${comment.id}, 'like')">
                        👍 ${comment.likes || 0}
                    </button>
                    <button class="vote-btn ${
                      comment.user_vote === "dislike" ? "active dislike" : ""
                    }" 
                            onclick="app.voteComment(${comment.id}, 'dislike')">
                        👎 ${comment.dislikes || 0}
                    </button>
                    ${deleteBtn}
                </div>
            </div>
        </div>
    `;
    })
    .join("");
}

export function renderCategories() {
  const container = document.getElementById("categories");

  container.innerHTML = "";

  const allLi = document.createElement("li");
  allLi.innerHTML =
    '<button class="active" onclick="app.filterByCategory(null)">All Posts</button>';
  container.appendChild(allLi);

  state.categories.forEach((category) => {
    const li = document.createElement("li");
    li.innerHTML = `<button onclick="app.filterByCategory(${
      category.id
    })">${escapeHtml(category.name)}</button>`;
    container.appendChild(li);
  });
}

export function renderPagination(pagination) {
  const container = document.getElementById("pagination");

  if (!pagination || pagination.total_pages <= 1) {
    container.innerHTML = "";
    return;
  }

  let html = "";

  html += `<button ${!pagination.has_prev ? "disabled" : ""} 
                   onclick="app.loadPosts(${
                     pagination.current_page - 1
                   })">Previous</button>`;

  const startPage = Math.max(1, pagination.current_page - 2);
  const endPage = Math.min(pagination.total_pages, pagination.current_page + 2);

  if (startPage > 1) {
    html += `<button onclick="app.loadPosts(1)">1</button>`;
    if (startPage > 2) html += "<span>...</span>";
  }

  for (let i = startPage; i <= endPage; i++) {
    html += `<button class="${i === pagination.current_page ? "active" : ""}" 
                       onclick="app.loadPosts(${i})">${i}</button>`;
  }

  if (endPage < pagination.total_pages) {
    if (endPage < pagination.total_pages - 1) html += "<span>...</span>";
    html += `<button onclick="app.loadPosts(${pagination.total_pages})">${pagination.total_pages}</button>`;
  }

  html += `<button ${!pagination.has_next ? "disabled" : ""} 
                   onclick="app.loadPosts(${
                     pagination.current_page + 1
                   })">Next</button>`;

  container.innerHTML = html;
}

export function showMessage(message, type = "info") {
  const messageArea = document.getElementById("message-area");
  const messageDiv = document.createElement("div");
  messageDiv.className = type;
  messageDiv.textContent = message;
  messageDiv.style.cssText = `
        position: fixed;
        top: 100px;
        right: 20px;
        padding: 1rem 2rem;
        border-radius: 4px;
        z-index: 3000;
        max-width: 400px;
        box-shadow: 0 4px 8px rgba(0,0,0,0.2);
        white-space: pre-line;
    `;

  messageArea.appendChild(messageDiv);

  setTimeout(() => {
    if (messageDiv.parentNode) {
      messageDiv.parentNode.removeChild(messageDiv);
    }
  }, 5000);
}

export function updateAvatarDisplay(avatarUrl) {
  const avatarImg = document.getElementById("profile-avatar-img");
  if (!avatarImg) return;

  if (avatarUrl) {
    if (avatarImg.tagName === "DIV") {
      // Replace placeholder with image
      const newImg = document.createElement("img");
      newImg.src = avatarUrl + "?t=" + Date.now();
      newImg.alt = state.currentProfile.username;
      newImg.className = "profile-avatar";
      newImg.id = "profile-avatar-img";
      avatarImg.parentNode.replaceChild(newImg, avatarImg);
    } else {
      // Update existing image
      avatarImg.src = avatarUrl + "?t=" + Date.now();
    }
  } else {
    // Replace with placeholder
    const newDiv = document.createElement("div");
    newDiv.className = "profile-avatar avatar-placeholder";
    newDiv.id = "profile-avatar-img";
    newDiv.textContent = state.currentProfile.username.charAt(0).toUpperCase();
    avatarImg.parentNode.replaceChild(newDiv, avatarImg);
  }
}

// Helper function to update avatar buttons
export function updateAvatarButtons(hasAvatar) {
  const avatarActions = document.querySelector(".avatar-actions");
  if (!avatarActions) return;

  // Update upload button text
  const uploadLabel = avatarActions.querySelector('label[for="avatar-upload"]');
  if (uploadLabel) {
    uploadLabel.textContent = hasAvatar ? "Change Avatar" : "Upload Avatar";
  }

  // Handle remove button
  let removeBtn = avatarActions.querySelector(".btn-secondary");

  if (hasAvatar) {
    // Show remove button
    if (!removeBtn) {
      removeBtn = document.createElement("button");
      removeBtn.onclick = () => profile.deleteAvatar();
      removeBtn.className = "btn btn-small btn-secondary";
      removeBtn.textContent = "Remove";
      avatarActions.appendChild(removeBtn);
    }
  } else {
    // Hide remove button
    if (removeBtn) {
      removeBtn.remove();
    }
  }
}
