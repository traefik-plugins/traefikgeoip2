FROM traefik:2.4.9

# COPY *.yml *.mmdb go.* ./geoip.go /plugins/go/src/github.com/GiGInnovationLabs/traefikgeoip2/
# COPY vendor/ /plugins/go/src/github.com/GiGInnovationLabs/traefikgeoip2/vendor/

COPY GeoLite2-City.mmdb /var/lib/traefikgeoip2/ 

