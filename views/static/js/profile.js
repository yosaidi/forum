import { apiRequest } from "./api.js";
import { state, updateUser, updateCurrentProfile } from "./state.js";
import { showMessage, updateAvatarDisplay, updateAvatarButtons } from "./ui.js";

// Profile Management Functions
export async function loadUserProfile(userId) {
  try {
    const response = await apiRequest(`/users/${userId}`);
    if (response.success) {
      updateCurrentProfile(response.data);
      return { success: true, data: response.data };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
}

export async function updateProfile(userId, updates) {
  try {
    const response = await apiRequest(`/users/${userId}`, {
      method: "PUT",
      body: JSON.stringify(updates),
    });

    if (response.success) {
      // Update current user if it's their own profile
      if (state.user && state.user.id === userId) {
        updateUser(response.data);
        updateNavigation();
      }

      updateCurrentProfile(response.data);

      return { success: true, data: response.data };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
}

export async function uploadAvatar(userId, file) {
  try {
    // Validate file before upload
    if (!file) {
      throw new Error("No file selected");
    }

    // Check file size (5MB limit)
    if (file.size > 5 * 1024 * 1024) {
      throw new Error("File size must be less than 5MB");
    }

    // Check file type
    const allowedTypes = ["image/jpeg", "image/png", "image/gif", "image/webp"];
    if (!allowedTypes.includes(file.type)) {
      throw new Error("File must be an image (JPEG, PNG, GIF, or WebP)");
    }

    // Create FormData for file upload
    const formData = new FormData();
    formData.append("avatar", file);

    // Upload file (note: different from regular API request due to FormData)
    const response = await fetch(`/api/users/${userId}/avatar`, {
      method: "POST",
      body: formData,
      credentials: "include",
    });

    const data = await response.json();

    if (!response.ok) {
      throw new Error(data.error || data.message || "Upload failed");
    }

    // Update current user avatar if it's their own profile
    if (state.user && state.user.id === userId) {
      state.user.avatar = data.data.avatar_url;
      updateNavigation();
    }

    return { success: true, data: data.data };
  } catch (error) {
    return { success: false, error: error.message };
  }
}

export async function deleteAvatar(userId) {
  try {
    const response = await apiRequest(`/users/${userId}/avatar`, {
      method: "DELETE",
    });

    if (response.success) {
      // Update current user avatar if it's their own profile
      if (state.user && state.user.id === userId) {
        state.user.avatar = null;
        updateNavigation();
      }
      return { success: true, data: response.data };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
}

// UI Functions for Profile Management
export function renderProfile() {
  const container = document.getElementById("profile-content");
  const profile = state.currentProfile;
  const isOwnProfile = state.user && state.user.id === profile.id;
  const hasAvatar = profile && profile.avatar;

  const avatarContent = profile.avatar
    ? `<img src="${profile.avatar}" alt="${profile.username}" class="profile-avatar" id="profile-avatar-img">`
    : `<div class="profile-avatar avatar-placeholder" id="profile-avatar-img">
        ${profile.username.charAt(0)}
    </div>`;

  const uploadButtonText = hasAvatar ? "Change Avatar" : "Upload Avatar";
  const removeBtn = profile.avatar
    ? `   <button onclick="profile.deleteAvatar()" class="btn btn-small btn-secondary">Remove</button>`
    : `<span></span>`;

  container.innerHTML = /*html*/ `
    
    <div class="profile-header">
      <div class="profile-avatar-section" id="profile-avatar-section">
        ${avatarContent}
        ${
          isOwnProfile
            ? `
          <div class="avatar-actions">
            <label for="avatar-upload" class="btn btn-small btn-primary">
            ${uploadButtonText}
            </label>
            <input type="file" id="avatar-upload" accept="image/*" style="display: none;">
            ${removeBtn}
            </div>
        `
            : ""
        }
      </div>
      <div class="profile-info">
        <h1>${profile.username}</h1>
        ${
          isOwnProfile
            ? `<p class="profile-email">${profile.email || ""}</p>`
            : ""
        }
        <div class="profile-stats">
          <div class="stat">
            <span class="stat-number">${profile.post_count}</span>
            <span class="stat-label">Posts</span>
          </div>
          <div class="stat">
            <span class="stat-number">${profile.comment_count}</span>
            <span class="stat-label">Comments</span>
          </div>
          <div class="stat">
            <span class="stat-number">${formatAccountAge(
              profile.created_at
            )}</span>
            <span class="stat-label">Member for</span>
          </div>
        </div>
        ${
          isOwnProfile
            ? `<button onclick="profile.showEditForm()" class="btn btn-primary">Edit Profile</button>`
            : ""
        }
      </div>
    </div>

    <div class="profile-content-tabs">
      <div class="tabs">
        <button class="tab-btn active" onclick="profile.showTab('posts')">Posts</button>
        <button class="tab-btn" onclick="profile.showTab('comments')">Comments</button>
      </div>
      <div id="profile-tab-content" class="tab-content">
        <div id="profile-posts" class="tab-pane active"></div>
        <div id="profile-comments" class="tab-pane"></div>
      </div>
    </div>
  `;

  // Setup avatar upload handler if it's user's own profile
  if (isOwnProfile) {
    setupAvatarUpload();
  }

  // Load posts by default
  loadUserPosts(profile.id);
}

export function renderEditProfile() {
  const container = document.getElementById("profile-content");
  const profile = state.currentProfile;

  container.innerHTML = `
        <div class="form">
            <h2>Edit Profile</h2>
            <form onsubmit="profile.updateProfile(event)">
                <div class="form-group">
                    <label>Username</label>
                    <input type="text" id="edit-username" value="${
                      profile.username
                    }" required>
                    <div class="form-error" id="username-edit-error"></div>
                </div>
                <div class="form-group">
                    <label>Email</label>
                    <input type="email" id="edit-email" value="${
                      profile.email || ""
                    }" required>
                    <div class="form-error" id="email-edit-error"></div>
                </div>
                <div class="form-group">
                    <button type="submit" class="btn btn-primary">Save Changes</button>
                    <button type="button" onclick="profile.cancelEdit()" class="btn btn-secondary">Cancel</button>
                </div>
            </form>
        </div>
    `;
}

function setupAvatarUpload() {
  const fileInput = document.getElementById("avatar-upload");
  if (fileInput) {
    fileInput.addEventListener("change", async (event) => {
      const file = event.target.files[0];
      if (file) {
        await handleAvatarUpload(file);
      }
    });
  }
}

async function handleAvatarUpload(file) {
  showMessage("Uploading avatar...", "info");

  try {
    const result = await uploadAvatar(state.user.id, file);

    if (result.success) {
      // Cache-buster to force browser to load new image
      const AvatarUrl = result.data.avatar_url;

      // Update header avatar
      if (state.user) {
        state.user.avatar = AvatarUrl;
        updateNavigation();
      }

      if (state.currentProfile) {
        state.currentProfile.avatar = AvatarUrl;
      }

      updateAvatarDisplay(AvatarUrl);
      updateAvatarButtons(true);

      showMessage("Avatar updated successfully!", "success");
    } else {
      showMessage(result.error, "error");
    }
  } catch (error) {
    showMessage("Failed to upload avatar: " + error.message, "error");
  } finally {
    document.getElementById("avatar-upload").value = "";
  }

  // Clear file input
}

async function loadUserPosts(userId, page = 1) {
  try {
    const response = await apiRequest(
      `/users/${userId}/posts?page=${page}&limit=10`
    );
    if (response.success) {
      renderUserPosts(response.data);
    }
  } catch (error) {
    document.getElementById(
      "profile-posts"
    ).innerHTML = `<p class="error">Failed to load posts: ${error.message}</p>`;
  }
}

async function loadUserComments(userId, page = 1) {
  try {
    const response = await apiRequest(
      `/users/${userId}/comments?page=${page}&limit=10`
    );
    if (response.success) {
      renderUserComments(response.data);
    }
  } catch (error) {
    document.getElementById(
      "profile-comments"
    ).innerHTML = `<p class="error">Failed to load comments: ${error.message}</p>`;
  }
}

function renderUserPosts(data) {
  const container = document.getElementById("profile-posts");

  if (!data.posts || data.posts.length === 0) {
    container.innerHTML = "<p>No posts yet.</p>";
    return;
  }

  const categories = {
    1: "general",
    2: "tech",
    3: "programming",
    4: "web-dev",
    5: "mobile",
    6: "career",
    7: "help",
    8: "showcase",
  };

  container.innerHTML = `
        <div class="user-content-list">
            ${data.posts
              .map(
                (post) => `
                <div class="user-content-item" onclick="app.showPost(${
                  post.id
                })">
                    <h4>${escapeHtml(post.title)}</h4>
                    <p>${escapeHtml(post.content)}</p>
                    <div class="item-meta">
                        <span>${
                          categories[Number(post.category_id)] || "Unknown"
                        }</span>
                        <span>•</span>
                        <span>${formatDate(post.created_at)}</span>
                        <span>•</span>
                        <span>${post.vote_score} points</span>
                        <span>•</span>
                        <span>${post.comment_count} comments</span>
                    </div>
                </div>
            `
              )
              .join("")}
        </div>
        ${renderProfilePagination(data, "posts")}
    `;
}

function renderUserComments(data) {
  const container = document.getElementById("profile-comments");

  if (!data.comments || data.comments.length === 0) {
    container.innerHTML = "<p>No comments yet.</p>";
    return;
  }

  container.innerHTML = `
        <div class="user-content-list">
            ${data.comments
              .map(
                (comment) => `
                <div class="user-content-item" onclick="app.showPost(${
                  comment.post_id
                })">
                    <h4>Replied to: ${escapeHtml(comment.post_title)}</h4>
                    <p>${escapeHtml(comment.content)}</p>
                    <div class="item-meta">
                        <span>${formatDate(comment.created_at)}</span>
                        <span>•</span>
                        <span>${comment.vote_score} points</span>
                    </div>
                </div>
            `
              )
              .join("")}
        </div>
        ${renderProfilePagination(data, "comments")}
    `;
}

function renderProfilePagination(data, type) {
  if (data.total_pages <= 1) return "";

  let html = '<div class="pagination">';

  if (data.page > 1) {
    html += `<button onclick="profile.loadPage('${type}', ${
      data.page - 1
    })">Previous</button>`;
  }

  for (let i = 1; i <= data.total_pages; i++) {
    const isActive = i === data.page ? "active" : "";
    html += `<button class="${isActive}" onclick="profile.loadPage('${type}', ${i})">${i}</button>`;
  }

  if (data.page < data.total_pages) {
    html += `<button onclick="profile.loadPage('${type}', ${
      data.page + 1
    })">Next</button>`;
  }

  html += "</div>";
  return html;
}

function formatAccountAge(createdAt) {
  const created = new Date(createdAt);
  const now = new Date();
  const diffTime = Math.abs(now - created);
  const diffDays = Math.ceil(diffTime / (1000 * 60 * 60 * 24));

  if (diffDays < 30) {
    return `${diffDays}d`;
  } else if (diffDays < 365) {
    return `${Math.floor(diffDays / 30)}mo`;
  } else {
    return `${Math.floor(diffDays / 365)}y`;
  }
}

function updateNavigation() {
  const avatar = document.getElementById("user-avatar");
  if (!avatar || !state.user) return;

  if (avatar && state.user) {
    if (state.user.avatar) {
      avatar.style.backgroundImage = `url(${state.user.avatar})`;
      avatar.style.backgroundSize = "cover";
      avatar.style.backgroundPosition = "center";
      avatar.textContent = "";
    } else {
      avatar.style.backgroundImage = "";
      avatar.textContent = state.user.username.charAt(0).toUpperCase();
    }
    // Update username in header
    const usernameSpan = document.getElementById("username");
    if (usernameSpan) {
      usernameSpan.textContent = state.user.username;
    }
  }
}

// Helper function to escape HTML
function escapeHtml(text) {
  const div = document.createElement("div");
  div.textContent = text;
  return div.innerHTML;
}

// Helper function to format dates
function formatDate(dateString) {
  const date = new Date(dateString);
  return (
    date.toLocaleDateString() +
    " " +
    date.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })
  );
}

