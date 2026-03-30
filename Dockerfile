FROM traefik:v3.3

# COPY *.yml *.mmdb go.* *.go /plugins/go/src/github.com/traefik-plugins/traefikgeoip2/
# COPY vendor/ /plugins/go/src/github.com/traefik-plugins/traefikgeoip2/vendor/

COPY GeoLite2-City.mmdb /var/lib/traefikgeoip2/
