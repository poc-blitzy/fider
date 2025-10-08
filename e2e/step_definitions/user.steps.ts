import { Given } from "@cucumber/cucumber"
import { FiderWorld } from "../world.js"
import { getLatestLinkSentTo, isAuthenticated, isAuthenticatedAsUser } from "./fns.js"

/**
 * Sign-in step for E2E testing with cross-origin authentication support.
 * 
 * Cross-Origin Authentication Flow:
 * - This step tests the UI-based magic link authentication flow
 * - When the activation link is visited (line 21), the backend authenticates the user
 *   and issues JWT tokens (access token + refresh token cookie)
 * - The frontend SPA automatically receives and manages these tokens internally
 * - JWT access tokens are transmitted via Authorization: Bearer header for subsequent API calls
 * - Refresh tokens are stored as HttpOnly cookies for secure token renewal
 * 
 * Token Access for API Testing:
 * - If needed for direct API testing, JWT access tokens can be accessed via world.accessToken
 * - This step primarily tests UI flows and doesn't directly interact with /api/v1/auth/* endpoints
 * - The authentication state is managed transparently by the frontend application
 */
Given("I sign in as {string}", async function (this: FiderWorld, userName: string) {
  if (await isAuthenticatedAsUser(this.page, userName)) {
    return
  }

  // Sign out if already authenticated as a different user
  // This triggers the logout endpoint which clears JWT tokens and refresh cookies
  if (await isAuthenticated(this.page)) {
    await this.page.click(".c-menu-user .c-dropdown__handle")
    await this.page.click("a[href='/signout']")
  }

  const userEmail = `${userName}-${this.tenantName}@fider.io`
  await this.page.click(".c-menu .uppercase.text-sm")
  await this.page.type(".c-signin-control #input-email", userEmail)
  await this.page.click(".c-signin-control .c-button--primary")

  // Navigate to the magic link activation URL
  // Cross-origin flow: Backend authenticates user and issues JWT tokens upon activation
  // Frontend automatically stores access token and receives refresh token cookie
  const activationLink = await getLatestLinkSentTo(userEmail)
  await this.page.goto(activationLink)
})
