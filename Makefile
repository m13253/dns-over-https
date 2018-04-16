.PHONY: all clean install uninstall deps

GOBUILD=go build
GOGET=go get -d -u -v
PREFIX=/usr/local
ifeq ($(shell uname),Darwin)
	CONFDIR=/usr/local/etc/dns-over-https
else
	CONFDIR=/etc/dns-over-https
endif

all: doh-client/doh-client doh-server/doh-server

clean:
	rm -f doh-client/doh-client doh-server/doh-server

install:
	[ -e doh-client/doh-client ] || $(MAKE) doh-client/doh-client
	[ -e doh-server/doh-server ] || $(MAKE) doh-server/doh-server
	mkdir -p "$(DESTDIR)$(PREFIX)/bin/"
	install -m0755 doh-client/doh-client "$(DESTDIR)$(PREFIX)/bin/doh-client"
	install -m0755 doh-server/doh-server "$(DESTDIR)$(PREFIX)/bin/doh-server"
	mkdir -p "$(DESTDIR)$(CONFDIR)/"
	[ -e "$(DESTDIR)$(CONFDIR)/doh-client.conf" ] || install -m0644 doh-client/doh-client.conf "$(DESTDIR)$(CONFDIR)/doh-client.conf"
	[ -e "$(DESTDIR)$(CONFDIR)/doh-server.conf" ] || install -m0644 doh-server/doh-server.conf "$(DESTDIR)$(CONFDIR)/doh-server.conf"
	if [ "`uname`" = "Linux" ]; then \
		$(MAKE) -C systemd install "DESTDIR=$(DESTDIR)"; \
		$(MAKE) -C NetworkManager install "DESTDIR=$(DESTDIR)"; \
	elif [ "`uname`" = "Darwin" ]; then \
		$(MAKE) -C launchd install "DESTDIR=$(DESTDIR)"; \
	fi

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/doh-client" "$(DESTDIR)$(PREFIX)/bin/doh-server"
	if [ "`uname`" = "Linux" ]; then \
		$(MAKE) -C systemd uninstall "DESTDIR=$(DESTDIR)"; \
		$(MAKE) -C NetworkManager uninstall "DESTDIR=$(DESTDIR)"; \
	elif [ "`uname`" = "Darwin" ]; then \
		$(MAKE) -C launchd uninstall "DESTDIR=$(DESTDIR)"; \
	fi

deps:
	$(GOGET) ./doh-client ./doh-server

doh-client/doh-client: deps doh-client/client.go doh-client/config.go doh-client/google.go doh-client/ietf.go doh-client/main.go json-dns/error.go json-dns/globalip.go json-dns/marshal.go json-dns/response.go json-dns/unmarshal.go
	cd doh-client && $(GOBUILD)

doh-server/doh-server: deps doh-server/config.go doh-server/google.go doh-server/ietf.go doh-server/main.go doh-server/server.go json-dns/error.go json-dns/globalip.go json-dns/marshal.go json-dns/response.go json-dns/unmarshal.go
	cd doh-server && $(GOBUILD)
