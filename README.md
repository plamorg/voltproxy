# ⚡ voltproxy - Configurable Reverse Proxy

voltproxy is a reverse proxy designed to simplify the process of proxying Docker containers and other services.
With voltproxy, you can easily manage and redirect traffic to different services using a single YAML configuration file.

### Features:

- **Single configuration** for all your services with YAML.
- **Streamlined** Docker integration.
  - Simply provide a Docker container's name and network and voltproxy will do the rest.
  - No need to define per-container labels.
- **Automatic HTTPS** with support for ACME-based certificates.
- **Load Balancing** to enhance service scalability.
  - Customize service selection strategy.
  - Optionally persist client sessions through cookies.
- **Health Checking** functionality to facilitate failover schemes.
- **Middlewares** to attach additional functionality to existing services.
- **Customized structured logging** options to provide detailed logs for monitoring.

## 🔧 Configuration

Here is a simple configuration to get started:

```yaml
# config.yml
services:
  plam:
    host: example.plam.dev
    redirect: "https://example.com"
  bar:
    host: bar.plam.dev
    tls: true
    container:
      name: "/bar"
      network: "bar_net"
      port: 8080
```

This configuration instructs voltproxy to proxy incoming requests with the URL `http://example.plam.dev` to <https://example.com> and proxy incoming requests with the URL `https://bar.plam.dev` to the specified Docker container.

### Configuration examples

These examples can be found in [integration/examples](./integration/examples/).
They are listed here for convenience:

- 🔧 [Basic configuration](./integration/examples/basic.yml)
- ⚖️ [Load Balancing](./integration/examples/load-balancer.yml)
- 🏥 [Health Checking](./integration/examples/health-check.yml)
- 🔗 [Multiple Middlewares](./integration/examples/multiple-middlewares.yml)
- ➕[Additional configuration options](./integration/examples/additional-configuration.yml)

#### Middleware Configuration

- 🔑 [Auth Forward](./integration/examples/middlewares/auth-forward.yml)
- 🔒 [IP Allow](./integration/examples/middlewares/auth-forward.yml)

####

## 📝 Usage

You can either run voltproxy locally or deploy with Docker.

### Deploying with Docker

1. Create voltproxy configuration `config.yml`.

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
      # If proxying to another Docker container, ensure the containers are on this same network.
      # This is optional if you are not proxying another Docker container.
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

- Additional load balancing selection strategies.
- More middlewares.
