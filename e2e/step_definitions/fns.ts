import { Page } from "@playwright/test"

export function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

export async function isAuthenticated(page: Page): Promise<boolean> {
  const serverData = JSON.parse(await page.innerText("#server-data"))
  return serverData.user !== undefined
}

// On E2E test, every user is created as {userName}-{tenantName}
export async function isAuthenticatedAsUser(page: Page, userName: string): Promise<boolean> {
  const serverData = JSON.parse(await page.innerText("#server-data"))
  return serverData.user ? serverData.email.startsWith(userName) : false
}

export async function getLatestLinkSentTo(address: string): Promise<string> {
  await delay(1000)

  // Use environment variable for configurable MailHog location in cross-origin testing
  const mailhogUrl = process.env.MAILHOG_URL || 'http://localhost:8025'
  const response = await fetch(`${mailhogUrl}/api/v2/search?kind=to&query=${address}`)
  const responseBody = await response.json()
  const emailHtml = responseBody.items[0].Content.Body
  // Updated regex pattern to work with configurable frontend URLs in cross-origin setup
  const reg = /https?:\/\/[^\/]+\/(.*)verify\?k=.+?(?=')/gim
  const result = reg.exec(emailHtml)
  if (!result) {
    throw new Error("Could not find a link in email content.")
  }

  return result[0]
}

/**
 * Decodes a JWT token and returns its payload without verification.
 * For testing purposes only - does not validate signature.
 * 
 * @param token - JWT token string
 * @returns Decoded JWT payload object
 */
export function parseJwtToken(token: string): any {
  try {
    const base64Url = token.split('.')[1]
    const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/')
    const jsonPayload = decodeURIComponent(
      atob(base64)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    )
    return JSON.parse(jsonPayload)
  } catch (error) {
    throw new Error(`Failed to parse JWT token: ${error}`)
  }
}

/**
 * Checks if a JWT token is expired by comparing the exp claim with current timestamp.
 * 
 * @param token - JWT token string
 * @returns true if token is expired, false otherwise
 */
export function isTokenExpired(token: string): boolean {
  try {
    const payload = parseJwtToken(token)
    if (!payload.exp) {
      return false // No expiration claim means token doesn't expire
    }
    // exp claim is in seconds, Date.now() is in milliseconds
    return payload.exp * 1000 < Date.now()
  } catch (error) {
    return true // If we can't parse it, consider it expired
  }
}

/**
 * Extracts the access token from a login API response.
 * Supports both { accessToken: "..." } and { token: "..." } response formats.
 * 
 * @param responseBody - Parsed JSON response body from login endpoint
 * @returns Access token string or null if not found
 */
export function extractAccessTokenFromResponse(responseBody: any): string | null {
  if (responseBody && typeof responseBody === 'object') {
    return responseBody.accessToken || responseBody.token || null
  }
  return null
}
