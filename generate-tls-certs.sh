#!/bin/bash

# DEPRECATED: Internal TLS between containers has been removed.
# gRPC and REST traffic between containers runs plaintext on the Docker network.
# TLS termination should be handled at the edge (reverse proxy: Traefik, Caddy, nginx).
#
# This script is kept for reference. If you need edge TLS certs, use:
#   - Let's Encrypt / certbot for production
#   - mkcert (https://github.com/FiloSottile/mkcert) for local dev
#
# See Phase 8 (Infrastructure & Operations) for the edge TLS setup.

echo "This script is deprecated."
echo "Internal TLS has been removed — containers communicate over plaintext on the container network."
echo "For edge TLS, configure a reverse proxy (Traefik, Caddy, nginx) in front of the services."
exit 0
