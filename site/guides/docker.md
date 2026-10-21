# Running on a server

`origins browse` serves a corpus on your own computer. To reach it from another
computer or a phone, or to let a relative work on it with you, run Origins on a server
with the Docker image `ghcr.io/origins-project/origins`, behind a reverse proxy that
serves it over HTTPS on a name of its own, such as `tree.example.org`. The image is
built for linux/amd64 and linux/arm64.

On any address other than the loopback, Origins asks every browser for a passphrase
before it shows anything, and it doesn't start without one.

## With Docker Compose and Caddy

[Caddy](https://caddyserver.com) gets a certificate for the name by itself, so it is
the shortest way to serve Origins over HTTPS. Make a directory on the server with the
corpus in it, a file holding the passphrase, and these two files:

```text
origins/
├── compose.yaml
├── Caddyfile
├── passphrase.txt
└── rossi/            the corpus, a git repository
```

```yaml
# compose.yaml
services:
  origins:
    image: ghcr.io/origins-project/origins:latest
    restart: unless-stopped
    command: ["--passphrase-file", "/run/secrets/passphrase", "/corpus"]
    secrets: [passphrase]
    volumes:
      - ./rossi:/corpus
    environment:
      GIT_AUTHOR_NAME: Rosa Bianchi
      GIT_AUTHOR_EMAIL: rosa@example.org
      GIT_COMMITTER_NAME: Rosa Bianchi
      GIT_COMMITTER_EMAIL: rosa@example.org

  caddy:
    image: caddy:2
    restart: unless-stopped
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy-data:/data

secrets:
  passphrase:
    file: ./passphrase.txt

volumes:
  caddy-data:
```

```text
# Caddyfile
tree.example.org {
	reverse_proxy origins:8100
}
```

Point the name at the server in your DNS, then start both:

```bash
docker compose up -d
```

Open `https://tree.example.org` and give the passphrase. Caddy keeps its certificates
in the `caddy-data` volume, and renews them by itself.

The corpus is mounted at `/corpus`, and it must be writable, because each change you
make in the pages is a commit in it. The server runs as user ID 1000, so that user must
be able to write the corpus and read `passphrase.txt`. If they belong to somebody else,
add `user: "1001:1001"` to the service, with the owner's IDs.

Each commit carries the name and email in the `GIT_` variables, so everyone who
changes the corpus through this server is recorded under them.

Compose mounts the passphrase file at `/run/secrets/passphrase`. You can set
`ORIGINS_PASSPHRASE` instead, but then the passphrase shows in `docker inspect`.

Caddy reaches Origins over compose's own network, so the `origins` service publishes
no port.

## Behind Traefik

If the server already runs [Traefik](https://traefik.io) for other services, give
the Origins service labels and put it on Traefik's network. This assumes an
entrypoint named `websecure` and a certificate resolver named `letsencrypt` in
Traefik's own configuration, and a network named `proxy`:

```yaml
services:
  origins:
    image: ghcr.io/origins-project/origins:latest
    restart: unless-stopped
    command: ["--passphrase-file", "/run/secrets/passphrase", "/corpus"]
    secrets: [passphrase]
    volumes:
      - ./rossi:/corpus
    environment:
      GIT_AUTHOR_NAME: Rosa Bianchi
      GIT_AUTHOR_EMAIL: rosa@example.org
      GIT_COMMITTER_NAME: Rosa Bianchi
      GIT_COMMITTER_EMAIL: rosa@example.org
    networks: [proxy]
    labels:
      - traefik.enable=true
      - traefik.http.routers.origins.rule=Host(`tree.example.org`)
      - traefik.http.routers.origins.entrypoints=websecure
      - traefik.http.routers.origins.tls.certresolver=letsencrypt
      - traefik.http.services.origins.loadbalancer.server.port=8100

secrets:
  passphrase:
    file: ./passphrase.txt

networks:
  proxy:
    external: true
```

Traefik passes on the host name and the WebSocket that the pages use, and needs nothing
else.

## Behind nginx

If nginx already serves the name, publish Origins on the loopback only, with
`ports: ["127.0.0.1:8100:8100"]` in place of the proxy service, and add a server block:

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 443 ssl;
    server_name tree.example.org;
    # ssl_certificate and ssl_certificate_key as for your other sites

    location / {
        proxy_pass http://127.0.0.1:8100;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 1h;
    }
}
```

Each of these lines is needed:

- `Host` is the name the browser asked for. Origins takes a sign-in or a change only
  from a page it drew itself, and it compares the browser's address with this header.
  Without it, nginx sends its own name for the upstream, and Origins refuses the
  passphrase.
- `X-Forwarded-Proto` tells Origins that the browser used HTTPS, so that it marks its
  cookie as secure.
- `Upgrade`, `Connection` and the long `proxy_read_timeout` keep open the WebSocket
  through which the server tells every page that the corpus has changed.

## A read-only view for relatives

A second service can serve the same corpus without what is private and without the
living, read-only, on another name. Mount the corpus read-only, and give the view its
own passphrase:

```yaml
  family:
    image: ghcr.io/origins-project/origins:latest
    restart: unless-stopped
    command: ["--public", "--passphrase-file", "/run/secrets/family-passphrase", "/corpus"]
    secrets: [family-passphrase]
    volumes:
      - ./rossi:/corpus:ro
```

Add `family-passphrase` under `secrets`, and a second block to the `Caddyfile`, such
as `family.example.org { reverse_proxy family:8100 }`. The view shows each change made
through the first service as soon as it is committed.

[Privacy](./privacy) describes what `--public` leaves out, and the other views.

## Fetching and pushing from the pages

When you fetch, pull or push from the pages, git runs in the container, where nobody
can type a password, so the image tells git never to ask. A remote over SSH needs a key
without a passphrase and the remote's host key:

```yaml
    volumes:
      - ./rossi:/corpus
      - ./ssh/id_ed25519:/home/origins/.ssh/id_ed25519:ro
      - ./ssh/known_hosts:/home/origins/.ssh/known_hosts:ro
```

Make a key for this server alone, with `ssh-keygen -t ed25519 -N '' -f ssh/id_ed25519`,
add its public half to the remote as a deploy key with write access, and fill
`known_hosts` with `ssh-keyscan github.com > ssh/known_hosts`, or with your own host.
A remote over HTTPS needs its credentials in a credential helper that git can read
without asking, such as `store` with a `.git-credentials` file mounted the same way.

## Updating

Each release publishes the image tagged with its version and with `latest`. To update,
pull the new image and start the service again:

```bash
docker compose pull origins
docker compose up -d
```

To stay on one release, use its version in place of `latest`, such as
`ghcr.io/origins-project/origins:0.1.0`.

## Without Compose

The same container started by hand, published on the loopback for a proxy that is
already running:

```bash
docker run -d --name origins --restart unless-stopped \
  -p 127.0.0.1:8100:8100 \
  -v "$HOME/genealogy/rossi:/corpus" \
  -e ORIGINS_PASSPHRASE='a long passphrase' \
  -e GIT_AUTHOR_NAME='Rosa Bianchi' -e GIT_AUTHOR_EMAIL=rosa@example.org \
  -e GIT_COMMITTER_NAME='Rosa Bianchi' -e GIT_COMMITTER_EMAIL=rosa@example.org \
  ghcr.io/origins-project/origins:latest
```

The arguments after the image name replace `/corpus`, so they end with the corpus. You
can give any flag of [`origins browse`](../reference/commands#origins-browse) there.
