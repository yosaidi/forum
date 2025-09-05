import { API_BASE } from "./config.js";

// API Helper Functions
export async function apiRequest(endpoint, options = {}) {
  const url = `${API_BASE}${endpoint}`;

  const config = {
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    credentials: "include",
    ...options,
  };

  try {
    const response = await fetch(url, config);
    const data = await response.json();

    if (!response.ok) {
      if (data.error) {
        throw new Error(data.error);
      }

      let errorMsg = data.message || "API request failed";

      if (data.data && typeof data.data === "object") {
        const errors = Object.values(data.data).join("\n");
        errorMsg += ":\n" + errors;
      }
      throw new Error(errorMsg);
    }

    return data;
  } catch (error) {
    console.log("API Error:", error);
    throw error;
  }
}
