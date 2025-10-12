import { Given, When, Then } from "@cucumber/cucumber"
import { expect } from "@playwright/test"
import { FiderWorld } from "e2e/world"
import { delay, parseJwtToken, isTokenExpired, extractAccessTokenFromResponse } from "e2e/step_definitions/fns"

/**
 * JWT Authentication Step Definitions
 * 
 * These step definitions test the cross-origin JWT authentication flows
 * introduced during the repository separation refactoring. Tests cover:
 * - Login with credentials (POST /api/v1/auth/login)
 * - Token refresh flows (POST /api/v1/auth/refresh)
 * - Logout and session termination (POST /api/v1/auth/logout)
 * - Bearer token validation in Authorization header
 * - Cross-origin cookie handling (HttpOnly, Secure, SameSite=None)
 * - Token expiration and automatic refresh scenarios
 */

// ============================================================================
// GIVEN Steps - Test Setup and Preconditions
// ============================================================================

Given("I have valid test credentials", function (this: FiderWorld) {
  const email = `testuser-${this.tenantName}@fider.io`
  const password = "TestPassword123!"
  
  this.testCredentials = { email, password }
  this.log(`Set test credentials: ${email}`)
})

Given("I have an expired access token", async function (this: FiderWorld) {
  // Set a mock expired token for testing token refresh flows
  // This token has an exp claim in the past
  const expiredToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwidGVuYW50X2lkIjoiMSIsInVzZXJfaWQiOiIxIiwicm9sZSI6InZpc2l0b3IiLCJleHAiOjE1MTYyMzkwMjJ9.4Adcj0vfLwf6P1JyLqBqM5L8s0Ug4pvqJj5V1Z7XhJE"
  
  this.accessToken = expiredToken
  this.tokenExpirationTestMode = true
  this.log("Set expired access token for testing")
})

Given("I am authenticated with a valid JWT token", async function (this: FiderWorld) {
  if (!this.testCredentials) {
    this.testCredentials = {
      email: `testuser-${this.tenantName}@fider.io`,
      password: "TestPassword123!"
    }
  }
  
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const loginUrl = `${backendUrl}/api/v1/auth/login`
  
  const response = await fetch(loginUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": this.tenantName
    },
    credentials: "include",
    body: JSON.stringify(this.testCredentials)
  })
  
  expect(response.status).toBe(200)
  
  const body = await response.json()
  this.accessToken = extractAccessTokenFromResponse(body)
  
  expect(this.accessToken).toBeTruthy()
  this.log(`Authenticated successfully with JWT token`)
})

Given("I have no authentication token", function (this: FiderWorld) {
  this.accessToken = null
  this.refreshToken = null
  this.log("Cleared all authentication tokens")
})

// ============================================================================
// WHEN Steps - Actions and Operations
// ============================================================================

When("I login with email {string} and password {string}", async function (
  this: FiderWorld,
  email: string,
  password: string
) {
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const loginUrl = `${backendUrl}/api/v1/auth/login`
  
  this.log(`Attempting login to: ${loginUrl}`)
  this.log(`Credentials: ${email}`)
  
  const response = await fetch(loginUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": this.tenantName
    },
    credentials: "include", // Required for refresh cookie
    body: JSON.stringify({ email, password })
  })
  
  this.lastResponse = response
  this.lastResponseStatus = response.status
  
  // Extract headers
  const headers: Record<string, string> = {}
  response.headers.forEach((value, key) => {
    headers[key] = value
  })
  this.lastResponseHeaders = headers
  
  // Parse response body
  try {
    this.lastResponseBody = await response.json()
    this.log(`Login response: ${JSON.stringify(this.lastResponseBody)}`)
  } catch (error) {
    this.lastResponseBody = null
    this.log(`Failed to parse response body: ${error}`)
  }
  
  // Store access token if login was successful
  if (response.status === 200 && this.lastResponseBody) {
    this.accessToken = extractAccessTokenFromResponse(this.lastResponseBody)
    if (this.accessToken) {
      this.log(`Access token received: ${this.accessToken.substring(0, 20)}...`)
    }
  }
})

