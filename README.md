# ⚡ voltproxy - Configurable Reverse Proxy

voltproxy is a reverse proxy designed to simplify the process of proxying Docker containers and other services.
With voltproxy, you can easily manage and redirect traffic to different services using a single YAML configuration file.

## 📝 Usage

To use voltproxy, you need to create a YAML configuration file that specifies the services to be proxied.
The configuration file should follow a specific format.

## 🔧 Configuration Format

Example configuration file:

```yaml
services:
  foo:
    host: example.com
    redirect: "https://example.com"
    middlewares:
      ipallow:
        - 127.0.0.1
        - 10.9.0.0/24 # CIDR notation is supported!
        - 192.168.1.7
  bar:
    host: bar.example.com
    tls: true
    container:
      name: "/container1"
      network: "network1"
      port: 8080
  baz:
    host: baz.example.com
    tls: true
    redirect: protected.example.com
    # Multiple middlewares can be added to a single service.
    middlewares:
      ipallow:
        - 10.9.0.1
      authforward:
        address: "https://auth.example.com"
        xforwarded: true
        requestheaders: []
        responseheaders:
          ["Remote-User", "Remote-Groups", "Remote-Name", "Remote-Email"]
```

- Each service should have a unique name.
- Each service can optionally support HTTPS through TLS.
- For services that redirect to a remote URL, provide the `redirect` field.
- For services running in Docker containers, provide the `container` field.
  The `container` field should include the name, network, and port of the container.
- Middlewares can optionally be attached to any service.

## 🌟 Future Improvements

- More middlewares.
- Enhanced logging and monitoring capabilities.
