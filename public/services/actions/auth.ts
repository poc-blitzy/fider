import { http, Result } from "@fider/services"
import { CurrentUser } from "@fider/models"

/**
 * Login request payload
 */
export interface LoginRequest {
  email: string
  password?: string // Optional for magic link flow
}

/**
 * Login response containing access token and user data
 */
export interface LoginResponse {
  accessToken: string
  user: CurrentUser
}

/**
 * Refresh response containing new access token
 */
export interface RefreshResponse {
  accessToken: string
}

/**
 * Authenticate user with email and optional password
 * Returns access token and user data
 * Sets HttpOnly refresh cookie automatically
 * 
 * @param request - Login credentials (email and optional password)
 * @returns Promise<Result<LoginResponse>>
 */
export const login = async (request: LoginRequest): Promise<Result<LoginResponse>> => {
  return await http.post<LoginResponse>("/api/v1/auth/login", request)
}

/**
 * Refresh access token using HttpOnly refresh cookie
 * Returns new access token
 * Rotates refresh cookie automatically
 * 
 * @returns Promise<Result<RefreshResponse>>
 */
export const refresh = async (): Promise<Result<RefreshResponse>> => {
  return await http.post<RefreshResponse>("/api/v1/auth/refresh")
}

/**
 * Logout user and clear authentication state
 * Clears HttpOnly refresh cookie
 * Client must also clear access token from storage
 * 
 * @returns Promise<Result<void>>
 */
export const logout = async (): Promise<Result<void>> => {
  return await http.post<void>("/api/v1/auth/logout")
}
