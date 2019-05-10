.PHONY: all clean install uninstall

SWIFTC = swiftc
PREFIX = /usr/local

SWIFTC_VERS = $(shell swiftc -version | grep -i swift | awk 'match($$0, /[0-9]/) {print substr($$0, RSTART, RLENGTH) }')

all: doh-logger
    $(info SWIFTC_VERS="$(SWIFTC_VERS)")

doh-logger: doh-logger.swift
    ifeq ($(SWIFTC_VERS), 5)
	    $(SWIFTC) -o $@ -O $<
    else
	    $(SWIFTC) -o $@ -O -static-stdlib $<
    endif

clean:
	rm -f doh-logger

install: doh-logger
	mkdir -p $(DESTDIR)$(PREFIX)/bin
	install -m0755 doh-logger $(DESTDIR)$(PREFIX)/bin

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/doh-logger
