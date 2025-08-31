import { app } from "./app.js";
import { state } from "./state.js";

// Initialize the application when DOM is loaded
document.addEventListener("DOMContentLoaded", () => {
  app.init();
});

// Make state and profile globally accessible
window.state = state;
window.profile = profile;

// Handle keyboard shortcuts
document.addEventListener("keydown", (e) => {
  if (e.ctrlKey || e.metaKey) {
    switch (e.key) {
      case "k":
        e.preventDefault();
        document.querySelector("#login-username, #register-username")?.focus();
        break;
      case "n":
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
