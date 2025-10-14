@jwt @authentication
Feature: JWT Authentication
  As a frontend SPA on a different origin
  I need JWT-based authentication with refresh tokens
  So that I can securely authenticate users across origins

  Background:
    Given I have valid test credentials

  Scenario: Successful login with valid credentials
    When I login with my test credentials
    Then the login should succeed
    And I should receive an access token
    And I should receive a refresh cookie

  Scenario: Failed login with invalid credentials
    When I login with email "invalid@test.com" and password "wrongpassword"
    Then the login should fail with status 401

  Scenario: Access token is a valid JWT with proper claims
    When I login with my test credentials
    Then the login should succeed
    And the access token should be a valid JWT
    And the access token should contain tenant information
    And the access token should contain user information
    And the access token should have an expiration time
    And the access token should not be expired

  Scenario: Refresh cookie has correct security attributes
    When I login with my test credentials
    Then the login should succeed
    And the refresh cookie should be HttpOnly
    And the refresh cookie should be Secure
    And the refresh cookie should have SameSite=None
    And the refresh cookie should be scoped to the API domain

  Scenario: Token refresh with valid refresh cookie
    Given I am authenticated with a valid JWT token
    When I request a token refresh
    Then the token refresh should succeed
    And I should receive a new access token
    And the new access token should be different from the old token
    And the new access token should be a valid JWT

  Scenario: Token refresh fails without refresh cookie
    Given I have no authentication token
    When I request a token refresh
    Then the token refresh should fail with status 401

  Scenario: Authenticated request with valid Bearer token
    Given I am authenticated with a valid JWT token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should succeed
    And the Authorization header should contain Bearer token

  Scenario: Authenticated request fails with no token
    Given I have no authentication token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should fail with status 401
    And the response should indicate unauthorized

  Scenario: Authenticated request fails with expired token
    Given I have an expired access token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should fail with status 401
    And the response should indicate unauthorized

  Scenario: Successful logout clears refresh cookie
    Given I am authenticated with a valid JWT token
    When I logout
    Then the logout should succeed
    And the refresh cookie should be cleared

  Scenario: Cross-origin authenticated request with CORS headers
    Given I am authenticated with a valid JWT token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should succeed
    And the response should have CORS headers for cross-origin access

  Scenario: API v1 contract preserved with JWT authentication
    Given I am authenticated with a valid JWT token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should succeed
    And the response should preserve the API v1 contract

  Scenario: Login returns user information
    When I login with my test credentials
    Then the login should succeed
    And the response should contain user information

  Scenario: Multiple authentication scenarios in sequence
    When I login with my test credentials
    Then the login should succeed
    And I should receive an access token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should succeed
    When I request a token refresh
    Then the token refresh should succeed
    And I should receive a new access token
    When I logout
    Then the logout should succeed

  Scenario: Token expiration flow
    Given I am authenticated with a valid JWT token
    When I wait for the token to expire
    And I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should fail with status 401

  Scenario: Login with valid email and specific password
    When I login with email "test@example.com" and password "testpassword123"
    Then the login should fail with status 401

  Scenario: Access token has expiration claim
    When I login with my test credentials
    Then the login should succeed
    And the access token should have an expiration time
    And the access token should not be expired

  Scenario: Bearer token format in Authorization header
    Given I am authenticated with a valid JWT token
    When I make an authenticated request to "/api/v1/tags"
    Then the authenticated request should succeed
    And the Authorization header should contain Bearer token

  Scenario: Authenticated access to multiple API endpoints
    Given I am authenticated with a valid JWT token
    When I make an authenticated request to "/api/v1/posts"
    Then the authenticated request should succeed
    When I make an authenticated request to "/api/v1/tags"
    Then the authenticated request should succeed
    When I make an authenticated request to "/api/v1/users"
    Then the authenticated request should succeed

  Scenario: Refresh token rotation on each refresh
    Given I am authenticated with a valid JWT token
    When I request a token refresh
    Then the token refresh should succeed
    And I should receive a new access token
    When I request a token refresh
    Then the token refresh should succeed
    And I should receive a new access token

  Scenario: Logout without authentication
    Given I have no authentication token
    When I logout
    Then the logout should succeed

  Scenario: Login endpoint is accessible
    When I login with my test credentials
    Then the login should succeed
    And I should receive an access token

  Scenario: Refresh endpoint requires valid refresh cookie
    Given I have no authentication token
    When I request a token refresh
    Then the token refresh should fail with status 401

  Scenario: Access token contains tenant context
    When I login with my test credentials
    Then the login should succeed
    And the access token should contain tenant information

  Scenario: Access token contains user context
    When I login with my test credentials
    Then the login should succeed
    And the access token should contain user information
