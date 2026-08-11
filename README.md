# iTrigger

A simple deployment automation server written in pure Go.

## Environment

Create a local `.env` file or export the secret in your shell:

```bash
GITHUB_WEBHOOK_SECRET=your-secret
```

`.env` is ignored by git, and `.env.example` shows the required variables.

## Run

### Local (Go)

```bash
go run ./cmd/server
```

### Docker

1. **Build the Docker Image:**

```bash
docker build -t itrigger .
```

2. **Run with `.env` file:**

```bash
docker run -d --name itrigger --env-file .env -p 8080:8080 itrigger
```

3. **Or Run with inline secret:**

```bash
docker run -d --name itrigger -e GITHUB_WEBHOOK_SECRET=your-secret -p 8080:8080 itrigger
```

### Docker Compose (with Automatic HTTPS via Caddy)

1. **Configure environment variables in `.env`:**

```env
GITHUB_WEBHOOK_SECRET=your-secret
DOMAIN_NAME=yourdomain.com
```
*(Leave `DOMAIN_NAME` unset or as `localhost` for local testing with self-signed SSL)*.

2. **Start the server & reverse proxy:**

```bash
docker compose up --build -d
```

3. **Stop and remove containers:**

```bash
docker compose down
```