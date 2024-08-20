# 🎁 Wishpage

Wishpage is a self-hosted wishlist for family and friends. It displays gift
ideas for multiple people. The page viewers can anonymously reserve a gift if
they plan to buy it. The wishlist authors can edit the content by authenticating
with a common password.

The frontend is a Vite React App written in Typescript, while the backend is
written in Go.

## Deployment

The easiest way to deploy this app is via Docker. Here is a Docker Compose
configuration example:

```yaml
name: wishpage

services:
  wishpage:
    container_name: wishpage_server
    image: wishpage:latest
    environment:
      - ADMIN_PASSWORD=<<admin_password>>
      - DATABASE_DIR=/app/db
    volumes:
      - <<local_database_directory>>:/app/db
    ports:
      - 10000:8080
    restart: always
```

Remember to specify a strong admin password and a local directory where to store
the database.

The container has only an HTTP endpoint, so it shouldn't be directly exposed on
the internet. You can configure a reverse proxy or something like [Cloudflare
Tunnel](https://www.cloudflare.com/products/tunnel/) to provide an additional
layer of security.

## Development

First, install `go` and `node`. You can run the server with the following
commands:

```bash
cd wishpage-server
export ADMIN_PASSWORD=toto
export DEV_MODE=1
go run main.go
```

The server will create an in-memory database and it will fill it with dummy
data. The frontend can be run with:

```bash
cd wishpage-app
npm run dev
```

A development server for the web app will run on `http://localhost:5173`. Any
change to the frontend will automatically be applied.

In order to build the Docker image, install [Docker
Desktop](https://docs.docker.com/desktop/) and then run:

```bash
docker build -t wishpage .
```
