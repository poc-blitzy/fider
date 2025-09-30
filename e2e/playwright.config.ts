import { defineConfig } from '@playwright/test';

/**
 * Playwright Configuration for Cross-Origin E2E Testing
 * 
 * This configuration supports testing the Fider application across separate
 * frontend and backend origins, validating CORS functionality, token-based
 * authentication, and cross-origin API communication.
 * 
 * Environment Variables:
 * - FRONTEND_URL: Frontend SPA origin (default: http://localhost:5173)
 * - BACKEND_URL: Backend API origin (default: http://localhost:8080)
 * - CI: Set to 'true' in CI environments for optimized settings
 */

// Read environment variables with fallback defaults for local development
const FRONTEND_URL = process.env.FRONTEND_URL || 'http://localhost:5173';
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
    // Base URL for frontend navigation (used with page.goto('/path'))
    baseURL: FRONTEND_URL,

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
  // Automatically starts frontend and backend servers before running tests
  webServer: [
    {
      // Frontend development server (Vite)
      command: 'npm run dev',
      url: FRONTEND_URL,
      timeout: 120 * 1000, // 2 minutes for initial build
      reuseExistingServer: !IS_CI,
      stdout: 'pipe',
      stderr: 'pipe',
      env: {
        VITE_API_BASE_URL: BACKEND_URL,
      },
    },
    {
      // Backend API server (Go)
      command: 'make start',
      url: `${BACKEND_URL}/api/v1/health`,
      timeout: 120 * 1000, // 2 minutes for compilation
      reuseExistingServer: !IS_CI,
      stdout: 'pipe',
      stderr: 'pipe',
      env: {
        ALLOWED_ORIGINS: FRONTEND_URL,
        PORT: new URL(BACKEND_URL).port || '8080',
      },
    },
  ],
});
