DNS-over-HTTPS
==============

Client and server software to query DNS over HTTPS, using [Google DNS-over-HTTPS protocol](https://developers.google.com/speed/public-dns/docs/dns-over-https)
and [IETF DNS-over-HTTPS (RFC 8484)](https://www.rfc-editor.org/rfc/rfc8484.txt).

## Guide

[Tutorial to setup your own DNS-over-HTTPS (DoH) server](https://www.aaflalo.me/2018/10/tutorial-setup-dns-over-https-server/). (Thanks to Antoine Aflalo)

## Installing
### From Source
Install [Go](https://golang.org), at least version 1.10.

(Note for Debian/Ubuntu users: You need to set `$GOROOT` if you could not get your new version of Go selected by the Makefile.)

First create an empty directory, used for `$GOPATH`:

    mkdir ~/gopath
    export GOPATH=~/gopath

To build the program, type:

    make

To install DNS-over-HTTPS as Systemd services, type:

    sudo make install

By default, [Google DNS over HTTPS](https://dns.google.com) is used. It should
work for most users (except for People's Republic of China). If you need to
modify the default settings, type:

    sudoedit /etc/dns-over-https/doh-client.conf

To automatically start DNS-over-HTTPS client as a system service, type:

    sudo systemctl start doh-client.service
    sudo systemctl enable doh-client.service

Then, modify your DNS settings (usually with NetworkManager) to 127.0.0.1.

To test your configuration, type:

    dig www.google.com

If it is OK, you will see:

    ;; SERVER: 127.0.0.1#53(127.0.0.1)

#### Uninstall

To uninstall, type:

    sudo make uninstall

The configuration files are kept at `/etc/dns-over-https`. Remove them manually if you want.

### Using docker image
```
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

(Credit: [Joan Moreau](https://github.com/m13253/dns-over-https/issues/51#issuecomment-526820884))

### Example configuration: Nginx

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

(Credit: [Cipherli.st](https://cipherli.st/))

### Example configuration: Caddy

    https://MY_SERVER_NAME {
            log     / syslog "{remote} - {user} [{when}] \"{method} {scheme}://{host}{uri} {proto}\" {status} {size} \"{>Referer}\" \"{>User-Agent}\" {>X-Forwarded-For}"
            errors  syslog
            gzip
            proxy   /dns-query      http://[::1]:18053 {
                    header_upstream Host {host}
                    header_upstream X-Real-IP {remote}
                    header_upstream X-Forwarded-For {>X-Forwarded-For},{remote}
                    header_upstream X-Forwarded-Proto {scheme}
            }
            root    /var/www
            tls {
                    ciphers ECDHE-ECDSA-WITH-CHACHA20-POLY1305 ECDHE-RSA-WITH-CHACHA20-POLY1305 ECDHE-ECDSA-AES256-GCM-SHA384 ECDHE-RSA-AES256-GCM-SHA384 ECDHE-ECDSA-AES128-GCM-SHA256 ECDHE-RSA-AES128-GCM-SHA256
                    curves  X25519 p384 p521
                    must_staple
            }
    }

### Example configuration: Docker Flow Proxy + Docker Swarm

```
version: '3.7'
networks:
  default:
    driver: overlay
    attachable: true
    external: false
  proxy:
    external: true
services:
  swarm-listener:
    image: dockerflow/docker-flow-swarm-listener:latest
    hostname: swarm-listener
    init: true
    networks:
      - default
      - proxy
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - DF_NOTIFY_CREATE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/reconfigure
      - DF_NOTIFY_REMOVE_SERVICE_URL=http://proxy:8080/v1/docker-flow-proxy/remove
    deploy:
      placement:
        constraints:
          - node.role==manager
      restart_policy:
        condition: any
        delay: 10s
        max_attempts: 99
        window: 180s
    healthcheck:
      test: [ "CMD", "wget", "http://localhost:8080/v1/docker-flow-swarm-listener/ping", "-O", "/dev/null" ]
      interval: 2m
      timeout: 1m
      retries: 3
  proxy:
    image: dockerflow/docker-flow-proxy:latest
    hostname: proxy
    init: true
    networks:
      - default
      - proxy
    ports:
      - 80:80
      - 443:443
    volumes:
      - ./data/proxy/certs:/certs
    environment:
      TINI_SUBREAPER: 1
      LISTENER_ADDRESS: swarm-listener
      MODE: swarm
      COMPRESSION_ALGO: gzip
      COMPRESSION_TYPE: text/css text/html text/javascript application/javascript text/plain text/xml application/json
      CONNECTION_MODE: http-keep-alive
      DEBUG: "true"
      HTTPS_ONLY: "true"
      STATS_URI: /stats
      EXTRA_FRONTEND: http-request set-log-level debug,http-response set-log-level debug,capture request header User-Agent len 64,acl is_vd path -i /dns-admin,http-request redirect scheme https drop-query append-slash if is_vd,http-response set-header X-Frame-Options DENY,http-response set-header X-Content-Type-Options nosniff,
      EXTRA_GLOBAL:
      SSL_BIND_OPTIONS: no-sslv3 no-tls-tickets no-tlsv10 no-tlsv11
      SSL_BIND_CIPHERS: EECDH+AESGCM:EDH+AESGCM
    deploy:
      replicas: 1
      restart_policy:
        condition: any
        delay: 10s
        max_attempts: 99
        window: 180s
    healthcheck:
      test: [ "CMD", "sh", "-c", "/usr/local/bin/check.sh"]
      interval: 2m
      timeout: 1m
      retries: 3

  doh-server:
    image: satishweb/doh-server
    # Docker Image based on https://github.com/m13253/dns-over-https
    hostname: doh-server
    networks:
      - default
    environment:
      DEBUG: "0"
      UPSTREAM_DNS_SERVER: "udp:YOUR-DNS-SERVER-IP:53"
      DOH_HTTP_PREFIX: "/dns-query"
      DOH_SERVER_LISTEN: ":8053"
      DOH_SERVER_TIMEOUT: "10"
      DOH_SERVER_TRIES: "3"
      DOH_SERVER_VERBOSE: "true"
      # You can add more variables here or as docker secret and entrypoint
      # script will replace them inside doh-server.conf file
      # Entrypoint script source is at https://github.com/satishweb/docker-doh
    volumes:
      # If you want to use your custom doh-server.conf, use below volume mount.
      # - ./doh-server.conf:/server/doh-server.conf
      # Mount app-config script with your customizations
      # - ./app-config:/app-config
    deploy:
      replicas: 1
      restart_policy:
        condition: any
        delay: 10s
        max_attempts: 99
        window: 180s
      labels:
        - com.df.notify=true
        - com.df.distribute=true
        - com.df.servicePath='/dns-query'
        - com.df.port=8053
````
> Above example needs you to add your chained SSL certificate in folder: ./data/proxy/certs and configure upstream DNS server address.

> Complete Docker Stack with DFProxy + Lets Encrypt SSL: https://github.com/satishweb/docker-doh

> Docker Flow Proxy: https://github.com/docker-flow/docker-flow-proxy

> No IPV6 Support: Docker Swarm does not support IPV6 as of yet. Issue is logged [here](https://github.com/moby/moby/issues/24379)

## DNSSEC

DNS-over-HTTPS is compatible with DNSSEC, and requests DNSSEC signatures by
default. However signature validation is not built-in. It is highly recommended
that you install `unbound` or `bind` and pass results for them to validate DNS
records.

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
