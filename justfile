# Run both Go backend and SvelteKit frontend for development

# Run both servers concurrently
dev:
    concurrently --names "go,svelte" --prefix-colors "blue,magenta" "go run ./cmd/main.go" "cd webui && npm run dev"