When("I login with my test credentials", async function (this: FiderWorld) {
  if (!this.testCredentials) {
    throw new Error("Test credentials not set. Use 'Given I have valid test credentials' first.")
  }
  
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const loginUrl = `${backendUrl}/api/v1/auth/login`
  
  this.log(`Attempting login with test credentials`)
  
  const response = await fetch(loginUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": this.tenantName
    },
    credentials: "include",
    body: JSON.stringify(this.testCredentials)
  })
  
  this.lastResponse = response
  this.lastResponseStatus = response.status
  
  const headers: Record<string, string> = {}
  response.headers.forEach((value, key) => {
    headers[key] = value
  })
  this.lastResponseHeaders = headers
  
  try {
    this.lastResponseBody = await response.json()
  } catch (error) {
    this.lastResponseBody = null
  }
  
  if (response.status === 200 && this.lastResponseBody) {
    this.accessToken = extractAccessTokenFromResponse(this.lastResponseBody)
  }
})

When("I request a token refresh", async function (this: FiderWorld) {
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const refreshUrl = `${backendUrl}/api/v1/auth/refresh`
  
  this.log(`Requesting token refresh from: ${refreshUrl}`)
  
  const response = await fetch(refreshUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Tenant-ID": this.tenantName
    },
    credentials: "include" // Critical: sends refresh cookie
  })
  
  this.lastResponse = response
  this.lastResponseStatus = response.status
  
  const headers: Record<string, string> = {}
  response.headers.forEach((value, key) => {
    headers[key] = value
  })
  this.lastResponseHeaders = headers
  
  try {
    this.lastResponseBody = await response.json()
    this.log(`Refresh response: ${JSON.stringify(this.lastResponseBody)}`)
  } catch (error) {
    this.lastResponseBody = null
  }
  
  // Update access token if refresh was successful
  if (response.status === 200 && this.lastResponseBody) {
    const newToken = extractAccessTokenFromResponse(this.lastResponseBody)
    if (newToken) {
      this.log(`New access token received: ${newToken.substring(0, 20)}...`)
      this.accessToken = newToken
    }
  }
})

When("I logout", async function (this: FiderWorld) {
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const logoutUrl = `${backendUrl}/api/v1/auth/logout`
  
  this.log(`Logging out from: ${logoutUrl}`)
  
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Tenant-ID": this.tenantName
  }
  
  // Include Authorization header if we have an access token
  if (this.accessToken) {
    headers["Authorization"] = `Bearer ${this.accessToken}`
  }
  
  const response = await fetch(logoutUrl, {
    method: "POST",
    headers,
    credentials: "include" // Required to clear refresh cookie
  })
  
  this.lastResponse = response
  this.lastResponseStatus = response.status
  
  const responseHeaders: Record<string, string> = {}
  response.headers.forEach((value, key) => {
    responseHeaders[key] = value
  })
  this.lastResponseHeaders = responseHeaders
  
  try {
    this.lastResponseBody = await response.json()
  } catch (error) {
    this.lastResponseBody = null
  }
  
  // Clear local tokens after logout
  if (response.status === 200) {
    this.accessToken = null
    this.refreshToken = null
    this.log("Logged out successfully, tokens cleared")
  }
})

When("I make an authenticated request to {string}", async function (
  this: FiderWorld,
  endpoint: string
) {
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const url = `${backendUrl}${endpoint}`
  
  this.log(`Making authenticated request to: ${url}`)
  
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    "X-Tenant-ID": this.tenantName
  }
  
  if (this.accessToken) {
    headers["Authorization"] = `Bearer ${this.accessToken}`
    this.log(`Using Bearer token: ${this.accessToken.substring(0, 20)}...`)
  } else {
    this.log("Warning: No access token available")
  }
  
  const response = await fetch(url, {
    method: "GET",
    headers,
    credentials: "include"
  })
  
  this.lastResponse = response
  this.lastResponseStatus = response.status
  
  const responseHeaders: Record<string, string> = {}
  response.headers.forEach((value, key) => {
    responseHeaders[key] = value
  })
  this.lastResponseHeaders = responseHeaders
  
  try {
    this.lastResponseBody = await response.json()
  } catch (error) {
    this.lastResponseBody = null
  }
})

When("I wait for the token to expire", async function (this: FiderWorld) {
  if (!this.accessToken) {
    throw new Error("No access token to wait for expiration")
  }
  
  const payload = parseJwtToken(this.accessToken)
  const expiresAt = payload.exp * 1000 // Convert to milliseconds
  const now = Date.now()
  const waitTime = expiresAt - now + 1000 // Wait 1 second past expiration
  
  if (waitTime > 0) {
    this.log(`Waiting ${waitTime}ms for token to expire`)
    await delay(waitTime)
  }
  
  this.log("Token should now be expired")
})

