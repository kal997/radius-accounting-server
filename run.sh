#!/bin/bash

set -e

# switch case on the first argument
case "$1" in
    --help|-h|help)
        echo "Usage: $0 {full|complete|down}"
        echo ""
        echo "Options:"
        echo "  full      - Start services + interactive radclient shell"
        echo "  complete  - Start all services including radclient-test"
        echo "  down      - Stop all services"
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

    *)
        echo "Usage: $0 {full|complete|down}"
        echo "Run '$0 --help' for more info"
        exit 1
        ;;
esac