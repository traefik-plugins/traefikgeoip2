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

We're using this plugin in Kubernetes, thus the configuration guide is Kubernetes oriented.

You are welcome: 

  * refer to the Traefik configuraiton documentation for other orchestration frameworks.
  * contribute to this repository about configuraion for other orchestration environments.

### Enable plugin in Traefik

We recommend to use [official Helm chart](https://github.com/traefik/traefik-helm-chart)
for installing Traefik into Kubernetes cluster.

Below, there's an instruction for adjusting 
[official Helm chart](https://github.com/traefik/traefik-helm-chart)
to install the plugin.

1. Create a file named `traefik.yaml`
   
    ```yaml
    additionalArguments:
      - "--experimental.plugins.geoip2.modulename=github.com/GiGInnovationLabs/traefikgeoip2"
      - "--experimental.plugins.geoip2.version=v0.1.1"
    ```
2. Install [Traefik Helm chart](https://github.com/traefik/traefik-helm-chart)
    ```
      helm repo add traefik https://helm.traefik.io/traefik
      helm repo update
      helm install my-traefik traefik/traefik --version 10.1.1 -f ./traefik.yaml      
    ```

### Create Traefik Middleware

!!! TO BE DEFINED

### Apply Traefik Middleware to Traefik route

!!! TO BE DEFINED

## Development

To run linter and tests execute this command

```sh
make prepare
make
```