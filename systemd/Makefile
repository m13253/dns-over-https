.PHONY: install uninstall

SYSTEMD_DIR = /usr/lib/systemd
SYSTEMD_UNIT_DIR = $(SYSTEMD_DIR)/system

install:
	install -Dm0644 doh-client.service "$(DESTDIR)$(PREFIX)$(SYSTEMD_UNIT_DIR)/doh-client.service"
	install -Dm0644 doh-server.service "$(DESTDIR)$(PREFIX)$(SYSTEMD_UNIT_DIR)/doh-server.service"
	systemctl daemon-reload || true

uninstall:
	rm -f "$(DESTDIR)$(PREFIX)$(SYSTEMD_UNIT_DIR)/doh-client.service" "$(DESTDIR)$(PREFIX)$(SYSTEMD_UNIT_DIR)/doh-server.service"
	systemctl daemon-reload || true
