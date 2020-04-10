DNS-over-HTTPS
==============

Client and server software to query DNS over HTTPS, using [Google DNS-over-HTTPS protocol](https://developers.google.com/speed/public-dns/docs/dns-over-https)
and [IETF DNS-over-HTTPS (RFC 8484)](https://www.rfc-editor.org/rfc/rfc8484.txt).

## Guides

- [Tutorial: Setup your own DNS-over-HTTPS (DoH) server](https://www.aaflalo.me/2018/10/tutorial-setup-dns-over-https-server/). (Thanks to Antoine Aflalo)
- [Tutorial: Setup your own Docker based DNS-over-HTTPS (DoH) server](https://github.com/satishweb/docker-doh/blob/master/README.md). (Thanks to Satish Gaikwad)

## Installing
### From Source
- Install [Go](https://golang.org), at least version 1.10.
> Note for Debian/Ubuntu users: You need to set `$GOROOT` if you could not get your new version of Go selected by the Makefile.)

- First create an empty directory, used for `$GOPATH`:
```bash
mkdir ~/gopath
export GOPATH=~/gopath
```
- To build the program, type:
```bash
make
```
- To install DNS-over-HTTPS as Systemd services, type:
```bash
sudo make install
```
- By default, [Google DNS over HTTPS](https://dns.google.com) is used. It should
work for most users (except for People's Republic of China). If you need to
modify the default settings, type:
```bash
sudoedit /etc/dns-over-https/doh-client.conf
```
- To automatically start DNS-over-HTTPS client as a system service, type:
```bash
sudo systemctl start doh-client.service
sudo systemctl enable doh-client.service
```
- Then, modify your DNS settings (usually with NetworkManager) to 127.0.0.1.

- To test your configuration, type:
```bash
dig www.google.com
Output:
;; SERVER: 127.0.0.1#53(127.0.0.1)
```
#### Uninstall

- To uninstall, type:
```bash
sudo make uninstall
```
> Note: The configuration files are kept at `/etc/dns-over-https`. Remove them manually if you want.

### Using docker image
```bash
docker run -itd --name doh-server \
  -p 8053:8053 \
  -e UPSTREAM_DNS_SERVER="udp:8.8.8.8:53" \
  -e DOH_HTTP_PREFIX="/dns-query"
  -e DOH_SERVER_LISTEN=":8053"
  -e DOH_SERVER_TIMEOUT="10"
  -e DOH_SERVER_TRIES="3"
  -e DOH_SERVER_VERBOSE="false"
satishweb/doh-server
```

## Server Configuration

The following is a typical DNS-over-HTTPS architecture:

    +--------------+                                +------------------------+
    | Application  |                                |  Recursive DNS Server  |
    +-------+------+                                +-----------+------------+
            |                                                   |
    +-------+------+                                +-----------+------------+
    | Client side  |                                |      doh-server        |
    | cache (nscd) |                                +-----------+------------+
    +-------+------+                                            |
            |         +--------------------------+  +-----------+------------+
    +-------+------+  |    HTTP cache server /   |  |   HTTP service muxer   |
    |  doh-client  +--+ Content Delivery Network +--+ (Apache, Nginx, Caddy) |
    +--------------+  +--------------------------+  +------------------------+

Although DNS-over-HTTPS can work alone, a HTTP service muxer would be useful as
you can host DNS-over-HTTPS along with other HTTPS services.

HTTP/2 with at least TLS v1.3 is recommended. OCSP stapling must be enabled,
otherwise DNS recursion may happen.

### Configuration file

The main configuration file is `doh-client.conf`.

**Server selectors.** If several upstream servers are set, one is selected according to `upstream_selector` for each request. With `upstream_selector = "random"`, a random upstream server will be chosen for each request.

```toml
# available selector: random (default) or weighted_round_robin or lvs_weighted_round_robin
upstream_selector = "random"
```

### Example configuration: Apache
```bash
SSLProtocol TLSv1.2
SSLHonorCipherOrder On
SSLCipherSuite ECDH+AESGCM:DH+AESGCM:ECDH+AES256:DH+AES256:ECDH+AES128:DH+AES:ECDH+3DES:DH+3DES:RSA+3DES:!aNULL:!MD5:!DSS:!eNULL:!EXP:!LOW:!MD5
SSLUseStapling on
SSLStaplingCache shmcb:/var/lib/apache2/stapling_cache(512000)

<VirtualHost *:443>
    ServerName MY_SERVER_NAME
    Protocols h2 http/1.1
    ProxyPass /dns-query http://[::1]:8053/dns-query
    ProxyPassReverse /dns-query http://[::1]:8053/dns-query
</VirtualHost>
```
(Credit: [Joan Moreau](https://github.com/m13253/dns-over-https/issues/51#issuecomment-526820884))

### Example configuration: Nginx
```bash
server {
  listen       443 ssl http2 default_server;
  listen       [::]:443 ssl http2 default_server;
  server_name  MY_SERVER_NAME;

  server_tokens off;

  ssl_protocols TLSv1.2 TLSv1.3;          # TLS 1.3 requires nginx >= 1.13.0
  ssl_prefer_server_ciphers on;
  ssl_dhparam /etc/nginx/dhparam.pem;     # openssl dhparam -dsaparam -out /etc/nginx/dhparam.pem 4096
  ssl_ciphers EECDH+AESGCM:EDH+AESGCM;
  ssl_ecdh_curve secp384r1;               # Requires nginx >= 1.1.0
  ssl_session_timeout  10m;
  ssl_session_cache shared:SSL:10m;
  ssl_session_tickets off;                # Requires nginx >= 1.5.9
  ssl_stapling on;                        # Requires nginx >= 1.3.7
  ssl_stapling_verify on;                 # Requires nginx => 1.3.7
  ssl_early_data off;                     # 0-RTT, enable if desired - Requires nginx >= 1.15.4
  resolver 1.1.1.1 valid=300s;            # Replace with your local resolver
  resolver_timeout 5s;
  # HTTP Security Headers
  add_header X-Frame-Options DENY;
  add_header X-Content-Type-Options nosniff;
  add_header X-XSS-Protection "1; mode=block";
  add_header Strict-Transport-Security "max-age=63072000";
  ssl_certificate /path/to/your/server/certificates/fullchain.pem;
  ssl_certificate_key /path/to/your/server/certificates/privkey.pem;
  location /dns-query {
    proxy_pass       http://localhost:8053/dns-query;
    proxy_set_header Host      $host;
    proxy_set_header X-Real-IP $remote_addr;
  }
}
```
(Credit: [Cipherli.st](https://cipherli.st/))

### Example configuration: Caddy (v2)
```bash
my.server.name {
        reverse_proxy * localhost:8053
        tls your@email.address
        try_files {path} {path}/index.php /index.php?{query}
}
```
### Example configuration: Docker Compose + Traefik + Unbound (Raspberry Pi/Linux/Mac) [linux/amd64,linux/arm64,linux/arm/v7]

```yaml
version: '2.2'
networks:
  default:
services:
  proxy:
    # The official v2 Traefik docker image
    image: traefik:v2.2
    hostname: proxy
    networks:
      - default
    environment:
      TRAEFIK_ACCESSLOG: "true"
      TRAEFIK_API: "true"
      TRAEFIK_PROVIDERS_DOCKER: "true"
      TRAEFIK_API_INSECURE: "true"
      TRAEFIK_PROVIDERS_DOCKER_NETWORK: "${STACK}_default"
      # DNS provider specific environment variables for DNS Challenge using route53 (AWS)
      AWS_ACCESS_KEY_ID: ${AWS_ACCESS_KEY_ID}
      AWS_SECRET_ACCESS_KEY: ${AWS_SECRET_ACCESS_KEY}
      AWS_REGION: ${AWS_REGION}
      AWS_HOSTED_ZONE_ID: ${AWS_HOSTED_ZONE_ID}
    ports:
      # The HTTP port
      - "80:80"
      # The HTTPS port
      - "443:443"
      # The Web UI (enabled by --api.insecure=true)
      - "8080:8080"
    command:
      #- "--log.level=DEBUG"
      - "--providers.docker.exposedbydefault=false"
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--certificatesresolvers.letsencrypt.acme.dnschallenge=true"
      # Providers list:
      #  https://docs.traefik.io/https/acme/#providers
      #  https://go-acme.github.io/lego/dns/
      - "--certificatesresolvers.letsencrypt.acme.dnschallenge.provider=route53"
      # Enable below line to use staging letsencrypt server.
      #- "--certificatesresolvers.letsencrypt.acme.caserver=https://acme-staging-v02.api.letsencrypt.org/directory"
      - "--certificatesresolvers.letsencrypt.acme.email=${EMAIL}"
      - "--certificatesresolvers.letsencrypt.acme.storage=/certs/acme.json"
    volumes:
      # So that Traefik can listen to the Docker events
      - /var/run/docker.sock:/var/run/docker.sock
      - ./data/proxy/certs:/certs
  doh-server:
    image: satishweb/doh-server:latest
    hostname: doh-server
    networks:
      - default
    environment:
      # Enable below line to see more logs
      # DEBUG: "1"
      UPSTREAM_DNS_SERVER: "udp:unbound:53"
      DOH_HTTP_PREFIX: "${DOH_HTTP_PREFIX}"
      DOH_SERVER_LISTEN: ":${DOH_SERVER_LISTEN}"
      DOH_SERVER_TIMEOUT: "10"
      DOH_SERVER_TRIES: "3"
      DOH_SERVER_VERBOSE: "false"
    #volumes:
      # - ./doh-server.conf:/server/doh-server.conf
      # - ./app-config:/app-config
    depends_on:
      - unbound
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.doh-server.rule=Host(`${SUBDOMAIN}.${DOMAIN}`) && Path(`${DOH_HTTP_PREFIX}`)"
      - "traefik.http.services.doh-server.loadbalancer.server.port=${DOH_SERVER_LISTEN}"
      - "traefik.http.middlewares.mw-doh-compression.compress=true"
      - "traefik.http.routers.doh-server.tls=true"
      - "traefik.http.middlewares.mw-doh-tls.headers.sslredirect=true"
      - "traefik.http.middlewares.mw-doh-tls.headers.sslforcehost=true"
      - "traefik.http.routers.doh-server.tls.certresolver=letsencrypt"
      - "traefik.http.routers.doh-server.tls.domains[0].main=${DOMAIN}"
      - "traefik.http.routers.doh-server.tls.domains[0].sans=${SUBDOMAIN}.${DOMAIN}"
      # Protection from requests flood
      - "traefik.http.middlewares.mw-doh-ratelimit.ratelimit.average=100"
      - "traefik.http.middlewares.mw-doh-ratelimit.ratelimit.burst=50"
      - "traefik.http.middlewares.mw-doh-ratelimit.ratelimit.period=10s"
  unbound:
    image: satishweb/unbound:latest
    hostname: unbound
    networks:
      - default
    ports:
      # Disable these ports if DOH server is the only client
      - 53:53/tcp
      - 53:53/udp
    volumes:
      - ./unbound.sample.conf:/templates/unbound.sample.conf
      - ./data/unbound/custom:/etc/unbound/custom
      # Keep your custom.hosts file inside custom folder
    #environment:
    #  DEBUG: "1"
````

> Complete Guide available at: https://github.com/satishweb/docker-doh

> No IPV6 Support: Docker Swarm does not support IPV6 as of yet. Issue is logged [here](https://github.com/moby/moby/issues/24379)

> IPV6 Support for Docker Compose based configuration TBA

## DNSSEC

DNS-over-HTTPS is compatible with DNSSEC, and requests DNSSEC signatures by
default. However signature validation is not built-in. It is highly recommended
that you install `unbound` or `bind` and pass results for them to validate DNS
records. An instance of [Pi Hole](https://pi-hole.net) could also be used to validate DNS signatures as well as provide other capabilities.

## EDNS0-Client-Subnet (GeoDNS)

DNS-over-HTTPS supports EDNS0-Client-Subnet protocol, which submits part of the
client's IP address (/24 for IPv4, /56 for IPv6 by default) to the upstream
server. This is useful for GeoDNS and CDNs to work, and is exactly the same
configuration as most public DNS servers.

Keep in mind that /24 is not enough to track a single user, although it is
precise enough to know the city where the user is located. If you think
EDNS0-Client-Subnet is affecting your privacy, you can set `no_ecs = true` in
`/etc/dns-over-https/doh-client.conf`, with the cost of slower video streaming
or software downloading speed.

To ultilize ECS, `X-Forwarded-For` or `X-Real-IP` should be enabled on your
HTTP service muxer. If your server is backed by `unbound` or `bind`, you
probably want to configure it to enable the EDNS0-Client-Subnet feature as
well.

## Protocol compatibility

### Google DNS-over-HTTPS Protocol

DNS-over-HTTPS uses a protocol compatible to [Google DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https),
except for absolute expire time is preferred to relative TTL value. Refer to
[json-dns/response.go](json-dns/response.go) for a complete description of the
API.

### IETF DNS-over-HTTPS Protocol

DNS-over-HTTPS uses a protocol compatible to [IETF DNS-over-HTTPS (RFC 8484)](https://www.rfc-editor.org/rfc/rfc8484.txt).

### Supported features

Currently supported features are:

- [X] IPv4 / IPv6
- [X] EDNS0 large UDP packet (4 KiB by default)
- [X] EDNS0-Client-Subnet (/24 for IPv4, /56 for IPv6 by default)

## The name of the project

This project is named "DNS-over-HTTPS" because it was written before the IETF DoH project. Although this project is compatible with IETF DoH, the project is not affiliated with IETF.

To avoid confusion, you may also call this project "m13253/DNS-over-HTTPS" or anything you like.

## License

DNS-over-HTTPS is licensed under the [MIT License](LICENSE). You are encouraged
to embed DNS-over-HTTPS into your other projects, as long as the license
permits.

You are also encouraged to disclose your improvements to the public, so that
others may benefit from your modification, in the same way you receive benefits
from this project.
