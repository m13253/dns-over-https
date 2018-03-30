.PHONY: install uninstall

PREFIX = /usr/local
LAUNCHD_DIR = /Library/LaunchDaemons

install:
	mkdir -p "$(DESTDIR)$(LAUNCHD_DIR)"
	install -m0644 doh-client.plist "$(DESTDIR)$(LAUNCHD_DIR)/doh-client.plist"
	install -m0644 doh-server.plist "$(DESTDIR)$(LAUNCHD_DIR)/doh-server.plist"
	@echo
	@echo 'Note:'
	@echo '    Use "sudo launchctl load $(DESTDIR)$(LAUNCHD_DIR)/doh-client.plist" to start doh-client,'
	@echo '    use "sudo launchctl load -w $(DESTDIR)$(LAUNCHD_DIR)/doh-server.plist" to enable doh-server.'

uninstall:
	rm -f "$(DESTDIR)$(LAUNCHD_DIR)/doh-client.plist" "$(DESTDIR)$(LAUNCHD_DIR)/doh-server.plist"
