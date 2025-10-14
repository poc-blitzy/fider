import { useState, useEffect, useCallback } from "react"
import { CurrentUser } from "@fider/models"
import {
  getCurrentAccessToken,
  setAccessToken,
  clearAccessToken,
  refreshAccessToken,
  isAuthenticated,
} from "@fider/services/auth"
import { login as loginAPI, logout as logoutAPI } from "@fider/services/actions/auth"

export interface UseAuthReturn {
  /**
   * Current authenticated user (null if not authenticated)
   */
  user: CurrentUser | null

  /**
   * Whether user is currently authenticated
   */
  isAuthenticated: boolean

  /**
   * Whether an authentication operation is in progress
   */
  isLoading: boolean

  /**
   * Authenticate user with email and optional password
   * 
   * @param email - User email address
   * @param password - User password (optional for magic link flow)
   * @returns Promise<boolean> - True if login successful
   */
  login: (email: string, password?: string) => Promise<boolean>

  /**
   * Logout user and clear authentication state
   * 
   * @returns Promise<void>
   */
  logout: () => Promise<void>

  /**
   * Manually refresh access token
   * Automatically called on 401 responses
   * 
   * @returns Promise<boolean> - True if refresh successful
   */
  refresh: () => Promise<boolean>

  /**
   * Get current access token
   * 
   * @returns string | null - Access token or null
   */
  getAccessToken: () => string | null
}

/**
 * React hook for authentication state and operations
 * Provides login, logout, and token refresh functionality
 * 
 * @returns UseAuthReturn - Authentication state and methods
 */
export const useAuth = (): UseAuthReturn => {
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [isLoading, setIsLoading] = useState<boolean>(false)
  const [authenticated, setAuthenticated] = useState<boolean>(isAuthenticated())

  /**
   * Login user with credentials
   */
  const login = useCallback(async (email: string, password?: string): Promise<boolean> => {
    setIsLoading(true)
    try {
      const result = await loginAPI({ email, password })
      
      if (result.ok && result.data) {
        // Store access token
        setAccessToken(result.data.accessToken)
        
        // Update user state
        setUser(result.data.user)
        setAuthenticated(true)
        
        return true
      } else {
        // Login failed
        setUser(null)
        setAuthenticated(false)
        return false
      }
    } catch (error) {
      console.error("Login failed:", error)
      setUser(null)
      setAuthenticated(false)
      return false
    } finally {
      setIsLoading(false)
    }
  }, [])

  /**
   * Logout user and clear state
   */
  const logout = useCallback(async (): Promise<void> => {
    setIsLoading(true)
    try {
      // Call backend logout to clear refresh cookie
      await logoutAPI()
    } catch (error) {
      console.error("Logout request failed:", error)
    } finally {
      // Always clear local state regardless of API result
      clearAccessToken()
      setUser(null)
      setAuthenticated(false)
      setIsLoading(false)
    }
  }, [])

  /**
   * Refresh access token
   */
  const refresh = useCallback(async (): Promise<boolean> => {
    setIsLoading(true)
    try {
      const newToken = await refreshAccessToken()
      
      if (newToken) {
        setAuthenticated(true)
        return true
      } else {
        // Refresh failed - clear authentication state
        setUser(null)
        setAuthenticated(false)
        return false
      }
    } catch (error) {
      console.error("Token refresh failed:", error)
      setUser(null)
      setAuthenticated(false)
      return false
    } finally {
      setIsLoading(false)
    }
  }, [])

  /**
   * Get current access token
   */
  const getAccessToken = useCallback((): string | null => {
    return getCurrentAccessToken()
  }, [])

  /**
   * Initialize authentication state on mount
   */
  useEffect(() => {
    const token = getCurrentAccessToken()
    setAuthenticated(token !== null)
    
    // Note: We don't automatically fetch user data here
    // The user data will be set after successful login
    // or can be fetched separately if needed
  }, [])

  return {
    user,
    isAuthenticated: authenticated,
    isLoading,
    login,
    logout,
    refresh,
    getAccessToken,
  }
}
