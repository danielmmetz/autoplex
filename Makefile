.PHONY: bootstrap
boostrap: install
	systemctl stop autoplex
	systemctl daemon-reload
	systemctl enable autoplex.timer
	systemctl start autoplex

.PHONY: install
install:
	GO111MODULE=on go build -o /usr/local/bin/autoplex
	cp autoplex.service /etc/systemd/system/autoplex.service
	cp autoplex.timer /etc/systemd/system/autoplex.timer

.PHONY: clean
clean:
	systemctl stop autoplex.timer
	systemctl disable autoplex.timer
	systemctl stop autoplex
	systemctl daemon-reload
	systemctl reset-failed
	rm /etc/systemd/system/autoplex.timer
	rm /etc/systemd/system/autoplex.service
	rm /usr/local/bin/autoplex

.PHONY: run
run:
	GO111MODULE=on go run main.go
