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
  ghcr.io/showmycards/showmycards:latest

# Or use Docker Compose
docker compose up -d
```

Access the application at `http://localhost:3001`

## Useful Links

- [Installation Guide](https://showmy.cards/download)
- [User Manual](https://showmy.cards/docs)

## Features

- Manage storage locations, such as binders and boxes
- Search and Bulk Data powered by Scryfall
- Master Lists - track collections separately from inventory
- Rule evaluation engine - express your rules, and when a card is evaluated, it'll be sent to the right location

More features are coming soon!

## Docker

A single container runs both the API backend and web frontend:

```bash
docker compose up -d
```

- Backend API: `http://localhost:3000`
- Frontend: `http://localhost:3001`

## Tech Stack

- **Backend**: Go with Fiber and SQLite
- **Frontend**: TypeScript with SvelteKit and Tailwind CSS
- **Type Generation**: Tygo (Go structs to TypeScript)

## Why Separate the Front and Backend?

If you don't want to self host the UI, or want to build your own UI, you can; all that's required is to consume the APIs provided by the backend (ensuring consistent behaviour).

I'm also considering building a native mobile app in the future too.

[ShowMyCards]: https://showmy.cards
[Scryfall]: https://scryfall.com
