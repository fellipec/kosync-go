BINARY  := kosync-go
MODULE  := $(shell go list -m 2>/dev/null || echo github.com/fellipec/kosync-go)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X main.Version=$(VERSION)"

INSTALL_DIR := /usr/local/bin
SERVICE_DIR := /etc/systemd/system
DATA_DIR    := /var/lib/kosync
SERVICE_USER := kosync

.PHONY: all build clean test vet fmt install uninstall run help

all: build

## build: compila o binário para a arquitetura local
build:
	go build $(LDFLAGS) -o $(BINARY) .

## build-linux-amd64: cross-compile para Linux amd64 (para deploy no servidor)
build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY)-linux-amd64 .

## clean: remove binários gerados
clean:
	rm -f $(BINARY) $(BINARY)-linux-amd64

## test: roda os testes
test:
	go test ./...

## vet: roda go vet
vet:
	go vet ./...

## fmt: formata o código com gofmt
fmt:
	gofmt -w .

## check: vet + test
check: vet test

## run: compila e executa localmente (HTTP, porta 17200)
run: build
	./$(BINARY) --insecure --port 17200

## install: instala o binário e configura o serviço systemd
install: build
	install -Dm755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	id -u $(SERVICE_USER) &>/dev/null || useradd --system --no-create-home --shell /usr/sbin/nologin $(SERVICE_USER)
	mkdir -p $(DATA_DIR)
	chown $(SERVICE_USER):$(SERVICE_USER) $(DATA_DIR)
	install -Dm644 kosync.service $(SERVICE_DIR)/kosync.service
	systemctl daemon-reload
	systemctl enable --now kosync
	@echo "kosync-go instalado e serviço iniciado."

## uninstall: para o serviço e remove os arquivos instalados
uninstall:
	systemctl disable --now kosync || true
	rm -f $(INSTALL_DIR)/$(BINARY) $(SERVICE_DIR)/kosync.service
	systemctl daemon-reload
	@echo "kosync-go removido. Os dados em $(DATA_DIR) foram mantidos."

## version: mostra a versão que será embutida no binário
version:
	@echo $(VERSION)

## help: lista os alvos disponíveis
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'