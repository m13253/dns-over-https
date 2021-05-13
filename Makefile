.PHONY: all clean install uninstall

PREFIX = /usr/local

ifeq ($(GOROOT),)
GOBUILD = go build
else
GOBUILD = $(GOROOT)/bin/go build
endif

ifeq ($(shell uname),Darwin)
CONFDIR = /usr/local/etc/dns-over-https
else
CONFDIR = /etc/dns-over-https
endif

all: doh-client/doh-client doh-server/doh-server
	if [ "`uname`" = "Darwin" ]; then \
		$(MAKE) -C darwin-wrapper; \
	fi

clean:
	rm -f doh-client/doh-client doh-server/doh-server
	if [ "`uname`" = "Darwin" ]; then \
		$(MAKE) -C darwin-wrapper clean; \
	fi

install:
	[ -e doh-client/doh-client ] || $(MAKE) doh-client/doh-client
	[ -e doh-server/doh-server ] || $(MAKE) doh-server/doh-server
	mkdir -p "$(DESTDIR)$(PREFIX)/bin/"
	install -m0755 doh-client/doh-client "$(DESTDIR)$(PREFIX)/bin/doh-client"
	install -m0755 doh-server/doh-server "$(DESTDIR)$(PREFIX)/bin/doh-server"
	mkdir -p "$(DESTDIR)$(CONFDIR)/"
	install -m0644 doh-client/doh-client.conf "$(DESTDIR)$(CONFDIR)/doh-client.conf.example"
	install -m0644 doh-server/doh-server.conf "$(DESTDIR)$(CONFDIR)/doh-server.conf.example"
	[ -e "$(DESTDIR)$(CONFDIR)/doh-client.conf" ] || install -m0644 doh-client/doh-client.conf "$(DESTDIR)$(CONFDIR)/doh-client.conf"
	[ -e "$(DESTDIR)$(CONFDIR)/doh-server.conf" ] || install -m0644 doh-server/doh-server.conf "$(DESTDIR)$(CONFDIR)/doh-server.conf"
	if [ "`uname`" = "Linux" ]; then \
		$(MAKE) -C systemd install "DESTDIR=$(DESTDIR)"; \
		$(MAKE) -C NetworkManager install "DESTDIR=$(DESTDIR)"; \
	elif [ "`uname`" = "Darwin" ]; then \
		$(MAKE) -C darwin-wrapper install "DESTDIR=$(DESTDIR)" "PREFIX=$(PREFIX)"; \
		$(MAKE) -C launchd install "DESTDIR=$(DESTDIR)"; \
	fi

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/bin/doh-client" "$(DESTDIR)$(PREFIX)/bin/doh-server" "$(DESTDIR)$(CONFDIR)/doh-client.conf.example" "$(DESTDIR)$(CONFDIR)/doh-server.conf.example"
	if [ "`uname`" = "Linux" ]; then \
		$(MAKE) -C systemd uninstall "DESTDIR=$(DESTDIR)"; \
		$(MAKE) -C NetworkManager uninstall "DESTDIR=$(DESTDIR)"; \
	elif [ "`uname`" = "Darwin" ]; then \
		$(MAKE) -C launchd uninstall "DESTDIR=$(DESTDIR)"; \
	fi

doh-client/doh-client: doh-client/client.go doh-client/config/config.go doh-client/google.go doh-client/ietf.go doh-client/main.go doh-client/version.go json-dns/error.go json-dns/globalip.go json-dns/marshal.go json-dns/response.go json-dns/unmarshal.go
	cd doh-client && $(GOBUILD)

doh-server/doh-server: doh-server/config.go doh-server/google.go doh-server/ietf.go doh-server/main.go doh-server/server.go doh-server/version.go json-dns/error.go json-dns/globalip.go json-dns/marshal.go json-dns/response.go json-dns/unmarshal.go
	cd doh-server && $(GOBUILD)
