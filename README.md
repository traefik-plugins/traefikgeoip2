# Traefik plugin for MaxMind GeoIP2

[Traefik](https://doc.traefik.io/traefik/) plugin 
that registers a custom middleware 
for getting data from 
[MaxMind GeoIP databases](https://www.maxmind.com/en/geoip2-services-and-databases) 
and pass it downstream via HTTP request headers.

Supports both 
[GeoIP2](https://www.maxmind.com/en/geoip2-databases) 
and 
[GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) databases.

## Installation 

The tricky part of installing this plugin into containerized environments, like Kubernetes,
is that a container should contain a database within it.

### Kubernetes

> [!WARNING]
> Setup below is provided for demonstration purpose and should not be used on production.
> Traefik's plugin site is observed to be frequently unavailable, 
> so plugin download may fail on pod restart.

Tested with [official Traefik chart](https://artifacthub.io/packages/helm/traefik/traefik) version 26.0.0.

The following snippet should be added to `values.yaml`:

```yaml
experimental:
  plugins:
    geoip2:
      moduleName: github.com/traefik-plugins/traefikgeoip2
      version: v0.22.0
deployment:
  additionalVolumes:
    - name: geoip2
      emptyDir: {}
  initContainers:
    - name: download
      image: alpine
      volumeMounts:
        - name: geoip2
          mountPath: /tmp/geoip2
      command:
        - "/bin/sh"
        - "-ce"
        - |
          wget -P /tmp https://raw.githubusercontent.com/traefik-plugins/traefikgeoip2/main/geolite2.tgz
          tar --directory /tmp/geoip2 -xvzf /tmp/geolite2.tgz
additionalVolumeMounts:
  - name: geoip2
    mountPath: /geoip2
```

### Create Traefik Middleware

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: geoip2
  namespace: traefik
spec:
  plugin:
    geoip2:
      dbPath: "/geoip2/GeoLite2-City.mmdb"
```

## Development

To run linter and tests execute this command

```sh
just test
```
