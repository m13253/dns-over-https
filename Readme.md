DNS-over-HTTPS
==============

Client and server software to query DNS over HTTPS, using [Google DNS-over-HTTPS protocol](https://developers.google.com/speed/public-dns/docs/dns-over-https)
and [IETF DNS-over-HTTPS (RFC 8484)](https://www.rfc-editor.org/rfc/rfc8484.txt).

## Guide

[Tutorial to setup your own DNS-over-HTTPS (DoH) server](https://www.aaflalo.me/2018/10/tutorial-setup-dns-over-https-server/). (Thanks to Antoine Aflalo)

## Installing

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

If it is OK, you will wee:

    ;; SERVER: 127.0.0.1#53(127.0.0.1)

### Uninstalling

To uninstall, type:

    sudo make uninstall

The configuration files are kept at `/etc/dns-over-https`. Remove them manually if you want.

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
