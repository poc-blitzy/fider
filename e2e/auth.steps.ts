import { Given, When, Then } from "@cucumber/cucumber"
import { expect } from "@playwright/test"
import { FiderWorld } from "e2e/world"
import { delay, parseJwtToken, isTokenExpired, extractAccessTokenFromResponse } from "./step_definitions/fns"

// Module-level variables for storing authentication request/response state
let authRequestUrl: string | undefined
let authRequestMethod: string | undefined
let authRequestHeaders: Record<string, string> = {}
let authRequestBody: any = undefined
let authResponse: Response | undefined
let authResponseBody: any = undefined

/**
 * Given step: Set up credentials for login request
 * Example: Given I have login credentials with email "user@example.com" and password "password123"
 */
Given(
  "I have login credentials with email {string} and password {string}",
  async function (this: FiderWorld, email: string, password: string) {
    authRequestBody = { email, password }
    this.log(`Prepared login credentials for email: ${email}`)
  }
)

/**
 * Given step: Set up the access token in Authorization header
 * Example: Given I have a valid access token
 */
Given("I have a valid access token", async function (this: FiderWorld) {
  if (!this.accessToken) {
    throw new Error("No access token available in world context. Please authenticate first.")
  }
  authRequestHeaders["Authorization"] = `Bearer ${this.accessToken}`
  this.log(`Set Authorization header with access token`)
})

/**
 * Given step: Set up an expired access token for testing refresh flow
 * Example: Given I have an expired access token
 */
Given("I have an expired access token", async function (this: FiderWorld) {
  // For testing purposes, create a minimal expired token structure
  // In real scenarios, this would come from a previous login that has expired
  if (this.accessToken && isTokenExpired(this.accessToken)) {
    authRequestHeaders["Authorization"] = `Bearer ${this.accessToken}`
    this.log(`Using expired access token from context`)
  } else {
    this.log(`Warning: Current access token is not expired. Test may not behave as expected.`)
  }
})

/**
 * Given step: Set up custom X-Tenant-ID header for cross-origin tenant resolution
 * Example: Given I set tenant ID header to "acme"
 */
Given("I set tenant ID header to {string}", async function (this: FiderWorld, tenantId: string) {
  authRequestHeaders["X-Tenant-ID"] = tenantId
  this.log(`Set X-Tenant-ID header to: ${tenantId}`)
})

/**
 * When step: Send login request to authentication endpoint
 * Example: When I send a login request
 */
When("I send a login request", async function (this: FiderWorld) {
  authRequestUrl = `${this.backendUrl}/api/v1/auth/login`
  authRequestMethod = "POST"
  authRequestHeaders["Content-Type"] = "application/json"

  this.log(`Sending POST request to ${authRequestUrl}`)
  this.log(`Request body: ${JSON.stringify(authRequestBody)}`)

  authResponse = await fetch(authRequestUrl, {
    method: authRequestMethod,
    headers: authRequestHeaders,
    body: JSON.stringify(authRequestBody),
    credentials: "include", // Important for refresh cookie handling
  })

  const contentType = authResponse.headers.get("content-type")
  if (contentType && contentType.includes("application/json")) {
    authResponseBody = await authResponse.json()
    this.log(`Response status: ${authResponse.status}`)
    this.log(`Response body: ${JSON.stringify(authResponseBody)}`)
  } else {
    authResponseBody = await authResponse.text()
    this.log(`Response status: ${authResponse.status}`)
    this.log(`Response body (text): ${authResponseBody}`)
  }

  // Store access token in world context for subsequent requests
  const accessToken = extractAccessTokenFromResponse(authResponseBody)
  if (accessToken) {
    this.accessToken = accessToken
    this.log(`Stored access token in world context`)
  }
})

/**
 * When step: Send token refresh request
 * Example: When I send a token refresh request
 */
