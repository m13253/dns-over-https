DNS-over-HTTPS
==============

Client and server software to query DNS over HTTPS, using [Google DNS-over-HTTPS protocol](https://developers.google.com/speed/public-dns/docs/dns-over-https).

## Easy start

Install [Go](https://golang.org), at least version 1.8.

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

    sudo cp /usr/lib/systemd/system/doh-client.service /etc/systemd/system/
    sudoedit /etc/systemd/system/doh-client.service

To automatically start DNS-over-HTTPS client as a system service, type:

    sudo systemctl start doh-client.service
    sudo systemctl enable doh-client.service

Then, modify your DNS settings (usually with NetworkManager) to 127.0.0.1.

To test your configuration, type:

    dig www.google.com

If it is OK, you will wee:

    ;; SERVER: 127.0.0.1#53(127.0.0.1)

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

## DNSSEC

DNSSEC validation is not built-in. It is highly recommended that you install
`unbound` or `bind` and pass results for them to validate DNS records.

If you are running a server without anycast, you probably want to enable EDNS0
Client Subnet during your configuring `unbound` or `bind`.

## Protocol compatibility

DNS-over-HTTPS use a protocol compatible to [Google DNS-over-HTTPS](https://developers.google.com/speed/public-dns/docs/dns-over-https),
except for absolute expire time is preferred to relative TTL value. Refer to
[json-dns/response.go](json-dns/response.go) for a complete description of the
API.

Currently supported features are:

- [X] IPv4 / IPv6
- [X] EDNS0 large UDP packet (4 KiB by default)
- [X] EDNS0 Client Subnet (/24 for IPv4, /48 for IPv6 by default)

## License

DNS-over-HTTPS is licensed under the [MIT License](LICENSE). You are encouraged
to embed DNS-over-HTTPS into your other projects, as long as the license
permits.

You are also encouraged to disclose your improvements to the public, so that
others may benefit from your modification, in the same way you receive benefits
from this project.
