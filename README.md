# ⚡ voltproxy - Configurable Reverse Proxy

voltproxy is a reverse proxy designed to simplify the process of proxying Docker containers and other services.
With voltproxy, you can easily manage and redirect traffic to different services using a single YAML configuration file.

Design goals:

1. Configuration file that acts as a single source of truth.
2. Streamlined Docker integration.
3. Automated ACME-based certificate generation and renewal.

## 🔧 Configuration

Example configuration file:

```yaml
# config.yml
services:
  foo:
    host: example.com
    redirect: "https://example.com"
  bar:
    host: bar.example.com
    tls: true
    container:
      name: "/container1"
      network: "network1"
      port: 8080
```

This configuration instructs voltproxy to proxy incoming requests with the URL `http://example.com` to <https://example.com> and proxy incoming requests with the URL `https://bar.example.com` to a specific Docker container.

Note:

- Each service should have a unique name.
- Each service can optionally support HTTPS through TLS.
- For services that redirect to a remote URL, provide the `redirect` field.
- For services running in Docker containers, provide the `container` field.

### Middleware Configuration

Middlewares allow you to apply additional functionality to your services, such as IP filtering or authentication.
Multiple middlewares can be added to a single service.

Example configuration with middlewares:

```yaml
# config.yml
services:
  foo:
    host: example.com
    redirect: "https://example.com"
    middlewares:
      authForward: &customAuth # YAML anchor.
        address: "https://auth.example.com"
        xForwarded: true
        requestHeaders: []
        responseHeaders:
          ["Remote-User", "Remote-Groups", "Remote-Name", "Remote-Email"]
  bar:
    host: bar.example.com
    tls: true
    container:
      name: "/container1"
      network: "network1"
      port: 8080
    middlewares:
      authForward: *customAuth # YAML alias is used here to avoid repetition.
      ipAllow:
        - 127.0.0.1
        - 10.9.0.0/24 # CIDR notation is supported!
        - 192.168.1.7
```

### Logging Configuration

You can customize logging to your liking by changing the logger's verbosity (through level) and format (through handler):

```yaml
# config.yml
services:
  # ...

log:
  level: "info" # can be "debug", "info", "warn", or "error".
  handler: "text" # can be "text" or "json".
```

## 📝 Usage

You can run voltproxy locally or deploy with Docker.

### Deploying with Docker

1. Create voltproxy configuration `config.yml`. Example:

```yaml
# config.yml
services:
  foobar:
    host: foobar.example.com
    container:
      name: "/foobar"
      network: "service_net"
      port: 5173
```

2. Create Docker Compose file `docker-compose.yml`. Example:

```yaml
# docker-compose.yml
version: "3.3"

services:
  voltproxy:
    container_name: voltproxy
    build:
      context: .
    restart: unless-stopped
    ports:
      - 80:80
      - 443:443
    volumes:
      - "./_certs:/usr/src/voltproxy/_certs"
      - "./config.yml:/usr/src/voltproxy/config.yml"
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
    networks:
      # Only needed if proxying to a Docker container on a different network.
      - service_net
networks:
  service_net:
    external: true
```

3. Run container with Docker Compose:

```sh
$ docker compose up -d --force-recreate
```

## 🌟 Future Improvements

- More middlewares.
