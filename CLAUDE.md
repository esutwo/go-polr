# go-polr

go-polr is a re-write of the popular [Polr](https://github.com/cydrobolt/polr) project in golang instead of PHP. Architecturally it is different, but most importantly, it uses the same database schema.

## Tech Stack

* **Web Framework:** Gin (github.com/gin-gonic/gin)
* **ORM:** GORM
* **Database:** MySQL / MariaDB (SQLite for testing)
* **Templates:** Go html/template
* **Sessions:** gin-contrib/sessions with cookie store

## Project Structure

```
cmd/server/          # Application entrypoint
internal/
  config/            # Configuration loading from environment
  database/          # Database connection setup
  handlers/          # HTTP request handlers (web + API)
  helpers/           # Utility functions (hashing, generation, etc.)
  middleware/        # Auth, CSRF, rate limiting, logging
  models/            # GORM models (User, Link, Click)
  router/            # Route definitions
  services/          # Business logic layer
web/
  static/            # CSS, JS assets
  templates/         # HTML templates organized by section
testutil/            # Test utilities
```

## Running

```bash
# Copy environment config
cp .env.example .env

# Run the server
go run cmd/server/main.go
```

## Key Features

* URL shortening with optional custom endings
* Secret links (require key to access)
* User authentication with session management
* Admin panel for user and link management
* API v2 compatible with original Polr
* Click tracking and analytics
