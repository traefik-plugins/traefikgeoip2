# Traefik GeoIP2 plugin

Plugin for getting information from MaxMind geo database and pass it to request headers.
Supports GeoIP2 and GeoLite2 databases.

## Usage

### Configuration

For each plugin, the Traefik static configuration must define the module name (as is usual for Go packages).

The following declaration (given here in YAML) defines an plugin:

```yaml
# Static configuration
pilot:
  token: xxxxx

experimental:
  plugins:
    geoip:
      moduleName: github.com/GiGInnovationLabs/traefik-geoip2
      version: v0.1.0
```

Here is an example of a file provider dynamic configuration (given here in YAML), where the interesting part is the `http.middlewares` section:

```yaml
# Dynamic configuration

http:
  routers:
    my-router:
      rule: host(`demo.localhost`)
      service: service-foo
      entryPoints:
        - web
      middlewares:
        - geoip

  services:
    service-foo:
      loadBalancer:
        servers:
          - url: http://127.0.0.1:5000
  
  middlewares:
    my-plugin:
      plugin:
        geoip:
          dbPath: ./GeoLite2-Country.mmdb
```

## Development

To run linter and tests - execute

```sh
make prepare
make
```
