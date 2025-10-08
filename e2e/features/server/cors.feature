@cors
Feature: CORS (Cross-Origin Resource Sharing)
  As a frontend application on a different origin
  I need proper CORS headers to communicate with the API
  So that cross-origin requests work securely

  Scenario: Preflight OPTIONS request to API endpoint
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should see http status 200
    And I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    And I should see a "Access-Control-Allow-Methods" header
    And I should see a "Access-Control-Allow-Headers" header

  Scenario: Preflight request includes allowed HTTP methods
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should see http status 200
    And the "Access-Control-Allow-Methods" header should include "GET"
    And the "Access-Control-Allow-Methods" header should include "POST"
    And the "Access-Control-Allow-Methods" header should include "PUT"
    And the "Access-Control-Allow-Methods" header should include "PATCH"
    And the "Access-Control-Allow-Methods" header should include "DELETE"
    And the "Access-Control-Allow-Methods" header should include "OPTIONS"

  Scenario: Preflight request includes allowed headers
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should see http status 200
    And the "Access-Control-Allow-Headers" header should include "Authorization"
    And the "Access-Control-Allow-Headers" header should include "Content-Type"
    And the "Access-Control-Allow-Headers" header should include "X-Tenant-ID"

  Scenario: Preflight request includes credentials support
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should see http status 200
    And I should see a "Access-Control-Allow-Credentials" header with value "true"

  Scenario: Preflight request includes max-age for caching
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should see http status 200
    And I should see a "Access-Control-Max-Age" header with value "600"

  Scenario: Cross-origin GET request with CORS headers
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    And I should see a "Access-Control-Allow-Credentials" header with value "true"

  Scenario: Cross-origin POST request with CORS headers
    Given I set the "Origin" header to "http://localhost:3000"
    And I set the "Content-Type" header to "application/json"
    When I send a "POST" request to "/api/v1/posts" with:
      """
      {
        "title": "CORS Test Post",
        "description": "Testing CORS functionality"
      }
      """
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    And I should see a "Access-Control-Allow-Credentials" header with value "true"

  Scenario: Request from allowed origin receives CORS headers
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header

  Scenario: Request from different allowed origin receives correct CORS headers
    Given I set the "Origin" header to "http://localhost:5173"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:5173"

  Scenario: Origin whitelist enforcement - disallowed origin
    Given I set the "Origin" header to "http://malicious-site.com"
    When I send a "GET" request to "/api/v1/posts"
    Then I should not see a "Access-Control-Allow-Origin" header

  Scenario: Preflight request from disallowed origin
    Given I set the "Origin" header to "http://unauthorized-domain.com"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should not see a "Access-Control-Allow-Origin" header

  Scenario: Cross-origin request with credentials (cookies)
    Given I set the "Origin" header to "http://localhost:3000"
    And I set the "Cookie" header to "session=test-session-id"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    And I should see a "Access-Control-Allow-Credentials" header with value "true"

  Scenario: Cross-origin authenticated request with Authorization header
    Given I set the "Origin" header to "http://localhost:3000"
    And I set the "Authorization" header to "Bearer test-jwt-token"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    And I should see a "Access-Control-Allow-Credentials" header with value "true"

  Scenario: Cross-origin request with X-Tenant-ID header
    Given I set the "Origin" header to "http://localhost:3000"
    And I set the "X-Tenant-ID" header to "demo"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"

  Scenario: CORS headers on authentication endpoints
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/auth/login"
    Then I should see http status 200
    And I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    And I should see a "Access-Control-Allow-Credentials" header with value "true"

  Scenario: CORS headers on tags endpoints
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "GET" request to "/api/v1/tags"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"

  Scenario: CORS headers persist across different API endpoints
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "GET" request to "/api/v1/posts"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    When I send a "GET" request to "/api/v1/tags"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"
    When I send a "GET" request to "/api/v1/users"
    Then I should see a "Access-Control-Allow-Origin" header with value "http://localhost:3000"

  Scenario: Multiple preflight requests to different endpoints
    Given I set the "Origin" header to "http://localhost:3000"
    When I send a "OPTIONS" request to "/api/v1/posts"
    Then I should see http status 200
    And I should see a "Access-Control-Allow-Methods" header
    When I send a "OPTIONS" request to "/api/v1/comments"
    Then I should see http status 200
    And I should see a "Access-Control-Allow-Methods" header
    When I send a "OPTIONS" request to "/api/v1/votes"
    Then I should see http status 200
    And I should see a "Access-Control-Allow-Methods" header
