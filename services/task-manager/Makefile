test-unit:
    go test ./... -short -v

test-integration:
    go test ./... -tags=integration -v -parallel=1

test-all:
    $(MAKE) test-unit && $(MAKE) test-integration
