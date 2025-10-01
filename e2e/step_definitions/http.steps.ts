import { Given, Then, When } from "@cucumber/cucumber"
import { FiderWorld } from "e2e/world"
import expect from "expect"

let requestUrl: string
let requestOpts: RequestInit
let requestHeaders: Headers

let response: Response
let responseBody: string

const getFullUrl = (world: FiderWorld, url: string): string => {
  // Use configurable backend URL from world context for cross-origin testing
  return url.startsWith("/") ? `${world.backendUrl}${url}` : url
}

Given("I prepare a {string} request to {string}", async function (this: FiderWorld, method: string, url: string) {
  requestUrl = getFullUrl(this, url)
  requestHeaders = new Headers()
  
  // Automatically include Bearer token authentication if access token is available
  if (this.accessToken) {
    requestHeaders.set("Authorization", `Bearer ${this.accessToken}`)
  }
  
  // Include X-Tenant-ID header for tenant context in cross-origin requests
  if (this.tenantName) {
    requestHeaders.set("X-Tenant-ID", this.tenantName)
  }
  
  requestOpts = {
    method,
    headers: {},
    credentials: 'include', // Enable cookies for refresh token support in cross-origin requests
  }
})

Given("I set the {string} header to {string}", async function (headerName: string, headerValue: string) {
  requestHeaders.set(headerName, headerValue)
})

Given("I send a {string} request to {string}", async function (this: FiderWorld, method: string, url: string) {
  const headers = new Headers()
  
  // Automatically include Bearer token authentication if access token is available
  if (this.accessToken) {
    headers.set("Authorization", `Bearer ${this.accessToken}`)
  }
  
  // Include X-Tenant-ID header for tenant context in cross-origin requests
  if (this.tenantName) {
    headers.set("X-Tenant-ID", this.tenantName)
  }
  
  response = await fetch(getFullUrl(this, url), {
    method,
    headers,
    credentials: 'include', // Enable cookies for refresh token support in cross-origin requests
  })
  responseBody = await response.text()
})

When("I send the request", async function () {
  // Merge requestHeaders with requestOpts, ensuring credentials is set for cross-origin support
  response = await fetch(requestUrl, {
    ...requestOpts,
    headers: requestHeaders,
    credentials: requestOpts.credentials || 'include', // Ensure credentials are included
  })
  responseBody = await response.text()
})

Then("I should see http status {int}", async function (code: number) {
  expect(response.status).toEqual(code)
})

Then("I should not see a {string} header", async function (headerName: string) {
  expect(response.headers.get(headerName)).toBeNull()
})

Then("I should see a {string} header with value {string}", async function (headerName: string, headerValue: string) {
  expect(response.headers.get(headerName)).toBe(headerValue)
})

Then("I should see {string} on the response body", async function (substring: string) {
  expect(responseBody).toContain(substring)
})

Then("I should not see {string} on the response body", async function (substring: string) {
  expect(responseBody).not.toContain(substring)
})
