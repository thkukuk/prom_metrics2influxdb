PM2I_BIN := bin/prom_metrics2influxdb

GO ?= go
GO_MD2MAN ?= go-md2man

VERSION := $(shell cat VERSION)
USE_VENDOR =
LOCAL_LDFLAGS = -buildmode=pie -ldflags "-X=github.com/thkukuk/prom_metrics2influxdb/cmd/prom_metrics2influxdb.Version=$(VERSION)"

.PHONY: all api build vendor
all: dep build

dep: ## Get the dependencies
	@$(GO) get -v -d ./...

update: ## Get and update the dependencies
	@$(GO) get -v -d -u ./...

tidy: ## Clean up dependencies
	@$(GO) mod tidy

vendor: dep ## Create vendor directory
	@$(GO) mod vendor

build: ## Build the binary files
	$(GO) build -v -o $(PM2I_BIN) $(USE_VENDOR) $(LOCAL_LDFLAGS) ./cmd/prom_metrics2influxdb

clean: ## Remove previous builds
	@rm -f $(PM2I_BIN)

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'


.PHONY: release
release: ## create release package from git
	git clone https://github.com/thkukuk/prom_metrics2influxdb
	mv prom_metrics2influxdb prom_metrics2influxdb-$(VERSION)
	sed -i -e 's|USE_VENDOR =|USE_VENDOR = -mod vendor|g' prom_metrics2influxdb-$(VERSION)/Makefile
	make -C prom_metrics2influxdb-$(VERSION) vendor
	cp VERSION prom_metrics2influxdb-$(VERSION)
	tar --exclude .git -cJf prom_metrics2influxdb-$(VERSION).tar.xz prom_metrics2influxdb-$(VERSION)
	rm -rf prom_metrics2influxdb-$(VERSION)
