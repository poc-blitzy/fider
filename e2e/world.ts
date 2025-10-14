import { World as CucumberWorld } from "@cucumber/cucumber"
import { Page } from "@playwright/test"

export interface FiderWorld extends CucumberWorld {
  tenantName: string
  page: Page
  log: (msg: string) => void
  // Cross-origin testing properties for separated frontend/backend repositories
  frontendUrl?: string
  backendUrl?: string
  // JWT authentication tokens (null = not authenticated, string = valid token)
  accessToken?: string | null
  refreshToken?: string | null
  // Authentication test credentials for JWT login flows
  testCredentials?: {
    email: string
    password: string
  }
  // API response tracking for authentication and HTTP request testing
  lastResponse?: Response
  lastResponseStatus?: number
  lastResponseBody?: any
  lastResponseJson?: any // Parsed JSON from lastResponseBody for easier assertion access
  lastResponseHeaders?: Headers
  // Token expiration testing flag for simulating expired token scenarios
  tokenExpirationTestMode?: boolean
}
