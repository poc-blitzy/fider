/**
 * Build-time configuration injected by webpack DefinePlugin for cross-origin SPA mode.
 *
 * This global constant is defined at compile time via webpack.config.js:
 *   new webpack.DefinePlugin({
 *     '__FIDER_CONFIG__': JSON.stringify({ apiHost: process.env.FIDER_PUBLIC_API_BASE_URL || '' })
 *   })
 *
 * - apiHost: The base URL of the Fider Backend API server (e.g., "https://api.example.com")
 *            Empty string for same-origin deployments (backward compatible)
 */
declare const __FIDER_CONFIG__: {
  /** Base URL of the backend API. Empty string for same-origin mode. */
  apiHost: string
}
