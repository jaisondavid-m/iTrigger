# iTrigger

A simple deployment automation server written in pure Go.

## Environment

Create a local `.env` file or export the secret in your shell:

```bash
GITHUB_WEBHOOK_SECRET=your-secret
```

`.env` is ignored by git, and `.env.example` shows the required variables.

## Run

```bash
go run ./cmd/server