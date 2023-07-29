# ⚡ voltproxy - Configurable Reverse Proxy

voltproxy is a reverse proxy designed to simplify the process of proxying Docker containers and other services.
With voltproxy, you can easily manage and redirect traffic to different services using a single YAML configuration file.

## 📝 Usage

To use voltproxy, you need to create a YAML configuration file that specifies the services to be proxied.
The configuration file should follow a specific format.

## 🔧 Configuration Format

The configuration file should have the following structure:

```yaml
services:
  service1:
    host: example.com
    redirect: https://example.com
  service2:
    host: service2.com
    tls: true
    container:
      name: container1
      network: network1
      port: 8080
```

- Each service should have a unique name.
- Each service can optionally support HTTPS through TLS.
- For services that redirect to a remote URL, provide the `redirect` field.
- For services running in Docker containers, provide the `container` field.
  The `container` field should include the name, network, and port of the container.

## 🌟 Future Improvements

- Support for custom middleware.
- Enhanced logging and monitoring capabilities.
