import { refresh as refreshAPI } from "@fider/services/actions/auth"

/**
 * Local storage key for access token
 */
const ACCESS_TOKEN_KEY = "fider_access_token"

/**
 * Flag to prevent multiple simultaneous refresh attempts
 */
let isRefreshing = false

/**
 * Queue of callbacks waiting for token refresh
 */
let refreshSubscribers: Array<(token: string | null) => void> = []

/**
 * Get current access token from storage
 * 
 * @returns Access token string or null if not found
 */
export const getCurrentAccessToken = (): string | null => {
  try {
    return localStorage.getItem(ACCESS_TOKEN_KEY)
  } catch (error) {
    console.error("Failed to read access token from storage:", error)
    return null
  }
}

/**
 * Store access token in local storage
 * 
 * @param token - JWT access token string
 */
export const setAccessToken = (token: string): void => {
  try {
    localStorage.setItem(ACCESS_TOKEN_KEY, token)
  } catch (error) {
    console.error("Failed to save access token to storage:", error)
  }
}

/**
 * Clear access token from storage
 * Call this during logout
 */
export const clearAccessToken = (): void => {
  try {
    localStorage.removeItem(ACCESS_TOKEN_KEY)
  } catch (error) {
    console.error("Failed to clear access token from storage:", error)
  }
}

/**
 * Subscribe to token refresh completion
 * Used internally to queue requests waiting for new token
 * 
 * @param callback - Function to call with new token (or null if refresh failed)
 */
const subscribeTokenRefresh = (callback: (token: string | null) => void): void => {
  refreshSubscribers.push(callback)
}

/**
 * Notify all subscribers with new token
 * 
 * @param token - New access token or null if refresh failed
 */
const notifyTokenRefresh = (token: string | null): void => {
  refreshSubscribers.forEach((callback) => callback(token))
  refreshSubscribers = []
}

/**
 * Refresh access token using HttpOnly refresh cookie
 * Prevents multiple simultaneous refresh attempts
 * Queues requests if refresh is already in progress
 * 
 * @returns Promise<string | null> - New access token or null if refresh failed
 */
export const refreshAccessToken = async (): Promise<string | null> => {
  // If already refreshing, wait for the result
  if (isRefreshing) {
    return new Promise<string | null>((resolve) => {
      subscribeTokenRefresh((token) => {
        resolve(token)
      })
    })
  }

  isRefreshing = true

  try {
    const result = await refreshAPI()

    if (result.ok && result.data) {
      const newToken = result.data.accessToken
      setAccessToken(newToken)
      notifyTokenRefresh(newToken)
      return newToken
    } else {
      // Refresh failed - clear token and notify subscribers
      clearAccessToken()
      notifyTokenRefresh(null)
      return null
    }
  } catch (error) {
    console.error("Token refresh failed:", error)
    clearAccessToken()
    notifyTokenRefresh(null)
    return null
  } finally {
    isRefreshing = false
  }
}

/**
 * Check if user is authenticated (has valid access token)
 * Note: This only checks if token exists, not if it's expired
 * Server will validate token on each request
 * 
 * @returns boolean - True if access token exists
 */
export const isAuthenticated = (): boolean => {
  return getCurrentAccessToken() !== null
}
