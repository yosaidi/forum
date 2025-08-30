import { apiRequest } from "./api.js";
import { state, updateUser } from "./state.js";
import { showMessage } from "./ui.js";

// Authentication Functions
export async function checkAuthStatus() {
  try {
    const response = await apiRequest("/auth/me");
    if (response.success) {
      updateUser(response.data);
      updateNavigation();
      return true;
    }
  } catch (error) {
    updateUser(null);
  }
  updateNavigation();
  return false;
}

function updateNavigation() {
  const guestNav = document.getElementById("guest-nav");
  const userNav = document.getElementById("user-nav");
  const username = document.getElementById("username");
  const avatar = document.getElementById("user-avatar");

  if (state.user) {
    guestNav.style.display = "none";
    userNav.style.display = "flex";
    username.textContent = state.user.username;
    avatar.textContent = state.user.username.charAt(0).toUpperCase();
  } else {
    guestNav.style.display = "flex";
    userNav.style.display = "none";
  }
}

export async function login(username, password) {
  try {
    const response = await apiRequest("/auth/login", {
      method: "POST",
      body: JSON.stringify({ username, password }),
    });

    if (response.success) {
      updateUser(response.data.user);
      updateNavigation();
      return { success: true };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
}

export async function register(username, email, password) {
  try {
    const response = await apiRequest("/auth/register", {
      method: "POST",
      body: JSON.stringify({ username, email, password }),
    });

    if (response.success) {
      updateUser(response.data.user);
      updateNavigation();
      return { success: true };
    }
  } catch (error) {
    return { success: false, error: error.message };
  }
}

export async function logout() {
  try {
    await apiRequest("/auth/logout", { method: "POST" });
    updateUser(null);
    updateNavigation();
    return { success: true };
  } catch (error) {
    return { success: false, error: error.message };
  }
}
