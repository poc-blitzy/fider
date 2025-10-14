import { defineConfig } from '@playwright/test';

/**
 * Playwright Configuration for Monolithic E2E Testing
 * 
 * This configuration tests the Fider application in its monolithic architecture
 * where the Go backend serves both the frontend static assets and API routes
 * from a single origin/port.
 * 
 * Environment Variables:
 * - BACKEND_URL: Application origin (default: http://localhost:8080)
 * - FRONTEND_URL: [Optional] If set, overrides baseURL (for future multi-server testing)
 * - CI: Set to 'true' in CI environments for optimized settings
 */

// Read environment variables with fallback defaults for local development
// MONOLITHIC MODE: Both frontend and backend served from the same Go server
const BACKEND_URL = process.env.BACKEND_URL || 'http://localhost:8080';
const IS_CI = process.env.CI === 'true';

export default defineConfig({
  // Test directory containing all E2E test files
  testDir: '.',

  // Global timeout for each test (30 seconds)
  timeout: 30 * 1000,

  // Expect assertion timeout (5 seconds)
  expect: {
    timeout: 5 * 1000,
  },

  // Fail fast in CI, continue locally for debugging
  fullyParallel: !IS_CI,
  
  // Abort test run on first failure in CI
  forbidOnly: IS_CI,
  
  // Retry failed tests in CI for flake detection
  retries: IS_CI ? 2 : 0,
  
  // Limit parallel workers in CI to reduce resource contention
  workers: IS_CI ? 2 : undefined,

  // Reporter configuration: HTML report for local, GitHub Actions reporter for CI
  reporter: IS_CI ? 'github' : 'html',

  // Shared settings for all test projects
  use: {
    // Base URL for monolithic application (frontend and API both served from here)
    baseURL: BACKEND_URL,

    // Ignore HTTPS certificate errors for local development and staging
    // Required for testing self-signed certificates in non-production environments
    ignoreHTTPSErrors: true,

    // Default viewport size for consistent cross-browser rendering
    viewport: { width: 1280, height: 720 },

    // Capture screenshots only on test failure for debugging
    screenshot: 'only-on-failure',

    // Record video only on first retry to capture flaky test behavior
    video: 'retain-on-failure',

    // Trace recording for debugging (on-first-retry captures flakes)
    trace: 'on-first-retry',

    // Action timeout for individual interactions (10 seconds)
    actionTimeout: 10 * 1000,

    // Navigation timeout for page loads (30 seconds)
    navigationTimeout: 30 * 1000,
  },

  // Browser-specific test projects for cross-browser validation
  projects: [
    {
      name: 'chromium',
      use: {
        // Chromium-based browsers (Chrome, Edge)
        ...require('@playwright/test').devices['Desktop Chrome'],
        
        // Cross-origin testing requires proper context isolation
        contextOptions: {
          ignoreHTTPSErrors: true,
        },
      },
    },
    {
      name: 'firefox',
      use: {
        // Firefox browser
        ...require('@playwright/test').devices['Desktop Firefox'],
        
        contextOptions: {
          ignoreHTTPSErrors: true,
        },
      },
    },
    {
      name: 'webkit',
      use: {
        // Safari/WebKit browser
        ...require('@playwright/test').devices['Desktop Safari'],
        
        contextOptions: {
          ignoreHTTPSErrors: true,
        },
      },
    },
  ],

  // Web server configuration for local development
  // Automatically starts backend server before running tests
  // Note: Frontend is not needed for CORS API testing
  webServer: {
    // Backend API server (Go) using test environment
    // Command uses full paths: godotenv from /root/go/bin and fider binary from ./dist
    command: '/root/go/bin/godotenv -f .test.env ./dist/fider',
    
    // Working directory must be project root where .test.env is located
    cwd: process.cwd().endsWith('/e2e') 
      ? process.cwd().replace(/\/e2e$/, '')  // If running from e2e/, go up one level
      : process.cwd(),  // Otherwise assume we're already in project root
    
    url: `${BACKEND_URL}/_health`,
    timeout: 120 * 1000, // 2 minutes for server startup
    reuseExistingServer: !IS_CI,
    stdout: 'pipe',
    stderr: 'pipe',
    // Environment variables are loaded from .test.env via godotenv
    // .test.env already contains: ALLOWED_ORIGINS, ALLOWED_HEADERS, ALLOWED_METHODS
  },
});
