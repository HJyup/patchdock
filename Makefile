# Rebuild and install the global patchdock binary (into $(go env GOPATH)/bin).
.PHONY: install
install:
	go install ./...
	@echo "patchdock has been installed"

# Compile the SDK into sdk/dist, which is what the package publishes.
.PHONY: sdk
sdk:
	pnpm --dir sdk install
	pnpm --dir sdk run build
	@echo "sdk built into sdk/dist"
