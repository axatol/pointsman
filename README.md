# pointsman

simple redirect service to alias domains to a authoritative domain

run alongside something like traefik to automatically handle TLS termination

for example, with docker

```yaml
services:
  traefik:
    container_name: traefik
    image: traefik:v3
    restart: unless-stopped
    command:
      - --entrypoints.websecure.address=:443
      - --certificatesresolvers.cloudflare.acme.email={{ letsencrypt.email }}
      - --certificatesresolvers.cloudflare.acme.storage=/traefik/acme.json
      - --certificatesresolvers.cloudflare.acme.caserver={{ letsencrypt.caserver }}
      - --certificatesresolvers.cloudflare.acme.dnschallenge=true
      - --certificatesresolvers.cloudflare.acme.dnschallenge.provider=cloudflare
    environment:
      CF_DNS_API_TOKEN: "{{ cloudflare.dns_api_token }}"
    pull_policy: missing
    ports:
      - 443:443
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
      - /opt/traefik:/traefik

  pointsman:
    container_name: pointsman
    image: public.ecr.aws/axatol/pointsman:latest
    restart: unless-stopped
    command:
      - /pointsman
      - -redirects=www.example.com=>https://example.com=301
      - -redirects=example.org=>https://example.com=301
      - -redirects=www.example.xyz=>https://example.com=301
      - -redirects=example.xyz=>https://example.com=301
    labels:
      traefik.http.services.pointsman.loadbalancer.server.port: "8000"
      traefik.http.routers.pointsman.tls.certresolver: route53
      traefik.http.routers.pointsman.rule: >-
        Host(`www.example.com`) ||
        Host(`example.org`) ||
        Host(`www.example.xyz`) ||
        Host(`example.xyz`)
```
