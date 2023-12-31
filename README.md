![voltproxy Logo](./voltproxy.png)

<div align="center">

[**Website**](https://voltproxy.plam.dev/) |
[**Documentation**](https://voltproxy.plam.dev/docs/getting-started)

</div>

# ⚡ voltproxy

voltproxy is a **reverse proxy** designed to simplify the process of proxying Docker containers and other services.
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
  foo:
    host: foo.plam.dev
    redirect: "http://192.168.0.1:3000"
  bar:
    host: bar.plam.dev
    tls: true
    container:
      name: "/bar"
      network: "bar_default"
      port: 8080
```

This configuration instructs voltproxy to proxy incoming requests with the URL `http://foo.plam.dev` to `http://192.168.0.1:3000` and proxy incoming requests with the URL `https://bar.plam.dev` to the specified Docker container.

See the [documentation](https://voltproxy.plam.dev/docs/getting-started) for more details.

### Examples

These examples can be found in [integration/examples](./integration/examples/).

- 🔧 [Basic Configuration](./integration/examples/basic.yml)
- ⚖️ [Load Balancing](./integration/examples/load-balancer.yml)
- 🏥 [Health Checking](./integration/examples/health-check.yml)
- 🔗 [Multiple Middlewares](./integration/examples/multiple-middlewares.yml)
- ➕ [Additional Configuration](./integration/examples/additional-configuration.yml)

#### Middleware Configuration

- 🔑 [Auth Forward](./integration/examples/middlewares/auth-forward.yml)
- 🔒 [IP Allow](./integration/examples/middlewares/auth-forward.yml)

####

## 📝 Usage

You can either run voltproxy locally or deploy with Docker.

### Local Installation

Ensure you have [Go](https://go.dev/doc/install) 1.21 or newer.

1. Clone the repository:

```sh
$ git clone "https://github.com/plamorg/voltproxy.git"
```

2. Build voltproxy:

```sh
$ cd voltproxy/
$ go build
```

### Deploying with Docker

1. Create voltproxy configuration `config.yml`.

2. Create Docker Compose file `docker-compose.yml`. Example:

```yaml
# docker-compose.yml
version: "3.3"

services:
  voltproxy:
    container_name: voltproxy
    image: claby2/voltproxy:latest
    restart: unless-stopped
    ports:
      - 80:80
      - 443:443
    volumes:
      - "./_certs:/usr/src/voltproxy/_certs"
      - "./config.yml:/usr/src/voltproxy/config.yml"
      - "/var/run/docker.sock:/var/run/docker.sock:ro"
    networks:
      # If proxying a Docker container, ensure the containers are connected to the same network.
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
