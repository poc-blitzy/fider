/* eslint-disable @typescript-eslint/no-var-requires */
require("isomorphic-fetch")
process.env.NODE_TLS_REJECT_UNAUTHORIZED = "0"

// Cross-origin testing support: If FRONTEND_URL or BACKEND_URL is set,
// both must be provided for proper cross-origin test execution
if (process.env.FRONTEND_URL || process.env.BACKEND_URL) {
  if (!process.env.FRONTEND_URL || !process.env.BACKEND_URL) {
    throw new Error(
      "Cross-origin testing requires both FRONTEND_URL and BACKEND_URL environment variables to be set.\n" +
        "Example: FRONTEND_URL=https://app.example.com BACKEND_URL=https://api.example.com npm run test:e2e"
    )
  }
  console.log("Cross-origin testing enabled:")
  console.log(`  FRONTEND_URL: ${process.env.FRONTEND_URL}`)
  console.log(`  BACKEND_URL: ${process.env.BACKEND_URL}`)
}

// Enable debug logging for CORS-related issues by setting DEBUG environment variable
// Example: DEBUG=e2e:* npm run test:e2e
if (process.env.DEBUG) {
  console.log(`Debug logging enabled: ${process.env.DEBUG}`)
}

require("ts-node").register({
  transpileOnly: true,
  compilerOptions: {
    target: "es6",
    strict: true,
    module: "commonjs",
  },
})
