# Rebuild and install the global patchdock binary (into $(go env GOPATH)/bin).
.PHONY: install
install:
	go install .
	@echo "installed: $$(command -v patchdock)"
