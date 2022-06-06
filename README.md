# Traefik geoblocking middleware

Simple geoblocking plugin for Traefik allows or blocks HTTP request for the specified countries or CIDR.
This plugin does not relies on third party SaaS APIs.

This project relies IP2Location LITE data available from [`lite.ip2location.com`](https://lite.ip2location.com/database/ip-country) database
- Databases: [https://download.ip2location.com/lite](https://download.ip2location.com/lite/)


## Configuration

### Local

Add inside your `traefik.yml`:

```yml
providers:
  providersThrottleDuration: 2s
  file:
    filename: /etc/traefik/dynamic-configuration.yml

experimental:
  localPlugins:
    geoblock:
      moduleName: github.com/mdouchement/geoblock
```

### Dynamic

Add inside your `dynamic-configuration.yml`:

```yml
http:
  middlewares:
    my-geoblock:
      plugin:
        geoblock:
          enabled: true
          allowLetsEncrypt: true
          databases:
          - /plugins-local/src/github.com/mdouchement/geoblock/IP2LOCATION-LITE-DB1.IPV6.BIN
          - /plugins-local/src/github.com/mdouchement/geoblock/IP2LOCATION-LITE-DB1.BIN
          # Or use default assets stored inside the code
          # - IP2LOCATION-LITE-DB1.IPV6.BIN
          # - IP2LOCATION-LITE-DB1.BIN
          defaultAction: block
          allowlist:
          - type: country
            value: FR
          blocklist:
          - type: cidr
            value: 127.0.0.0/8 # IPv4 loopback
          - type: cidr
            value: 10.0.0.0/8 # RFC1918
          - type: cidr
            value: 172.16.0.0/12 # RFC1918
          - type: cidr
            value: 192.168.0.0/16 # RFC1918
          - type: cidr
            value: 169.254.0.0/16 # RFC3927 link-local
          - type: cidr
            value: ::1/128 # IPv6 loopback
          - type: cidr
            value: fe80::/10 # IPv6 link-local
          - type: cidr
            value: fc00::/7 # IPv6 unique local addr
```

### Docker Compose

Add inside your `docker-compose.yml`:

```yml
version: "3.7"

services:
  traefik:
    image: traefik
    ports:
      - 80:80
    volumes:
    - /var/run/docker.sock:/var/run/docker.sock
    - /opt/traefik/traefik.yml:/etc/traefik/traefik.yml:ro
    - /opt/traefik/dynamic-configuration.yml:/etc/traefik/dynamic-configuration.yml:ro
    - /plugin/geoblock:/plugins-local/src/github.com/mdouchement/geoblock
```

### Usage

- Globally in `traefik.yml` by customizing entrypoints:
    ```yml
    # https://doc.traefik.io/traefik/routing/entrypoints/
    entryPoints:
    web:
        address: ":80"
        http:
        middlewares:
            - my-geoblock@file
    websecure:
        address: ":443"
        http:
        middlewares:
            - my-geoblock@file
    ```

- With labels for a specific service (docker-compose.yml):
    ```yml
    version: "3.7"

    services:
    hello:
        image: containous/whoami
        labels:
        - traefik.enable=true
        - traefik.http.routers.hello.entrypoints=http
        - traefik.http.routers.hello.rule=Host(`hello.localhost`)
        - traefik.http.services.hello.loadbalancer.server.port=80
        #
        - traefik.http.routers.hello.middlewares=my-geoblock@file
    ```

## License

**MIT**


## Contributing

All PRs are welcome.

1. Fork it
2. Create your feature branch (git checkout -b my-new-feature)
3. Commit your changes (git commit -am 'Add some feature')
5. Push to the branch (git push origin my-new-feature)
6. Create new Pull Request