When("I send a token refresh request", async function (this: FiderWorld) {
  authRequestUrl = `${this.backendUrl}/api/v1/auth/refresh`
  authRequestMethod = "POST"
  authRequestHeaders["Content-Type"] = "application/json"

  this.log(`Sending POST request to ${authRequestUrl}`)

  authResponse = await fetch(authRequestUrl, {
    method: authRequestMethod,
    headers: authRequestHeaders,
    credentials: "include", // Critical for sending refresh cookie
  })

  const contentType = authResponse.headers.get("content-type")
  if (contentType && contentType.includes("application/json")) {
    authResponseBody = await authResponse.json()
    this.log(`Response status: ${authResponse.status}`)
    this.log(`Response body: ${JSON.stringify(authResponseBody)}`)
  } else {
    authResponseBody = await authResponse.text()
    this.log(`Response status: ${authResponse.status}`)
    this.log(`Response body (text): ${authResponseBody}`)
  }

  // Update access token in world context
  const newAccessToken = extractAccessTokenFromResponse(authResponseBody)
  if (newAccessToken) {
    this.accessToken = newAccessToken
    this.log(`Updated access token in world context`)
  }
})

/**
 * When step: Send logout request
 * Example: When I send a logout request
 */
When("I send a logout request", async function (this: FiderWorld) {
  authRequestUrl = `${this.backendUrl}/api/v1/auth/logout`
  authRequestMethod = "POST"
  authRequestHeaders["Content-Type"] = "application/json"

  // Include Authorization header if access token is available
  if (this.accessToken) {
    authRequestHeaders["Authorization"] = `Bearer ${this.accessToken}`
  }

  this.log(`Sending POST request to ${authRequestUrl}`)

  authResponse = await fetch(authRequestUrl, {
    method: authRequestMethod,
    headers: authRequestHeaders,
    credentials: "include", // Required to clear refresh cookie
  })

  const contentType = authResponse.headers.get("content-type")
  if (contentType && contentType.includes("application/json")) {
    authResponseBody = await authResponse.json()
    this.log(`Response status: ${authResponse.status}`)
    this.log(`Response body: ${JSON.stringify(authResponseBody)}`)
  } else {
    authResponseBody = await authResponse.text()
    this.log(`Response status: ${authResponse.status}`)
    this.log(`Response body (text): ${authResponseBody}`)
  }

  // Clear tokens from world context after logout
  this.accessToken = null
  this.refreshToken = null
  this.log(`Cleared tokens from world context`)
})

/**
 * When step: Make authenticated request with Bearer token
 * Example: When I make an authenticated request to "/api/v1/posts"
 */
When("I make an authenticated request to {string}", async function (this: FiderWorld, endpoint: string) {
  if (!this.accessToken) {
    throw new Error("No access token available. Please authenticate first.")
  }

  authRequestUrl = `${this.backendUrl}${endpoint}`
  authRequestMethod = "GET"
  authRequestHeaders["Authorization"] = `Bearer ${this.accessToken}`
  authRequestHeaders["Content-Type"] = "application/json"

  this.log(`Sending GET request to ${authRequestUrl} with Bearer token`)

  authResponse = await fetch(authRequestUrl, {
    method: authRequestMethod,
    headers: authRequestHeaders,
    credentials: "include",
  })

  const contentType = authResponse.headers.get("content-type")
  if (contentType && contentType.includes("application/json")) {
    authResponseBody = await authResponse.json()
  } else {
    authResponseBody = await authResponse.text()
  }

  this.log(`Response status: ${authResponse.status}`)
})

/**
 * Then step: Verify login response returns access token and user object
 * Example: Then the login response should contain an access token and user
 */
Then("the login response should contain an access token and user", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  expect(authResponse?.status).toBe(200)
  expect(authResponseBody).toBeDefined()
  expect(authResponseBody).toHaveProperty("user")
  
  const accessToken = extractAccessTokenFromResponse(authResponseBody)
  expect(accessToken).toBeTruthy()
  expect(typeof accessToken).toBe("string")
  
  this.log(`Verified login response contains access token and user object`)
})

