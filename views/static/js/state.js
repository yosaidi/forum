// Application state
export const state = {
  user: null,
  posts: [],
  currentPost: null,
  categories: [],
  comments: [],
  currentPage: 1,
  currentCategory: null,
  currentSort: "newest",
};

// State update helpers
export function updateUser(user) {
  state.user = user;
}

export function updatePosts(posts) {
  state.posts = posts;
}

export function updateCurrentPost(post) {
  state.currentPost = post;
}

export function updateComments(comments) {
  state.comments = comments;
}

export function updateCategories(categories) {
  state.categories = categories;
}
