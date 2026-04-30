BINARY      := sqlitui
INSTALL_DIR := $(HOME)/.local/bin

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
  -X main.version=$(VERSION) \
  -X main.commit=$(COMMIT) \
  -X main.date=$(DATE)

.PHONY: build dev install clean snapshot release-check release tag-% help

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

# Build and run the local binary so you don't pick up the globally installed one.
# Pass a DB path via DB=path/to.sqlite (or as the first non-flag argument: `make dev DB=foo.sqlite`).
dev: build
	./$(BINARY) $(DB)

install: build
	mkdir -p $(INSTALL_DIR)
	install -m 0755 $(BINARY) $(INSTALL_DIR)/$(BINARY)
	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

clean:
	rm -rf $(BINARY) dist/

snapshot:
	goreleaser release --snapshot --clean

release-check:
	@if [ -n "$$(git status --porcelain)" ]; then \
	  echo "Working tree is dirty. Commit or stash changes first." >&2; \
	  exit 1; \
	fi
	@if [ "$$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then \
	  echo "Releases must be cut from main (currently on $$(git rev-parse --abbrev-ref HEAD))." >&2; \
	  exit 1; \
	fi
	@git fetch --tags

# Cut a release. By default, bumps the patch component of the latest vX.Y.Z tag.
# Override with an explicit tag: `make release VERSION_TAG=v0.2.0`.
# Pushing the tag triggers the GitHub Actions release workflow (goreleaser).
release: release-check
	@set -e; \
	if [ -n "$(VERSION_TAG)" ]; then \
	  TAG="$(VERSION_TAG)"; \
	else \
	  LATEST=$$(git tag --list 'v[0-9]*.[0-9]*.[0-9]*' --sort=-v:refname | head -n1); \
	  if [ -z "$$LATEST" ]; then \
	    echo "No vX.Y.Z tag found; pass VERSION_TAG=vX.Y.Z explicitly." >&2; exit 1; \
	  fi; \
	  V=$${LATEST#v}; \
	  MAJOR=$${V%%.*}; REST=$${V#*.}; MINOR=$${REST%%.*}; PATCH=$${REST#*.}; \
	  TAG="v$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	  echo "Latest tag: $$LATEST -> bumping patch to $$TAG"; \
	fi; \
	case "$$TAG" in v[0-9]*.[0-9]*.[0-9]*) ;; \
	  *) echo "Tag must look like vX.Y.Z (got: $$TAG)" >&2; exit 1 ;; \
	esac; \
	if git rev-parse "$$TAG" >/dev/null 2>&1; then \
	  echo "Tag $$TAG already exists." >&2; exit 1; \
	fi; \
	git tag -a "$$TAG" -m "Release $$TAG"; \
	git push origin "$$TAG"; \
	echo "Tag $$TAG pushed. Watch the release at:"; \
	echo "  https://github.com/markovic-nikola/sqlitui/actions"

help:
	@echo "Targets:"
	@echo "  build              build $(BINARY) for the host platform"
	@echo "  dev [DB=path]      build and run the local binary (does not touch global install)"
	@echo "  install            build and copy to $(INSTALL_DIR)"
	@echo "  clean              remove build artifacts"
	@echo "  snapshot           run goreleaser in snapshot mode (local cross-build into ./dist)"
	@echo "  release            bump patch from latest vX.Y.Z tag, push, trigger release workflow"
	@echo "  release VERSION_TAG=vX.Y.Z"
	@echo "                     same, but with an explicit tag (use for minor/major bumps)"
