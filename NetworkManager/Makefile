.PHONY: install uninstall

install:
	install -Dm0755 dispatcher.d/doh-client "$(DESTDIR)$(PREFIX)/etc/NetworkManager/dispatcher.d/doh-client"
	install -Dm0755 dispatcher.d/doh-server "$(DESTDIR)$(PREFIX)/etc/NetworkManager/dispatcher.d/doh-server"

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)/etc/NetworkManager/dispatcher.d/doh-client" "$(DESTDIR)$(PREFIX)/etc/NetworkManager/dispatcher.d/doh-server"
	rmdir "$(DESTDIR)$(PREFIX)/etc/NetworkManager/dispatcher.d" || true
	rmdir "$(DESTDIR)$(PREFIX)/etc/NetworkManager" || true
