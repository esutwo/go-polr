# go-polr

go-polr is a re-write of the popular [Polr](https://github.com/cydrobolt/polr) project in golang instead of PHP. It uses the same database schema as Polr v2.3.0.

## Disclaimer

This project is not affiliated with or endorsed by the original Polr project or its maintainers. Additionally, it is a work in progress and should not be considered feature complete.

## Usage

Edit your `.env` file to set your database and other configuration options. Then run:

```bash
go build -o go-polr ./cmd/server
./go-polr
```

## Environment Vars

The following environment variables can be set in the `.env` file:

```environment
# ===================
# Application
# ===================
APP_NAME=go-polr
APP_URL=http://localhost:8080
APP_PORT=8080

# ===================
# Database (MySQL/MariaDB)
# ===================
DB_HOST=localhost
DB_PORT=3306
DB_USER=polr
DB_PASSWORD=your-database-password
DB_NAME=polrdb

# ===================
# Security
# ===================
# Session secret (minimum 32 characters, used for session encryption)
SESSION_SECRET=change-me-to-a-random-32-char-string

# CSRF secret (minimum 32 characters, used for CSRF token generation)
CSRF_SECRET=change-me-to-another-random-string

# ===================
# Features
# ===================
# Allow anonymous API access (without API key)
ANON_API_ENABLED=false

# Allow new user registration
REGISTRATION_ENABLED=true
```
