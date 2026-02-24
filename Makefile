help:
	@echo "MOCHI — Multi-task AI coding orchestrator"
	@echo ""
	@echo "Usage:"
	@echo "  make build      Build the mochi binary"
	@echo "  make run        Run mochi using 'main.go'"
	@echo "  make test       Run all tests"
	@echo "  make clean      Remove build artifacts and worktrees"

build:
	go build -o build/mochi main.go

run:
	go run main.go

test:
	go test ./...

clean:
	rm -rf mochi .worktrees logs .mochi_manifest.json