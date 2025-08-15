# Contributing

## Prerequisites
- Docker & Docker Compose

## Setup & Development

### Start the Application
```bash
docker compose up --build
```

### Stop the Application
```bash
docker compose down
```

### View Logs
```bash
docker compose logs -f
```

### Rebuild After Changes
```bash
docker compose up --build --force-recreate
```
OR
```bash
docker compose down && docker compose up --build -d
```

## Development Workflow
1. You can use backend as is
2. Frontend was made as simple matchmaking platform for people who interested in learning english, so you can adapt it for your needs
