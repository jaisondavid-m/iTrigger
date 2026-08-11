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

### Docker Compose

1. **Start the server in background (builds automatically if needed):**

```bash
docker compose up --build -d
```

2. **Stop and remove the container:**

```bash
docker compose down
```