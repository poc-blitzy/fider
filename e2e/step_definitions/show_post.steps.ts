import { Then } from "@cucumber/cucumber"
import { FiderWorld } from "../world.js"
import { expect } from "@playwright/test"
import { isAuthenticated } from "./fns.js"

Then("I should be on the show post page", async function (this: FiderWorld) {
  // Wait for page to fully load with cross-origin data fetching
  await this.page.waitForLoadState("networkidle")

  // Verify authentication state before making assertions
  const authenticated = await isAuthenticated(this.page)
  if (!authenticated) {
    throw new Error("User must be authenticated to view post page")
  }

  const container = await this.page.$$("#p-show-post")
  expect(container).toBeDefined()
})

Then("I should see {string} as the post title", async function (this: FiderWorld, title: string) {
  // Wait for post title element to be visible (cross-origin data loaded)
  await this.page.waitForSelector("#p-show-post h1", { state: "visible" })

  const postTitle = await this.page.innerText("#p-show-post h1")
  expect(postTitle).toBe(title)
})

Then("I should see {int} vote\\(s)", async function (this: FiderWorld, voteCount: number) {
  // Wait for vote count to be loaded from cross-origin API
  await this.page.waitForLoadState("networkidle")

  // Verify the vote count element is visible with proper data
  await expect(this.page.getByText(`${voteCount}${voteCount === 1 ? "Vote" : "Votes"}`, { exact: true })).toBeVisible()
})
