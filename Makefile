.PHONY: dev build templ tailwind clean install

# Install dependencies
install:
	go mod download
	go install github.com/a-h/templ/cmd/templ@latest
	go install github.com/air-verse/air@latest
	npm install -D tailwindcss

# Generate templ files
templ:
	templ generate

# Watch templ files (optional if using air, but good for standalone)
templ-watch:
	templ generate --watch

# Build Tailwind CSS (one-time)
tailwind:
	npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --minify

# Watch Tailwind CSS (development)
tailwind-watch:
	npx tailwindcss -i ./static/css/input.css -o ./static/css/output.css --watch

# Build everything for production
build: templ tailwind
	go build -o bin/server ./cmd/server

# Run Air (Go + Templ Hot Reload)
air:
	air -c .air.toml

# Development mode (Full Hot Reload)
# 1. Build CSS first (ensures output.css exists)
# 2. Run air and tailwind-watch in parallel
dev: tailwind templ
	make -j 2 air tailwind-watch

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f static/css/output.css
