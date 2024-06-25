TOOLS_DIR := tools
BABYLON_PKG := github.com/babylonchain/babylon/cmd/babylond

install-babylond:
	cd $(TOOLS_DIR); \
	go install -trimpath $(BABYLON_PKG)

PACKAGES_E2E_OP=$(shell go list -tags=e2e_op ./... | grep '/itest')

# Clean up environments by stopping processes and removing data directories
clean-e2e:
	@pids=$$(ps aux | grep -E 'babylond start' | grep -v grep | awk '{print $$2}' | tr '\n' ' '); \
	if [ -n "$$pids" ]; then \
		echo $$pids | xargs kill; \
		echo "Killed processes $$pids"; \
	else \
		echo "No processes to kill"; \
	fi

.PHONY: test run clean-e2e test-e2e-op

# Target to run tests
test:
	go test ./sdk -v

# Target to run the demo
run:
	go run demo/main.go

test-e2e-op: clean-e2e install-babylond
	go test -mod=readonly -timeout=25m -v $(PACKAGES_E2E_OP) -count=1 --tags=e2e_op
