# ShowMyCards

ShowMyCards is a self-hosted application for managing your Magic: The Gathering collection, using data from the most awesome [Scryfall][Scryfall].

You can learn more about ShowMyCards at [showmy.cards][ShowMyCards]

## Quick Start with Docker

```bash
# Pull and run the combined image
docker run -d \
  --name showmycards \
  -p 3000:3000 \
  -p 3001:3001 \
  -v showmycards-data:/app/data \
  showmycards:latest

# Or use Docker Compose
docker compose up -d
```

Access the application at `http://localhost:3001`

## Useful Links

- [Setup and Install](https://showmy.cards/install)
- [User Guide](https://showmy.cards/user)
- [Self Hosting](https://showmy.cards/selfhost)
- [API Reference](https://showmy.cards/api)

## Features

- Manage storage locations, such as binders and boxes
- Search and Bulk Data powered by Scryfall
- Master Lists - track collections separately from inventory
- Rule evaluation engine - express your rules, and when a card is evaluated, it'll be sent to the right location

More features are coming soon!

## Docker Images

ShowMyCards can be deployed in two ways:

### Combined Image (Recommended)

A single container running both the API backend and web frontend:

```bash
docker compose up -d
```

- Backend API: `http://localhost:3000`
- Frontend: `http://localhost:3001`

### Separate Images

For more control, run backend and frontend as separate containers:

```bash
docker compose -f docker-compose.separate.yml up -d
```

This allows independent scaling and deployment of each service.

## Tech Stack

- **Backend**: Go with Fiber, GORM, and SQLite
- **Frontend**: TypeScript with SvelteKit and DaisyUI
- **Type Generation**: Tygo (Go structs to TypeScript)

## Why Separate the Front and Backend?

If you don't want to self host the UI, or want to build your own UI, you can; all that's required is to consume the APIs provided by the backend (ensuring consistent behaviour).

I'm also considering building a native mobile app in the future too.

## Development

See [DEVELOPMENT.md](DEVELOPMENT.md) for development setup and contribution guidelines.

[ShowMyCards]: https://showmy.cards
[Scryfall]: https://scryfall.com
