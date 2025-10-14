import { analytics, notify, truncate } from "@fider/services"
import { getCurrentAccessToken, refreshAccessToken } from "@fider/services/auth"

export interface ErrorItem {
  field?: string
  message: string
}

export interface Failure {
  errors?: ErrorItem[]
}

export interface Result<T = void> {
  ok: boolean
  data: T
  error?: Failure
}

/**
 * Get API base URL from environment or runtime config
 * Defaults to empty string for same-origin requests
 */
const getApiBaseUrl = (): string => {
  // Check for Vite environment variable
  if (typeof import.meta !== "undefined" && import.meta.env && import.meta.env.VITE_API_BASE_URL) {
    return import.meta.env.VITE_API_BASE_URL as string
  }
  
  // Fallback to empty string for same-origin (monorepo compatibility)
  return ""
}

/**
 * Get tenant ID from current hostname
 * Extracts subdomain for multi-tenant routing
 */
const getTenantFromHost = (): string => {
  const hostname = window.location.hostname
  
  // Extract subdomain (e.g., "demo.app.fider.com" -> "demo")
  const parts = hostname.split(".")
  if (parts.length > 2) {
    return parts[0]
  }
  
  // Return hostname for custom domains or localhost
  return hostname
}

async function toResult<T>(response: Response): Promise<Result<T>> {
  const body = await response.json()

  if (response.status < 400) {
    return {
      ok: true,
      data: body as T,
    }
  }

  if (response.status === 500) {
    notify.error("An unexpected error occurred while processing your request.")
  } else if (response.status === 401) {
    notify.error("You need to be authenticated to perform this operation.")
  } else if (response.status === 403) {
    notify.error("You are not authorized to perform this operation.")
  }

  return {
    ok: false,
    data: body as T,
    error: {
      errors: body.errors,
    },
  }
}
let isRetryingRequest = false

async function request<T>(url: string, method: "GET" | "POST" | "PUT" | "DELETE", body?: any, isRetry = false): Promise<Result<T>> {
  // Construct full URL with API base URL prefix
  const apiBaseUrl = getApiBaseUrl()
  const fullUrl = `${apiBaseUrl}${url}`
  
  const headers: [string, string][] = [
    ["Accept", "application/json"],
    ["Content-Type", "application/json"],
  ]
  
  // Add Authorization header with Bearer token if available
  const accessToken = getCurrentAccessToken()
  if (accessToken) {
    headers.push(["Authorization", `Bearer ${accessToken}`])
  }
  
  // Add X-Tenant-ID header for cross-origin tenant identification
  const tenantId = getTenantFromHost()
  if (tenantId) {
    headers.push(["X-Tenant-ID", tenantId])
  }
  
  try {
    const response = await fetch(fullUrl, {
      method,
      headers,
      body: body !== undefined ? JSON.stringify(body) : undefined,
      credentials: "include", // Changed from "same-origin" for cross-origin cookie support
    })
    
    // Handle 401 Unauthorized - attempt token refresh and retry once
    if (response.status === 401 && !isRetry && !isRetryingRequest) {
      isRetryingRequest = true
      
      try {
        // Attempt to refresh access token
        const newToken = await refreshAccessToken()
        
        if (newToken) {
          // Token refresh successful - retry original request with new token
          isRetryingRequest = false
          return await request<T>(url, method, body, true)
        }
      } catch (refreshError) {
        console.error("Token refresh failed:", refreshError)
      } finally {
        isRetryingRequest = false
      }
    }
    
    return await toResult<T>(response)
  } catch (err) {
    const truncatedBody = truncate(body ? JSON.stringify(body) : "<empty>", 1000)
    throw new Error(`Failed to ${method} ${fullUrl} with body '${truncatedBody}'`)
  }
}

export const http = {
  get: async <T = void>(url: string): Promise<Result<T>> => {
    return await request<T>(url, "GET")
  },
  post: async <T = void>(url: string, body?: any): Promise<Result<T>> => {
    return await request<T>(url, "POST", body)
  },
  put: async <T = void>(url: string, body?: any): Promise<Result<T>> => {
    return await request<T>(url, "PUT", body)
  },
  delete: async <T = void>(url: string, body?: any): Promise<Result<T>> => {
    return await request<T>(url, "DELETE", body)
  },
  event:
    (category: string, action: string) =>
    <T>(result: Result<T>): Result<T> => {
      if (result && result.ok) {
        analytics.event(category, action)
      }
      return result
    },
}
