build:
	@if [ ! -f .env ]; then echo "Error: .env file not found"; exit 1; fi
	@export $$(grep -v '^#' .env | grep -v '^$$' | xargs) && \
	if [ -z "$$CLIENT_ID" ]; then echo "Error: CLIENT_ID environment variable is required"; exit 1; fi && \
	if [ -z "$$PROJECT_ID" ]; then echo "Error: PROJECT_ID environment variable is required"; exit 1; fi && \
	go build -ldflags "\
		-X github.com/quicdb/quic-cli/internal/config.ClientID=$$CLIENT_ID \
		-X github.com/quicdb/quic-cli/internal/config.ProjectID=$$PROJECT_ID \
		-X github.com/quicdb/quic-cli/internal/config.APIURL=https://api.quicdb.com/api/cli \
		-X github.com/quicdb/quic-cli/internal/config.AuthorizeURL=https://dash.quicdb.com/oauth/authorize \
		-X github.com/quicdb/quic-cli/internal/config.StytchURL=https://api.stytch.com" \
		-o bin/quic

release:
	@if [ -z "$(VERSION)" ]; then echo "Usage: make release VERSION=v1.0.0"; exit 1; fi
	git tag $(VERSION)
	git push origin $(VERSION)
	@echo "> GitHub Action: https://github.com/quicdb/quic-cli/actions"


.PHONY: dev prod release