// Profile management object for global access
export const profile = {
  async showProfile(userId) {
    const result = await loadUserProfile(userId);
    if (result.success) {
      app.showSection("profile-section");
      renderProfile();
    } else {
      showMessage(result.error, "error");
    }
  },

  showEditForm() {
    renderEditProfile();
  },

  cancelEdit() {
    renderProfile();
  },

  async updateProfile(event) {
    event.preventDefault();

    const username = document.getElementById("edit-username").value.trim();
    const email = document.getElementById("edit-email").value.trim();

    const updates = {};

    // Only include changed fields
    if (username !== state.currentProfile.username) {
      updates.username = username;
    }

    if (email !== state.currentProfile.email) {
      updates.email = email;
    }

    if (Object.keys(updates).length === 0) {
      showMessage("No changes to save", "info");
      return;
    }

    const result = await updateProfile(state.user.id, updates);

    if (result.success) {
      renderProfile();
      showMessage("Profile updated successfully!", "success");
    } else {
      showMessage(result.error, "error");
    }
  },

  async deleteAvatar() {
    if (!confirm("Are you sure you want to remove your avatar?")) {
      return;
    }

    const result = await deleteAvatar(state.user.id);

    if (result.success) {
      if (state.user) {
        state.user.avatar = null;
        updateNavigation();
      }

      if (state.currentProfile) {
        state.currentProfile.avatar = null;
      }

      updateAvatarDisplay(null);
      updateAvatarButtons(false);

      showMessage("Avatar removed successfully!", "success");
    } else {
      showMessage(result.error, "error");
    }
  },

  showTab(tabName) {
    // Update tab buttons
    document.querySelectorAll(".tab-btn").forEach((btn) => {
      btn.classList.remove("active");
    });
    event.target.classList.add("active");

    // Update tab content
    document.querySelectorAll(".tab-pane").forEach((pane) => {
      pane.classList.remove("active");
    });
    document.getElementById(`profile-${tabName}`).classList.add("active");

    // Load content based on tab
    if (tabName === "posts") {
      loadUserPosts(state.currentProfile.id);
    } else if (tabName === "comments") {
      loadUserComments(state.currentProfile.id);
    }
  },

  loadPage(type, page) {
    if (type === "posts") {
      loadUserPosts(state.currentProfile.id, page);
    } else if (type === "comments") {
      loadUserComments(state.currentProfile.id, page);
    }
  },
};
