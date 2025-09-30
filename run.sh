#!/bin/bash

set -e

# switch case on the first argument
case "$1" in
    --help|-h|help)
        echo "Usage: $0 {full|complete|down|test|test-unit|test-integration}"
        echo ""
        echo "Options:"
        echo "  full              - Start services + interactive radclient shell"
        echo "  complete          - Start all services including radclient-test"
        echo "  down              - Stop all services"
        echo "  test              - Run all tests (unit + integration)"
        echo "  test-unit         - Run only unit tests"
        echo "  test-integration  - Run integration tests (requires Redis)"
        exit 0
        ;;
esac

# prerequisite: .env has to be in the current directory
if [ ! -f .env ]; then
    echo "Error: .env file not found, please create it in the current directory and use .env.example as a template"
    exit 1
fi

# sources all the .env vars in the current environment so that docker run can use them
set -a; source .env; set +a

# switch case on the first argument
case "$1" in
    full)
        # start services + interactive radclient shell
        docker compose up -d redis controlplane logger
        sleep 2
        docker rm -f radclient-test 2>/dev/null || true
        docker run -it --rm \
            --name radclient-test \
            --network radius-network \
            -e RADIUS_SHARED_SECRET="$RADIUS_SHARED_SECRET" \
            -e RADIUS_SERVER=radius-controlplane \
            -v ./examples:/requests \
            --entrypoint /bin/bash \
            radclient-test
        ;;
    complete)
        # start all services including radclient-test
        docker rm -f radclient-test 2>/dev/null || true
        docker compose up
        ;;
    down)
        # stop all services
        docker compose down
        docker rm -f radclient-test 2>/dev/null || true
        ;;
    test)
        echo "Running all tests with coverage..."
        echo ""
        echo "Starting Redis for integration tests..."
        docker compose up -d redis
        sleep 2
        echo ""
        REDIS_HOST=localhost go test -v -race -coverprofile=coverage.out ./internal/...
        echo ""
        echo "=== Overall Coverage ==="
        go tool cover -func=coverage.out | grep total
        echo ""

    ;;
    test-unit)
        echo "Running unit tests..."
        go test -v -race ./internal/config ./internal/models ./internal/notifier
        echo ""
        echo "✓ Unit tests passed"
        ;;
    test-integration)
        echo "Running integration tests..."
        echo "Starting Redis for integration tests..."
        docker compose up -d redis
        sleep 2
        echo "Running integration tests with REDIS_HOST=localhost..."
        REDIS_HOST=localhost go test -v -race ./internal/storage ./internal/logger
        echo ""
        echo "✓ Integration tests passed"
        ;;
    *)
        echo "Usage: $0 {full|complete|down|test|test-unit|test-integration}"
        echo "Run '$0 --help' for more info"
        exit 1
        ;;
esac