# Origins in a container

The image serves one corpus to a browser, as `origins browse` does on the machine the
corpus is on, for reaching it from elsewhere: another computer, a phone, a relative
who is helping. Off the loopback the server asks every browser for a passphrase before
it shows anything, and refuses to start without one.

[Running on a server](https://origins-project.github.io/origins/guides/docker) on the
site describes how to run the image with Docker Compose behind Caddy, Traefik or nginx.
This file describes the image itself.

## Getting the image

Each release of Origins publishes the image as `ghcr.io/origins-project/origins`, for
linux/amd64 and linux/arm64, tagged with the version and with `latest`. The examples
below call the image `origins`, so give it that name after you pull it:

```bash
docker pull ghcr.io/origins-project/origins:latest
docker tag ghcr.io/origins-project/origins:latest origins
```

You can also build the image yourself, from the repository's root:

```bash
docker build -f docker/Dockerfile -t origins .
```

That image reports its version as `dev`. A release passes the version and the commit
with `--build-arg VERSION=0.1.0 --build-arg COMMIT=<sha>`, and `origins version`
reports them.

## Running

The corpus is a git repository mounted at `/corpus`, writable, since a change made in
a page is a commit in it. The server runs as uid 1000; a corpus owned by somebody else
is served with `--user` set to its owner.

```bash
docker run -d --name origins \
  -p 127.0.0.1:8100:8100 \
  -v "$HOME/genealogy/rossi:/corpus" \
  -e ORIGINS_PASSPHRASE='a long passphrase' \
  -e GIT_AUTHOR_NAME='Rosa Bianchi' -e GIT_AUTHOR_EMAIL=rosa@example.org \
  -e GIT_COMMITTER_NAME='Rosa Bianchi' -e GIT_COMMITTER_EMAIL=rosa@example.org \
  origins
```

The passphrase can be a file instead, with `--passphrase-file` naming it inside the
container, which suits a Docker secret:

```bash
docker run -d -p 127.0.0.1:8100:8100 -v "$HOME/genealogy/rossi:/corpus" \
  -v "$HOME/.config/origins/passphrase:/run/secrets/passphrase:ro" \
  origins --passphrase-file /run/secrets/passphrase /corpus
```

Arguments after the image name replace `/corpus`, so they end with the corpus. Any
flag of `origins browse` can be given there: `--public` serves the corpus without what
is private or whoever is living, read-only, for sharing a tree with family.

## Sharing through git

Fetch, pull and push from a page run git where nobody can answer it, so git is told
never to ask. A remote over SSH needs a key without a passphrase, or an agent's
socket, and the remote's host key already known:

```bash
docker run -d -p 127.0.0.1:8100:8100 -v "$HOME/genealogy/rossi:/corpus" \
  -v "$HOME/.ssh/origins_ed25519:/home/origins/.ssh/id_ed25519:ro" \
  -v "$HOME/.ssh/known_hosts:/home/origins/.ssh/known_hosts:ro" \
  -e ORIGINS_PASSPHRASE='a long passphrase' \
  origins
```

A remote over HTTPS needs its credentials in a helper git can read without asking,
such as `store` with a `.git-credentials` mounted beside the key. A remote git cannot
reach without asking is refused in the page with git's message.

## HTTPS

The session cookie and the passphrase travel with every request, so anything but the
loopback is served behind HTTPS. [Caddy](https://caddyserver.com) obtains a
certificate on its own; with the container published on the loopback as above, a
`Caddyfile` of

```
tree.example.org {
	reverse_proxy 127.0.0.1:8100
}
```

serves it at `https://tree.example.org`. Caddy says the request came as HTTPS in
`X-Forwarded-Proto`, and the server marks its cookie `Secure` from that.
