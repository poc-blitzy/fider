/**
 * Cucumber.js Configuration
 * 
 * This configuration defines test execution profiles for the Fider e2e test suite.
 * Features are organized by type (server/ui) and can be selectively run using tags.
 * 
 * Profiles:
 * - default: Runs all feature tests
 * - cors: Runs CORS-specific tests (tagged with @cors)
 * - jwt: Runs JWT authentication tests (tagged with @jwt)
 * - api: Runs all server-side API tests
 * - ci: Optimized profile for CI/CD environments
 */

const common = {
  // TypeScript support via ts-node with path mapping support
  requireModule: ['ts-node/register', 'tsconfig-paths/register'],
  
  // Step definition files
  require: ['e2e/**/*.steps.ts'],
  
  // Initialization hooks
  requireFile: ['e2e/_init_.ts'],
  
  // Output formatting
  format: [
    'progress-bar',
    'html:e2e-results/cucumber-report.html',
    'json:e2e-results/cucumber-report.json'
  ],
  
  // Fail fast on first error
  failFast: false,
  
  // Strict mode - fail on undefined or pending steps
  strict: true,
  
  // Publish results to Cucumber Reports
  publish: false,
  
  // Retry failed scenarios
  retry: 0
};

module.exports = {
  // Default profile - runs all tests
  default: {
    ...common,
    paths: ['e2e/features/**/*.feature'],
    format: [
      'progress-bar',
      'html:e2e-results/cucumber-report.html',
      'json:e2e-results/cucumber-report.json',
      '@cucumber/pretty-formatter'
    ]
  },
  
  // CORS profile - runs tests tagged with @cors
  cors: {
    ...common,
    paths: ['e2e/features/**/*.feature'],
    tags: '@cors',
    format: [
      'progress-bar',
      'html:e2e-results/cors-report.html',
      'json:e2e-results/cors-report.json'
    ]
  },
  
  // JWT profile - runs tests tagged with @jwt
  jwt: {
    ...common,
    paths: ['e2e/features/**/*.feature'],
    tags: '@jwt',
    format: [
      'progress-bar',
      'html:e2e-results/jwt-report.html',
      'json:e2e-results/jwt-report.json'
    ]
  },
  
  // API profile - runs all server-side tests
  api: {
    ...common,
    paths: ['e2e/features/server/**/*.feature'],
    format: [
      'progress-bar',
      'html:e2e-results/api-report.html',
      'json:e2e-results/api-report.json'
    ]
  },
  
  // CI profile - optimized for CI/CD environments
  ci: {
    ...common,
    paths: ['e2e/features/**/*.feature'],
    // Run tests in parallel
    parallel: 2,
    // Format for CI/CD systems
    format: [
      'progress',
      'json:e2e-results/cucumber-report.json',
      'junit:e2e-results/junit-report.xml'
    ],
    // Fail fast in CI
    failFast: true,
    // Retry flaky tests once
    retry: 1,
    // Don't publish to Cucumber Reports in CI
    publish: false
  }
};