/**
 * Then step: Verify refresh cookie is set with correct security attributes
 * Example: Then the response should set a refresh token cookie with secure attributes
 */
Then(
  "the response should set a refresh token cookie with secure attributes",
  async function (this: FiderWorld) {
    expect(authResponse).toBeDefined()
    
    const setCookieHeader = authResponse?.headers.get("set-cookie")
    expect(setCookieHeader).toBeTruthy()
    
    // Verify HttpOnly attribute
    expect(setCookieHeader).toContain("HttpOnly")
    this.log(`Verified HttpOnly attribute in Set-Cookie header`)
    
    // Verify Secure attribute (for HTTPS)
    expect(setCookieHeader).toContain("Secure")
    this.log(`Verified Secure attribute in Set-Cookie header`)
    
    // Verify SameSite=None for cross-origin support
    expect(setCookieHeader).toContain("SameSite=None")
    this.log(`Verified SameSite=None attribute in Set-Cookie header`)
    
    this.log(`All refresh cookie security attributes validated`)
  }
)

/**
 * Then step: Verify access token has valid JWT structure
 * Example: Then the access token should be a valid JWT
 */
Then("the access token should be a valid JWT", async function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  // Verify JWT structure (header.payload.signature)
  const tokenParts = this.accessToken!.split(".")
  expect(tokenParts.length).toBe(3)
  this.log(`Verified JWT has three parts (header.payload.signature)`)
  
  // Decode and verify payload can be parsed
  const payload = parseJwtToken(this.accessToken!)
  expect(payload).toBeDefined()
  expect(payload).toHaveProperty("exp")
  this.log(`Verified JWT payload contains expiration claim`)
  
  // Verify token is not expired
  expect(isTokenExpired(this.accessToken!)).toBe(false)
  this.log(`Verified access token is not expired`)
})

/**
 * Then step: Verify token contains expected claims
 * Example: Then the access token should contain user ID and tenant ID claims
 */
Then("the access token should contain user ID and tenant ID claims", async function (this: FiderWorld) {
  expect(this.accessToken).toBeTruthy()
  
  const payload = parseJwtToken(this.accessToken!)
  
  // Verify user identification claims
  expect(payload).toHaveProperty("user_id")
  this.log(`Verified token contains user_id claim: ${payload.user_id}`)
  
  // Verify tenant isolation claims
  expect(payload).toHaveProperty("tenant_id")
  this.log(`Verified token contains tenant_id claim: ${payload.tenant_id}`)
  
  // Verify standard JWT claims
  expect(payload).toHaveProperty("exp")
  expect(payload).toHaveProperty("iat")
  this.log(`Verified token contains standard JWT claims (exp, iat)`)
})

/**
 * Then step: Verify refresh response returns new access token
 * Example: Then the refresh response should contain a new access token
 */
Then("the refresh response should contain a new access token", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  expect(authResponse?.status).toBe(200)
  expect(authResponseBody).toBeDefined()
  
  const newAccessToken = extractAccessTokenFromResponse(authResponseBody)
  expect(newAccessToken).toBeTruthy()
  expect(typeof newAccessToken).toBe("string")
  
  this.log(`Verified refresh response contains new access token`)
})

/**
 * Then step: Verify refresh cookie is rotated
 * Example: Then the response should rotate the refresh token cookie
 */
Then("the response should rotate the refresh token cookie", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  
  const setCookieHeader = authResponse?.headers.get("set-cookie")
  expect(setCookieHeader).toBeTruthy()
  
  // Verify new refresh cookie is set
  expect(setCookieHeader).toContain("refresh_token")
  this.log(`Verified new refresh token cookie is set`)
  
  // Store indication that refresh token was rotated
  this.refreshToken = "rotated"
  this.log(`Refresh token cookie has been rotated`)
})

