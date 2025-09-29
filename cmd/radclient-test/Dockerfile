FROM debian:stable-slim

RUN apt-get update && apt-get install -y freeradius-utils bash && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy the script
COPY ./cmd/radclient-test/main.sh /app/main.sh
RUN chmod +x /app/main.sh

# entrypoint
ENTRYPOINT ["/app/main.sh"]
