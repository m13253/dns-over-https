.PHONY: all clean install uninstall

SWIFTC = swiftc
PREFIX = /usr/local

all: doh-logger

doh-logger: doh-logger.swift
	$(SWIFTC) -o $@ -O $<

clean:
	rm -f doh-logger

install: doh-logger
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m0755 doh-logger $(DESTDIR)$(PREFIX)/bin

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/doh-logger
