.PHONY: all build dev clean deps run

# Build everything
all: deps build

# Install dependencies
deps:
	cd web && npm install

# Build frontend and embed in server
build: deps
	cd web && npm run build
	cd server && go mod tidy && go build -o ../neve .

# Run development server (frontend only)
dev:
	cd web && npm run dev

# Run the production server
run: build
	./neve

# Clean build artifacts
clean:
	rm -rf server/static
	rm -f neve

# Build and run for development
dev-server:
	cd server && go run .
