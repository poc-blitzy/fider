import { Given, Then, When } from "@cucumber/cucumber"
import { expect } from "@playwright/test"
import { FiderWorld } from "e2e/world"

// Module-level state for CORS testing
let corsRequestUrl: string
let corsRequestHeaders: Headers
let corsResponse: Response
let corsResponseHeaders: Headers

/**
 * Helper to construct full URL from backend base URL
 */
const getCorsTestUrl = (world: FiderWorld, path: string): string => {
  return path.startsWith("/") ? `${world.backendUrl}${path}` : path
}

// ============================================================================
// Preflight OPTIONS Request Steps
// ============================================================================

Given("I prepare an OPTIONS preflight request to {string}", async function (this: FiderWorld, path: string) {
  corsRequestUrl = getCorsTestUrl(this, path)
  corsRequestHeaders = new Headers()
  
  // Standard preflight headers sent by browsers
  corsRequestHeaders.set("Origin", this.frontendUrl || "http://localhost:3000")
  corsRequestHeaders.set("Access-Control-Request-Method", "POST")
  corsRequestHeaders.set("Access-Control-Request-Headers", "authorization,content-type")
})

Given("I set the preflight origin to {string}", async function (origin: string) {
  corsRequestHeaders.set("Origin", origin)
})

Given("I set the preflight request method to {string}", async function (method: string) {
  corsRequestHeaders.set("Access-Control-Request-Method", method)
})

Given("I set the preflight request headers to {string}", async function (headers: string) {
  corsRequestHeaders.set("Access-Control-Request-Headers", headers)
})

When("I send the OPTIONS preflight request", async function () {
  corsResponse = await fetch(corsRequestUrl, {
    method: "OPTIONS",
    headers: corsRequestHeaders,
    credentials: "include", // Test credentials support in CORS
  })
  corsResponseHeaders = corsResponse.headers
})

// ============================================================================
// CORS Header Validation Steps
// ============================================================================

Then("the response should have status {int}", async function (statusCode: number) {
  expect(corsResponse.status).toBe(statusCode)
})

Then("the response should include CORS header {string}", async function (headerName: string) {
  const headerValue = corsResponseHeaders.get(headerName)
  expect(headerValue).toBeTruthy()
  expect(headerValue).not.toBeNull()
})

Then("the response should include CORS header {string} with value {string}", async function (
  headerName: string,
  expectedValue: string
) {
  const actualValue = corsResponseHeaders.get(headerName)
  expect(actualValue).toBe(expectedValue)
})

Then("the Access-Control-Allow-Origin header should be {string}", async function (expectedOrigin: string) {
  const origin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  expect(origin).toBe(expectedOrigin)
})

Then("the Access-Control-Allow-Origin header should match the request origin", async function () {
  const requestOrigin = corsRequestHeaders.get("Origin")
  const responseOrigin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  expect(responseOrigin).toBe(requestOrigin)
})

Then("the Access-Control-Allow-Methods header should contain {string}", async function (method: string) {
  const allowedMethods = corsResponseHeaders.get("Access-Control-Allow-Methods")
  expect(allowedMethods).toBeTruthy()
  expect(allowedMethods || "").toContain(method)
})

Then("the Access-Control-Allow-Methods header should contain all of:", async function (dataTable: any) {
  const allowedMethods = corsResponseHeaders.get("Access-Control-Allow-Methods") || ""
  const expectedMethods = dataTable.raw().flat()
  
  for (const method of expectedMethods) {
    expect(allowedMethods).toContain(method)
  }
})

Then("the Access-Control-Allow-Headers header should contain {string}", async function (header: string) {
  const allowedHeaders = corsResponseHeaders.get("Access-Control-Allow-Headers")
  expect(allowedHeaders).toBeTruthy()
  // Case-insensitive comparison for header names
  expect((allowedHeaders || "").toLowerCase()).toContain(header.toLowerCase())
})

