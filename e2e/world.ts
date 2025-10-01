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
}
