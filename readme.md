# Traefik GeoIP2 plugin

[Traefik](https://doc.traefik.io/traefik/) plugin 
that allows to create a custom middleware 
for getting data from local 
[MaxMind GeoIP databases](https://www.maxmind.com/en/geoip2-services-and-databases) 
and pass it downstream via HTTP request headers.

Supports both 
[GeoIP2](https://www.maxmind.com/en/geoip2-databases) 
and 
[GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data) databases.

## Configuration

### Enable plugin in Traefik

!!! TO BE DEFINED

### Create a Middleware

!!! TO BE DEFINED

### Apply middleware to the Traefik route

!!! TO BE DEFINED

## Development

To run linter and tests execute this command

```sh
make prepare
make
```