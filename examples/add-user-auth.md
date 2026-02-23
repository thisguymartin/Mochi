# Add User Authentication

Implement a JWT-based authentication system for the REST API.

## Requirements

1. **Registration endpoint** (`POST /api/auth/register`)
   - Accept email and password
   - Hash passwords with bcrypt (cost factor 12)
   - Return a JWT access token on success

2. **Login endpoint** (`POST /api/auth/login`)
   - Validate credentials against stored hashes
   - Return a JWT access token (1h expiry) and refresh token (7d expiry)

3. **Auth middleware**
   - Extract Bearer token from Authorization header
   - Verify JWT signature and expiration
   - Attach user context to the request

4. **Token refresh** (`POST /api/auth/refresh`)
   - Accept a valid refresh token
   - Return a new access token

## Technical Notes

- Use `golang-jwt/jwt/v5` for JWT handling
- Store users in the existing PostgreSQL database
- Add a `users` table migration
- All auth endpoints should return proper HTTP status codes (401, 403, 409)
- Write unit tests for the middleware and token generation
