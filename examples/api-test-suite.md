# Write API Test Suite

Create comprehensive integration tests for the REST API endpoints.

## Scope

Cover the following endpoint groups with happy-path and error-case tests:

### Users API
- `GET /api/users` - list users (paginated)
- `GET /api/users/:id` - get single user
- `POST /api/users` - create user (validate required fields)
- `PUT /api/users/:id` - update user
- `DELETE /api/users/:id` - delete user (soft delete)

### Posts API
- `GET /api/posts` - list posts with optional `?author=` filter
- `GET /api/posts/:id` - get single post with author info
- `POST /api/posts` - create post (requires auth)
- `PUT /api/posts/:id` - update post (owner only)
- `DELETE /api/posts/:id` - delete post (owner or admin)

## Test Requirements

- Use the project's existing test framework
- Each test should set up its own data (no shared mutable state between tests)
- Test both success responses (200/201) and error responses (400/401/403/404)
- Validate response body structure, not just status codes
- Include at least one test for pagination (limit/offset)
- Test auth-protected endpoints with and without valid tokens
