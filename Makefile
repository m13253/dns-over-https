.PHONY: all clean install uninstall

GOBUILD=go build
GOGET=go get -d -v .
PREFIX=/usr/local

all: doh-client/doh-client doh-server/doh-server

clean:
	rm -f doh-client/doh-client doh-server/doh-server

install: doh-client/doh-client doh-server/doh-server
	install -Dm0755 doh-client/doh-client "$(DESTDIR)$(PREFIX)/bin/doh-client"
	install -Dm0755 doh-server/doh-server "$(DESTDIR)$(PREFIX)/bin/doh-server"
	[ -e "$(DESTDIR)/etc/dns-over-https/doh-client.conf" ] || install -Dm0644 doh-client/doh-client.conf "$(DESTDIR)/etc/dns-over-https/doh-client.conf"
	[ -e "$(DESTDIR)/etc/dns-over-https/doh-server.conf" ] || install -Dm0644 doh-server/doh-server.conf "$(DESTDIR)/etc/dns-over-https/doh-server.conf"
	$(MAKE) -C systemd install "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"
	$(MAKE) -C NetworkManager install "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/doh-client" "$(DESTDIR)$(PREFIX)/bin/doh-server"
	$(MAKE) -C systemd uninstall "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"
	$(MAKE) -C NetworkManager uninstall "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"

doh-client/doh-client: doh-client/client.go doh-client/config.go doh-client/main.go json-dns/error.go json-dns/globalip.go json-dns/marshal.go json-dns/response.go json-dns/unmarshal.go
	cd doh-client && $(GOGET) && $(GOBUILD)

doh-server/doh-server: doh-server/config.go doh-server/main.go doh-server/server.go json-dns/error.go json-dns/globalip.go json-dns/marshal.go json-dns/response.go json-dns/unmarshal.go
	cd doh-server && $(GOGET) && $(GOBUILD)