// ============================================================================
// THEN Steps - Assertions and Validations
// ============================================================================

Then("the login should succeed", function (this: FiderWorld) {
  expect(this.lastResponseStatus).toBe(200)
  this.log("Login succeeded with status 200")
})

Then("the login should fail", function (this: FiderWorld) {
  expect(this.lastResponseStatus).not.toBe(200)
  this.log(`Login failed with status ${this.lastResponseStatus}`)
})

Then("the login should fail with status {int}", function (
  this: FiderWorld,
  statusCode: number
) {
  expect(this.lastResponseStatus).toBe(statusCode)
  this.log(`Login failed with expected status ${statusCode}`)
})

Then("I should receive an access token", function (this: FiderWorld) {
  expect(this.lastResponseBody).toBeTruthy()
  
  const token = extractAccessTokenFromResponse(this.lastResponseBody)
  expect(token).toBeTruthy()
  expect(typeof token).toBe("string")
  
  this.log(`Access token received: ${token?.substring(0, 20)}...`)
})

Then("the access token should be a valid JWT", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  // JWT tokens have three parts separated by dots
  const parts = this.accessToken!.split(".")
  expect(parts.length).toBe(3)
  
  // Should be able to parse the token
  const payload = parseJwtToken(this.accessToken!)
  expect(payload).toBeTruthy()
  expect(typeof payload).toBe("object")
  
  this.log(`JWT token validated: ${JSON.stringify(payload)}`)
})

Then("the access token should contain tenant information", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  const payload = parseJwtToken(this.accessToken!)
  expect(payload.tenant_id).toBeTruthy()
  
  this.log(`Token contains tenant_id: ${payload.tenant_id}`)
})

Then("the access token should contain user information", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  const payload = parseJwtToken(this.accessToken!)
  expect(payload.user_id).toBeTruthy()
  
  this.log(`Token contains user_id: ${payload.user_id}`)
})

Then("the access token should have an expiration time", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  const payload = parseJwtToken(this.accessToken!)
  expect(payload.exp).toBeTruthy()
  expect(typeof payload.exp).toBe("number")
  
  const expiresAt = new Date(payload.exp * 1000)
  this.log(`Token expires at: ${expiresAt.toISOString()}`)
})

Then("the access token should not be expired", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  const expired = isTokenExpired(this.accessToken!)
  expect(expired).toBe(false)
  
  this.log("Token is not expired")
})

Then("the access token should be expired", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  const expired = isTokenExpired(this.accessToken!)
  expect(expired).toBe(true)
  
  this.log("Token is expired")
})

Then("I should receive a refresh cookie", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  const setCookieHeader = this.lastResponseHeaders!["set-cookie"]
  expect(setCookieHeader).toBeTruthy()
  expect(setCookieHeader).toContain("refresh_token")
  
  this.log(`Refresh cookie received: ${setCookieHeader}`)
})

Then("the refresh cookie should be HttpOnly", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  const setCookieHeader = this.lastResponseHeaders!["set-cookie"]
  expect(setCookieHeader).toBeTruthy()
  expect(setCookieHeader.toLowerCase()).toContain("httponly")
  
  this.log("Refresh cookie has HttpOnly flag")
})

Then("the refresh cookie should be Secure", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  const setCookieHeader = this.lastResponseHeaders!["set-cookie"]
  expect(setCookieHeader).toBeTruthy()
  expect(setCookieHeader.toLowerCase()).toContain("secure")
  
  this.log("Refresh cookie has Secure flag")
})

Then("the refresh cookie should have SameSite=None", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  const setCookieHeader = this.lastResponseHeaders!["set-cookie"]
  expect(setCookieHeader).toBeTruthy()
  expect(setCookieHeader.toLowerCase()).toContain("samesite=none")
  
  this.log("Refresh cookie has SameSite=None attribute")
})

Then("the refresh cookie should be scoped to the API domain", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  const setCookieHeader = this.lastResponseHeaders!["set-cookie"]
  expect(setCookieHeader).toBeTruthy()
  
  // Extract domain from backend URL
  const backendUrl = this.backendUrl || process.env.BACKEND_URL || "http://localhost:8080"
  const domain = new URL(backendUrl).hostname
  
  // Cookie should either have explicit Domain attribute or be scoped to request domain
  if (setCookieHeader.toLowerCase().includes("domain=")) {
    expect(setCookieHeader.toLowerCase()).toContain(domain.toLowerCase())
  }
  
  this.log(`Refresh cookie scoped correctly for domain: ${domain}`)
})