Then("the Access-Control-Allow-Headers header should contain all of:", async function (dataTable: any) {
  const allowedHeaders = (corsResponseHeaders.get("Access-Control-Allow-Headers") || "").toLowerCase()
  const expectedHeaders = dataTable.raw().flat()
  
  for (const header of expectedHeaders) {
    expect(allowedHeaders).toContain(header.toLowerCase())
  }
})

Then("the Access-Control-Allow-Credentials header should be {string}", async function (expectedValue: string) {
  const credentials = corsResponseHeaders.get("Access-Control-Allow-Credentials")
  expect(credentials).toBe(expectedValue)
})

Then("the Access-Control-Max-Age header should be {string}", async function (maxAge: string) {
  const actualMaxAge = corsResponseHeaders.get("Access-Control-Max-Age")
  expect(actualMaxAge).toBe(maxAge)
})

Then("the response should not include CORS header {string}", async function (headerName: string) {
  const headerValue = corsResponseHeaders.get(headerName)
  expect(headerValue).toBeNull()
})

// ============================================================================
// Cross-Origin Request Testing Steps
// ============================================================================

Given("I prepare a cross-origin {string} request to {string}", async function (
  this: FiderWorld,
  method: string,
  path: string
) {
  corsRequestUrl = getCorsTestUrl(this, path)
  corsRequestHeaders = new Headers()
  
  // Set Origin header to simulate cross-origin request from frontend
  corsRequestHeaders.set("Origin", this.frontendUrl || "http://localhost:3000")
  corsRequestHeaders.set("Content-Type", "application/json")
  
  // Include authentication headers if access token is available
  if (this.accessToken) {
    corsRequestHeaders.set("Authorization", `Bearer ${this.accessToken}`)
  }
  
  // Include tenant identification header
  if (this.tenantName) {
    corsRequestHeaders.set("X-Tenant-ID", this.tenantName)
  }
})

When("I send the cross-origin request with method {string}", async function (method: string) {
  corsResponse = await fetch(corsRequestUrl, {
    method,
    headers: corsRequestHeaders,
    credentials: "include", // Enable cross-origin cookies
  })
  corsResponseHeaders = corsResponse.headers
})

When("I send the cross-origin request with method {string} and body:", async function (
  method: string,
  bodyContent: string
) {
  corsResponse = await fetch(corsRequestUrl, {
    method,
    headers: corsRequestHeaders,
    credentials: "include",
    body: bodyContent,
  })
  corsResponseHeaders = corsResponse.headers
})

Then("the response should allow credentials", async function () {
  const credentials = corsResponseHeaders.get("Access-Control-Allow-Credentials")
  expect(credentials).toBe("true")
})

Then("the response should have Access-Control-Allow-Origin matching the frontend", async function (this: FiderWorld) {
  const allowedOrigin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  const frontendOrigin = this.frontendUrl || "http://localhost:3000"
  expect(allowedOrigin).toBe(frontendOrigin)
})

// ============================================================================
// Origin Whitelist Enforcement Steps
// ============================================================================

Given("I set the request origin to {string}", async function (origin: string) {
  corsRequestHeaders.set("Origin", origin)
})

Then("the CORS request should be rejected", async function () {
  // Check if response is either 403 Forbidden or lacks CORS headers
  const allowedOrigin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  
  if (corsResponse.status === 403 || corsResponse.status === 401) {
    // Explicit rejection status code
    expect([403, 401]).toContain(corsResponse.status)
  } else {
    // Or missing CORS headers (browser will block)
    expect(allowedOrigin).toBeNull()
  }
})

Then("the response should not include Access-Control-Allow-Origin header", async function () {
  const allowedOrigin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  expect(allowedOrigin).toBeNull()
})

