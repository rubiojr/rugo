.PHONY: build test rats

build:
	@go build -o bin/rugo .

test:
	@go test ./... -count=1

rats: build
	@PATH="$(CURDIR)/bin:$(PATH)" bin/rugo rats --recap --timing rats/
