# Run both Go backend and SvelteKit frontend for development

# Run both servers concurrently
dev:
        # Use reflex to auto-restart Go server on .go file changes
        concurrently --names "go,svelte" --prefix-colors "blue,magenta" \
            "reflex -r '\\.go$$' -s -- sh -c 'go run ./cmd/main.go'" \
            "cd webui && npm run dev -- --host 0.0.0.0"