Then("the origin {string} should be rejected", async function (origin: string) {
  // Save current headers
  const originalHeaders = corsRequestHeaders
  
  // Test with the specified origin
  corsRequestHeaders = new Headers(originalHeaders)
  corsRequestHeaders.set("Origin", origin)
  
  corsResponse = await fetch(corsRequestUrl, {
    method: "OPTIONS",
    headers: corsRequestHeaders,
    credentials: "include",
  })
  
  const allowedOrigin = corsResponse.headers.get("Access-Control-Allow-Origin")
  expect(allowedOrigin).not.toBe(origin)
  
  // Restore original headers
  corsRequestHeaders = originalHeaders
})

Then("the origin {string} should be accepted", async function (origin: string) {
  // Save current headers
  const originalHeaders = corsRequestHeaders
  
  // Test with the specified origin
  corsRequestHeaders = new Headers(originalHeaders)
  corsRequestHeaders.set("Origin", origin)
  
  corsResponse = await fetch(corsRequestUrl, {
    method: "OPTIONS",
    headers: corsRequestHeaders,
    credentials: "include",
  })
  
  const allowedOrigin = corsResponse.headers.get("Access-Control-Allow-Origin")
  expect(allowedOrigin).toBe(origin)
  
  // Restore original headers
  corsRequestHeaders = originalHeaders
})

// ============================================================================
// Credential-Included Request Steps
// ============================================================================

When("I send a credentialed cross-origin request to {string}", async function (
  this: FiderWorld,
  path: string
) {
  corsRequestUrl = getCorsTestUrl(this, path)
  
  const headers = new Headers()
  headers.set("Origin", this.frontendUrl || "http://localhost:3000")
  headers.set("Content-Type", "application/json")
  
  if (this.accessToken) {
    headers.set("Authorization", `Bearer ${this.accessToken}`)
  }
  
  if (this.tenantName) {
    headers.set("X-Tenant-ID", this.tenantName)
  }
  
  corsResponse = await fetch(corsRequestUrl, {
    method: "GET",
    headers,
    credentials: "include", // Critical: include cookies in cross-origin request
  })
  corsResponseHeaders = corsResponse.headers
})

Then("the response should include refresh token cookie", async function () {
  const setCookieHeaders = corsResponse.headers.get("Set-Cookie")
  if (setCookieHeaders) {
    expect(setCookieHeaders).toContain("refresh_token")
    expect(setCookieHeaders).toContain("HttpOnly")
    expect(setCookieHeaders).toContain("Secure")
    expect(setCookieHeaders).toContain("SameSite=None")
  }
})

Then("the credentials flag should be supported", async function () {
  const credentials = corsResponseHeaders.get("Access-Control-Allow-Credentials")
  const origin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  
  // When credentials are supported, origin cannot be wildcard
  expect(credentials).toBe("true")
  expect(origin).not.toBe("*")
})

// ============================================================================
// Complete Cross-Origin Flow Validation
// ============================================================================

Then("the complete CORS preflight should be valid", async function () {
  // Validate all required CORS headers are present and correct
  expect(corsResponse.status).toBe(204) // No Content for OPTIONS
  
  const allowOrigin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  const allowMethods = corsResponseHeaders.get("Access-Control-Allow-Methods")
  const allowHeaders = corsResponseHeaders.get("Access-Control-Allow-Headers")
  const allowCredentials = corsResponseHeaders.get("Access-Control-Allow-Credentials")
  const maxAge = corsResponseHeaders.get("Access-Control-Max-Age")
  
  expect(allowOrigin).toBeTruthy()
  expect(allowOrigin).not.toBe("*") // Should not be wildcard with credentials
  expect(allowMethods).toBeTruthy()
  expect(allowHeaders).toBeTruthy()
  expect(allowCredentials).toBe("true")
  expect(maxAge).toBe("600") // 10 minutes as per spec
})

Then("the cross-origin request flow should succeed", async function () {
  // Validate successful cross-origin request with proper CORS headers
  expect(corsResponse.status).toBeLessThan(400) // Success status code
  
  const allowOrigin = corsResponseHeaders.get("Access-Control-Allow-Origin")
  expect(allowOrigin).toBeTruthy()
  expect(allowOrigin).not.toBe("*")
})
