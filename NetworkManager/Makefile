.PHONY: install uninstall

install:
	install -Dm0755 dispatcher.d/doh-client "$(DESTDIR)/etc/NetworkManager/dispatcher.d/doh-client"
	install -Dm0755 dispatcher.d/doh-server "$(DESTDIR)/etc/NetworkManager/dispatcher.d/doh-server"

uninstall:
	rm -f "$(DESTDIR)/etc/NetworkManager/dispatcher.d/doh-client" "$(DESTDIR)/etc/NetworkManager/dispatcher.d/doh-server"
	rmdir "$(DESTDIR)/etc/NetworkManager/dispatcher.d" || true
	rmdir "$(DESTDIR)/etc/NetworkManager" || true
