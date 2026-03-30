# Traefik plugin for MaxMind GeoIP2

> [!IMPORTANT]
> This project is looking for maintainers. Please comment in [this issue](https://github.com/traefik-plugins/traefik-jwt-plugin/issues/82)

[Traefik](https://doc.traefik.io/traefik/) plugin 
that registers a custom middleware 
for getting data from 
[MaxMind GeoIP databases](https://www.maxmind.com/en/geoip2-services-and-databases) 
and pass it downstream via HTTP request headers.

Supports both 
[GeoIP2](https://www.maxmind.com/en/geoip2-databases) 
and 
[GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) databases.

## Features

- Country, region, city and **postal/ZIP code** enrichment via request headers
- Configurable **private / LAN IP override** — skips the DB lookup entirely and returns
  operator-defined location values instead
- New `X-GeoIP2-PrivateIP` header flags whether the lookup was performed or bypassed
- **File-based access log** — one structured line per request, written to a separate log file
- **Hot-reload** — Traefik periodically checks whether the `.mmdb` file has been replaced
  on disk and swaps in the new reader transparently, with no restart required
- Reverse-proxy aware — optionally reads the client IP from `X-Forwarded-For` (e.g. behind Cloudflare)
- Panic-safe DB loading — corrupt or truncated `.mmdb` files are caught and reported as
  errors rather than crashing the middleware

---

## Response Headers

The following headers are set on every request passed through this middleware:

| Header | Description |
|---|---|
| `X-GeoIP2-Country` | ISO 3166-1 alpha-2 country code (e.g. `DE`). `XX` if unknown. |
| `X-GeoIP2-Region` | ISO 3166-2 subdivision code (e.g. `BW`). `XX` if unknown or Country DB. |
| `X-GeoIP2-City` | English city name (e.g. `Stuttgart`). `XX` if unknown or Country DB. |
| `X-GeoIP2-PostalCode` | Postal / ZIP code (e.g. `70173`). `XX` if unknown or Country DB. |
| `X-GeoIP2-IPAddress` | Resolved client IP that was used for the lookup. |
| `X-GeoIP2-PrivateIP` | `true` when the IP is private/LAN/loopback and no DB lookup was done. |

> **Note:** Postal codes are only available with a `GeoLite2-City.mmdb` or `GeoIP2-City.mmdb`
> database. The Country DB always returns `XX` for region, city and postal code.

---

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

### All configuration options

| Option | Type | Default | Description |
|---|---|---|---|
| `dbPath` | `string` | `GeoLite2-Country.mmdb` | **Required.** Path to the MaxMind `.mmdb` database file inside the container. The filename must contain either `City` or `Country` so the correct reader is selected. |
| `preferXForwardedForHeader` | `bool` | `false` | Use the first IP from `X-Forwarded-For` instead of `RemoteAddr`. Enable when Traefik is behind a proxy such as Cloudflare. |
| `privateIPCountry` | `string` | `XX` | ISO 3166-1 alpha-2 country code to return for private/LAN/loopback IPs. |
| `privateIPRegion` | `string` | `XX` | ISO 3166-2 region code to return for private/LAN/loopback IPs. |
| `privateIPCity` | `string` | `XX` | City name to return for private/LAN/loopback IPs. |
| `privateIPPostalCode` | `string` | `XX` | Postal/ZIP code to return for private/LAN/loopback IPs. |
| `dbRefreshInterval` | `int` | `0` | How often in **seconds** to check whether the `.mmdb` file has been replaced on disk and reload it. `0` disables automatic refresh. |
| `logFilePath` | `string` | _(disabled)_ | Path to a file where one log line per request is appended. If empty, no file logging is done (Traefik's own log still receives error messages). |

#### Private IP ranges

The following address ranges are considered private and bypass the GeoIP lookup:

- `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16` (RFC 1918)
- `127.0.0.0/8` (loopback)
- `169.254.0.0/16` (link-local)
- `::1` (IPv6 loopback)
- `fc00::/7` (IPv6 unique local)
- `fe80::/10` (IPv6 link-local)

---

## Log File Format

When `logFilePath` is configured, each processed request appends one line:

```
2026-03-30T11:05:23Z ip=1.2.3.4 country=DE region=BW city=Stuttgart postal=70173 private=false
2026-03-30T11:05:24Z ip=192.168.1.5 country=DE region=BW city=Stuttgart postal=70173 private=true
```

Fields:

| Field | Description |
|---|---|
| timestamp | RFC 3339 UTC timestamp |
| `ip` | Resolved client IP |
| `country` | ISO 3166-1 alpha-2 code or `XX` |
| `region` | ISO 3166-2 code or `XX` |
| `city` | City name or `XX` |
| `postal` | Postal/ZIP code or `XX` |
| `private` | `true` / `false` |

---

## DB Hot-Reload

When `dbRefreshInterval` is set to a value greater than `0`, the middleware checks on each
incoming request whether the configured interval has elapsed since the last check. If so,
it stats the file and compares the modification time. Only when the file has actually
changed is the database re-read from disk — the new reader is swapped in atomically under a
`sync.RWMutex` so in-flight requests are never affected.

This allows automated MaxMind database updates (e.g. via a cron job or
[geoipupdate](https://github.com/maxmind/geoipupdate)) to take effect without restarting
Traefik.

**Example cron + config combination:**

```yaml
# Update DB every Wednesday at 02:00, check for changes every hour
dbRefreshInterval: 3600
```

```cron
0 2 * * 3  geoipupdate
```

---

## Development

### Requirements

- Go 1.25+
- MaxMind GeoLite2 database files placed in the repository root for tests
  (`GeoLite2-City.mmdb`, `GeoLite2-Country.mmdb`)

### Run tests

```sh
go test ./...
```

### Run linter

```sh
golangci-lint run
```

---

## Changelog

### v2.1.0

- **New:** `X-GeoIP2-PostalCode` header (populated from `GeoLite2-City.mmdb`)
- **New:** `X-GeoIP2-PrivateIP` header — `true` when IP is private/LAN/loopback
- **New:** Private IP override via `privateIPCountry` / `privateIPRegion` / `privateIPCity` / `privateIPPostalCode` config fields — private IPs now skip the DB lookup entirely instead of generating lookup errors
- **New:** `logFilePath` config field — structured per-request log file
- **New:** `dbRefreshInterval` config field — hot-reload of the `.mmdb` file without Traefik restart
- **Fix:** Private IPs previously caused spurious lookup errors in the Traefik log
- **Fix:** `X-GeoIP2-IPAddress` was incorrectly set to `XX` when the DB was unavailable; it now always reflects the resolved client IP
- **Fix:** `isPrivateIP` now also covers loopback (`127.x`, `::1`) and link-local (`169.254.x`, `fe80::`) addresses
- **Fix:** Global `lookup` variable replaced by per-instance field with `sync.RWMutex` — multiple middleware instances no longer share state
- **Chore:** `ioutil.ReadFile` replaced with `os.ReadFile` across all vendor reader files (deprecated since Go 1.16)
- **Chore:** Go version updated to 1.25

```
