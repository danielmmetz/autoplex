.PHONY: bootstrap
boostrap: install
	sudo systemctl stop autoplex.timer
	sudo systemctl daemon-reload
	sudo systemctl daemon-reexec
	sudo systemctl enable autoplex.timer
	sudo systemctl start autoplex.timer

.PHONY: install
install:
	sudo GO111MODULE=on go build -o /usr/local/bin/autoplex
	sudo cp autoplex.service /etc/systemd/system/autoplex.service
	sudo cp autoplex.timer /etc/systemd/system/autoplex.timer

.PHONY: clean
clean:
	sudo systemctl stop autoplex.timer
	sudo systemctl disable autoplex.timer
	sudo systemctl stop autoplex
	sudo systemctl daemon-reload
	sudo systemctl reset-failed
	sudo rm /etc/systemd/system/autoplex.timer
	sudo rm /etc/systemd/system/autoplex.service
	sudo rm /usr/local/bin/autoplex

.PHONY: run
run:
	GO111MODULE=on go run main.go
