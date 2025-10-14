import { Before, BeforeAll, AfterAll, After } from "@cucumber/cucumber"
import debug from "debug"
import * as playwright from "@playwright/test"
import { getLatestLinkSentTo } from "./step_definitions/fns.js"
import { FiderWorld } from "./world.js"

console.log('[DEBUG] setup.ts is being loaded')

let browser: playwright.Browser
let tenantName: string
type BrowserName = "chromium" | "firefox" | "webkit"

// Initialize URLs from environment variables with fallbacks for local testing
// MONOLITHIC MODE: Frontend and backend both served from the same Go server
// In the current monolithic architecture, the Go backend serves frontend static assets
// Both frontend and backend use the same base URL (localhost:8080 in tests)
const backendUrl = process.env.BACKEND_URL || "http://localhost:8080"
const frontendUrl = process.env.FRONTEND_URL || backendUrl // Use backendUrl as fallback for monolithic mode

console.log('[DEBUG] frontendUrl:', frontendUrl)
console.log('[DEBUG] backendUrl:', backendUrl)

BeforeAll({ timeout: 30 * 1000 }, async function () {
  console.log('[DEBUG] BeforeAll hook executing')
  // Skip browser-based setup for API-only tests (e.g., CORS tests)
  // API-only tests make direct HTTP requests without needing a browser or UI-based tenant creation
  const isApiOnlyTest = process.env.API_ONLY_TESTS === 'true'
  
  if (isApiOnlyTest) {
    console.log('[BeforeAll] Skipping UI setup for API-only tests')
    // Set a default tenant name for API tests (assuming single-tenant or test tenant exists)
    if (!tenantName) {
      tenantName = process.env.TEST_TENANT_NAME || 'apitests'
    }
    return
  }

  const name = (process.env.BROWSER || "chromium") as BrowserName
  browser = await playwright[name].launch({
    headless: true,
    slowMo: 10,
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  })

  if (!tenantName) {
    const now = new Date().getTime()
    tenantName = `feedback${now}`
    await createNewSite()
  }
})

AfterAll(async function () {
  console.log('[DEBUG] AfterAll hook executing')
  // Skip browser cleanup for API-only tests (browser was never launched)
  const isApiOnlyTest = process.env.API_ONLY_TESTS === 'true'
  
  if (isApiOnlyTest) {
    console.log('[AfterAll] Skipping browser cleanup for API-only tests')
    return
  }

  if (browser) {
    await browser.close()
  }
})

Before(async function (this: FiderWorld) {
  console.log('[DEBUG] Before hook executing - START')
  console.log('[DEBUG] Before hook - backendUrl value:', backendUrl)
  console.log('[DEBUG] Before hook - this object keys:', Object.keys(this))
  
  // Skip browser/page setup for API-only tests (making direct HTTP requests)
  const isApiOnlyTest = process.env.API_ONLY_TESTS === 'true'
  
  if (isApiOnlyTest) {
    console.log('[Before] Skipping browser/page setup for API-only tests')
    // Initialize test context without browser/page
    this.tenantName = tenantName
    this.log = debug("e2e")
    
    // Initialize URLs for cross-origin testing
    this.frontendUrl = frontendUrl
    this.backendUrl = backendUrl
    
    console.log('[DEBUG] Before hook - API-only path - this.backendUrl set to:', this.backendUrl)
    
    // Initialize JWT tokens (will be set by auth steps)
    this.accessToken = null
    this.refreshToken = null
    return
  }

  const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
    ignoreHTTPSErrors: true,
  })

  this.page = await context.newPage()
  this.tenantName = tenantName
  this.log = debug("e2e")
  
  // Initialize URLs for cross-origin testing
  this.frontendUrl = frontendUrl
  this.backendUrl = backendUrl
  
  console.log('[DEBUG] Before hook - normal path - this.backendUrl set to:', this.backendUrl)
  
  // Initialize JWT tokens (will be set by auth steps)
  this.accessToken = null
  this.refreshToken = null
  
  console.log('[DEBUG] Before hook executing - END')
})

After(async function (this: FiderWorld) {
  console.log('[DEBUG] After hook executing')
  // Skip page cleanup for API-only tests
  const isApiOnlyTest = process.env.API_ONLY_TESTS === 'true'
  
  if (isApiOnlyTest || !this.page) {
    console.log('[After] Skipping page cleanup for API-only tests or no page available')
    return
  }
  
  await this.page.close()
})

async function createNewSite() {
  const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
    ignoreHTTPSErrors: true,
  })
  const page = await context.newPage()

  const adminEmail = `admin-${tenantName}@fider.io`
  //Create site
  await page.goto(`${frontendUrl}/signup`)
  await page.type("#input-name", "admin")
  await page.type("#input-email", adminEmail)
  await page.type("#input-tenantName", tenantName)
  await page.type("#input-subdomain", tenantName)
  await page.check("#input-legalAgreement")
  await page.click(".c-button--primary")

  //Activate site
  const activationLink = await getLatestLinkSentTo(adminEmail)
  await page.goto(activationLink)
  await page.close()
}
