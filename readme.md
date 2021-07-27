# MaxMind GeoIP2 plugin for Traefik

[Traefik](https://doc.traefik.io/traefik/) plugin 
that registers a custom middleware 
for getting data from 
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
  * contribute to this README about the configuraion for other orchestration environments.

### Create custom Traefik Docker image

This is ~required in Kubernetes, since MaxMind DB size is bigger, 
than data size allowed for `ConfigMap` or `Secret` resource.

Assuming you want to try free 
[GeoLite2](https://dev.maxmind.com/geoip/geolite2-free-geolocation-data)
database, that is already downloaded locally.

1. Create custom `Dockerfile`
  
    ```
    FROM traefik:2.4.9
    COPY GeoLite2-City.mmdb /var/lib/traefikgeoip2/ 
    ```

2. Build and publish to a Docker registry
   
    ```sh
    export TDR=${...}
    docker build -t ${TDR}/traefik:2.4.9 .
    docker push ${TDR}/traefik:2.4.9


### Enable plugin in Traefik

We recommend to use [official Helm chart](https://github.com/traefik/traefik-helm-chart)
for installing Traefik into Kubernetes cluster.

Below, there's an instruction for adjusting 
[official Helm chart](https://github.com/traefik/traefik-helm-chart)
to install the plugin.

1. Create a file named `traefik.yaml`, replacing `${TDR}` with actual Docker registry path. 
   
    ```yaml
    image:
      name: ${TDR}/traefik
      tag: "2.4.9"

    pilot:
      enabled: true
      token: "${TRAEFIK_PILOT_TOKEN}"

    additionalArguments:
      - "--experimental.plugins.geoip2.modulename=github.com/GiGInnovationLabs/traefikgeoip2"
      - "--experimental.plugins.geoip2.version=v0.1.1"
    ```
2. Install customized [Traefik Helm chart](https://github.com/traefik/traefik-helm-chart).

    ```sh
    helm repo add traefik https://helm.traefik.io/traefik
    helm repo update
    helm upgrade --install -n traefik --create-namespace \
      my-traefik traefik/traefik --version 10.1.1 -f ./traefik.yaml      
    ```

### Create Traefik Middleware

1. Create a file named `mw.yaml`
    ```yaml
    apiVersion: traefik.containo.us/v1alpha1
    kind: Middleware
    metadata:
      name: geoip2
      namespace: traefik
    spec:
      plugin:
        geoip:
          dbPath: "/var/lib/geoip2/GeoLite2-City.mmdb"
    ```

2. Apply
   
    `kubectl apply -f mw.yaml`

### Apply GeoIP2 middleware to Traefik route

!!! warning TO BE DEFINED 

## Development

To run linter and tests execute this command

```sh
make prepare
make
```