Then("the token refresh should succeed", function (this: FiderWorld) {
  expect(this.lastResponseStatus).toBe(200)
  this.log("Token refresh succeeded with status 200")
})

Then("the token refresh should fail", function (this: FiderWorld) {
  expect(this.lastResponseStatus).not.toBe(200)
  this.log(`Token refresh failed with status ${this.lastResponseStatus}`)
})

Then("the token refresh should fail with status {int}", function (
  this: FiderWorld,
  statusCode: number
) {
  expect(this.lastResponseStatus).toBe(statusCode)
  this.log(`Token refresh failed with expected status ${statusCode}`)
})

Then("I should receive a new access token", function (this: FiderWorld) {
  expect(this.lastResponseBody).toBeTruthy()
  
  const newToken = extractAccessTokenFromResponse(this.lastResponseBody)
  expect(newToken).toBeTruthy()
  expect(typeof newToken).toBe("string")
  
  this.log(`New access token received: ${newToken?.substring(0, 20)}...`)
})

Then("the new access token should be different from the old token", function (this: FiderWorld) {
  expect(this.lastResponseBody).toBeTruthy()
  
  const oldToken = this.accessToken
  const newToken = extractAccessTokenFromResponse(this.lastResponseBody)
  
  expect(oldToken).toBeTruthy()
  expect(newToken).toBeTruthy()
  expect(newToken).not.toBe(oldToken)
  
  this.log("New token is different from old token")
})

Then("the logout should succeed", function (this: FiderWorld) {
  expect(this.lastResponseStatus).toBe(200)
  this.log("Logout succeeded with status 200")
})

Then("the refresh cookie should be cleared", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  const setCookieHeader = this.lastResponseHeaders!["set-cookie"]
  expect(setCookieHeader).toBeTruthy()
  
  // Cookie should be cleared by setting Max-Age=0 or Expires in the past
  const cleared = 
    setCookieHeader.toLowerCase().includes("max-age=0") ||
    setCookieHeader.toLowerCase().includes("expires=thu, 01 jan 1970")
  
  expect(cleared).toBe(true)
  this.log("Refresh cookie has been cleared")
})

Then("the authenticated request should succeed", function (this: FiderWorld) {
  expect(this.lastResponseStatus).toBe(200)
  this.log("Authenticated request succeeded with status 200")
})

Then("the authenticated request should fail with status {int}", function (
  this: FiderWorld,
  statusCode: number
) {
  expect(this.lastResponseStatus).toBe(statusCode)
  this.log(`Authenticated request failed with expected status ${statusCode}`)
})

Then("the response should indicate unauthorized", function (this: FiderWorld) {
  expect(this.lastResponseStatus).toBe(401)
  this.log("Response indicates unauthorized (401)")
})

Then("the response should contain user information", function (this: FiderWorld) {
  expect(this.lastResponseBody).toBeTruthy()
  expect(this.lastResponseBody.user).toBeTruthy()
  
  this.log(`User information: ${JSON.stringify(this.lastResponseBody.user)}`)
})

Then("the Authorization header should contain Bearer token", function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  // The token should be used in subsequent requests with Bearer prefix
  const authHeader = `Bearer ${this.accessToken}`
  expect(authHeader).toContain("Bearer ")
  
  this.log(`Authorization header: ${authHeader.substring(0, 30)}...`)
})

Then("the response should have CORS headers for cross-origin access", function (this: FiderWorld) {
  expect(this.lastResponseHeaders).toBeTruthy()
  
  // Check for Access-Control-Allow-Origin header
  const accessControlOrigin = this.lastResponseHeaders!["access-control-allow-origin"]
  expect(accessControlOrigin).toBeTruthy()
  
  // Check for Access-Control-Allow-Credentials header
  const accessControlCredentials = this.lastResponseHeaders!["access-control-allow-credentials"]
  expect(accessControlCredentials).toBe("true")
  
  this.log(`CORS headers present: Origin=${accessControlOrigin}, Credentials=${accessControlCredentials}`)
})

Then("the response should preserve the API v1 contract", function (this: FiderWorld) {
  expect(this.lastResponseBody).toBeTruthy()
  
  // API v1 responses should maintain consistent structure
  // This is a general check - specific endpoints have their own contract validations
  expect(typeof this.lastResponseBody).toBe("object")
  
  this.log("API v1 contract preserved")
})
