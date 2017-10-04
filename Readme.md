DNS-over-HTTPS
==============

Client and server software to query DNS over HTTPS protocol

## Easy start

Install [Go](https://golang.org), at least version 1.8.

First create an empty directory, used for `$GOPATH`:

    mkdir ~/gopath
    export GOPATH=~/gopath

To build the program, type:

    make

To install DNS-over-HTTPS as Systemd services, type:

    sudo make install

By default, [Google DNS over HTTPS](https://dns.google.com) is used. It should work
for most users (except for People's Republic of China). If you need to modify the
default settings, type:

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

Although DNS-over-HTTPS can work alone, a HTTP service muxer would be useful as you
can host DNS-over-HTTPS along with other HTTPS services.

## Protocol compatibility

- [X] IPv4 / IPv6
- [X] EDNS0 large UDP packet
- [X] EDNS0 Client Subnet
- [ ] DNSSEC

DNSSEC is planned but not implemented yet. Contributions are welcome.

## License

DNS-over-HTTPS is licensed under [GNU AFFERO GENERAL PUBLIC LICENSE](LICENSE)
version 3 or later. That means, if you improved DNS-over-HTTPS or fixed a bug, you
**must** disclose your modification to the public, so that others may benefit from
your modification, in the same way you receive benefits from this project.
