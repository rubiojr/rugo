PACKAGES = unicode strconv path/filepath encoding/base64 encoding/hex \
           math os math/rand/v2 time html path crypto/sha256 crypto/md5 strings

.PHONY: bridgegen build test rats

bridgegen: build
	@for pkg in $(PACKAGES); do \
		bin/rugo dev bridgegen "$$pkg"; \
	done
	@echo ""
	@echo "Bridge regenerated. Run 'make rats' to verify."

build:
	@go build -o bin/rugo .

test:
	@go test ./... -count=1

rats: build
	@bin/rugo rats rats/