/**
 * Then step: Verify logout clears refresh cookie
 * Example: Then the response should clear the refresh token cookie
 */
Then("the response should clear the refresh token cookie", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  expect(authResponse?.status).toBe(200)
  
  const setCookieHeader = authResponse?.headers.get("set-cookie")
  if (setCookieHeader) {
    // Verify cookie is cleared (Max-Age=0 or expires in the past)
    const isCookieCleared =
      setCookieHeader.includes("Max-Age=0") ||
      setCookieHeader.includes("expires=Thu, 01 Jan 1970")
    expect(isCookieCleared).toBe(true)
    this.log(`Verified refresh token cookie is cleared`)
  }
})

/**
 * Then step: Verify authenticated request succeeds with Bearer token
 * Example: Then the authenticated request should succeed
 */
Then("the authenticated request should succeed", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  expect(authResponse?.status).toBe(200)
  this.log(`Verified authenticated request succeeded with status 200`)
})

/**
 * Then step: Verify request without token fails with 401
 * Example: Then the request should fail with unauthorized error
 */
Then("the request should fail with unauthorized error", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  expect(authResponse?.status).toBe(401)
  this.log(`Verified request without valid token returned 401 Unauthorized`)
})

/**
 * Then step: Verify CORS headers are present for cross-origin requests
 * Example: Then the response should include CORS headers
 */
Then("the response should include CORS headers", async function (this: FiderWorld) {
  expect(authResponse).toBeDefined()
  
  const allowOrigin = authResponse?.headers.get("access-control-allow-origin")
  expect(allowOrigin).toBeTruthy()
  this.log(`Verified Access-Control-Allow-Origin header: ${allowOrigin}`)
  
  const allowCredentials = authResponse?.headers.get("access-control-allow-credentials")
  expect(allowCredentials).toBe("true")
  this.log(`Verified Access-Control-Allow-Credentials: true`)
})

/**
 * Then step: Verify the authentication response status code
 * Example: Then the authentication response status should be 200
 */
Then("the authentication response status should be {int}", async function (this: FiderWorld, expectedStatus: number) {
  expect(authResponse).toBeDefined()
  expect(authResponse?.status).toBe(expectedStatus)
  this.log(`Verified authentication response status: ${expectedStatus}`)
})

/**
 * Then step: Wait for token expiration (for testing refresh scenarios)
 * Example: Then I wait for the token to expire
 */
Then("I wait for the token to expire", async function (this: FiderWorld) {
  if (!this.accessToken) {
    throw new Error("No access token to wait for expiration")
  }
  
  const payload = parseJwtToken(this.accessToken)
  const expirationTime = payload.exp * 1000 // Convert to milliseconds
  const currentTime = Date.now()
  const waitTime = expirationTime - currentTime + 1000 // Wait 1 second past expiration
  
  if (waitTime > 0) {
    this.log(`Waiting ${waitTime}ms for token to expire`)
    await delay(waitTime)
    this.log(`Token should now be expired`)
  } else {
    this.log(`Token is already expired`)
  }
  
  // Verify token is now expired
  expect(isTokenExpired(this.accessToken)).toBe(true)
})

/**
 * Then step: Verify token expiration claim is within expected range
 * Example: Then the access token should expire within 60 minutes
 */
Then("the access token should expire within {int} minutes", async function (this: FiderWorld, minutes: number) {
  expect(this.accessToken).toBeTruthy()
  
  const payload = parseJwtToken(this.accessToken!)
  const expirationTime = payload.exp * 1000 // Convert to milliseconds
  const currentTime = Date.now()
  const maxExpirationTime = currentTime + minutes * 60 * 1000
  
  expect(expirationTime).toBeLessThanOrEqual(maxExpirationTime)
  this.log(`Verified token expiration is within ${minutes} minutes`)
})
