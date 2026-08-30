# Seatd

Seatd is organized as a single monorepo for backend services, user-facing
applications, shared contracts, infrastructure, operations, and documentation.

## Getting started

Prerequisites:

- Bash 4 or later
- GNU Make
- Git
- Go 1.24 or later
- Node.js 24 and npm 11
- Flutter stable
- Docker and Docker Compose

Bootstrap a clean checkout with:

```sh
make bootstrap
```

This validates the repository layout and local tools, creates `.env.local` from
the safe template when needed, and runs any application-specific bootstrap
hooks added in later phases.

See [the developer guide](docs/development.md) for commands, configuration, and
repository conventions.

## Applications

- `apps/api`: Go HTTP API with `/healthz`, `/status`, and `/version`;
- `apps/worker`: Go background worker skeleton with graceful shutdown;
- `apps/realtime`: Go realtime gateway skeleton with HTTP status endpoints;
- `apps/web`: Next.js owner/platform web skeleton;
- `apps/guest`: lightweight Vite guest web skeleton;
- `apps/waiter`: Flutter waiter app skeleton;
- `apps/display`: Flutter display app skeleton.

Contracts live in `packages/api-contract` and `packages/event-schema`.
Generated clients live in `packages/typescript-seatd-client` and
`packages/dart-seatd-client`.
