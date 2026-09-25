#!/usr/bin/env bash
# Rebuilds and redeploys Cloud Run services from the local source.
# Run bootstrap.sh once first; this only updates images, never infrastructure.
#
# Usage: ./deploy/scripts/deploy.sh [api|web|all]   (default: all)
# Env:   PROJECT_ID (default lemonade-app-dev), REGION (default us-central1)
set -euo pipefail
source "$(dirname "$0")/lib.sh"

case "${1:-all}" in
  api | web) SERVICES=("$1") ;;
  all) SERVICES=(api web) ;;
  *) echo "usage: $0 [api|web|all]" >&2; exit 2 ;;
esac

require gcloud
TAG="$(new_tag)"

echo "==> Building ${SERVICES[*]} ($TAG)"
build_images "$TAG" "${SERVICES[@]}"

echo "==> Deploying"
for svc in "${SERVICES[@]}"; do
  deploy_image "$svc" "$TAG"
  echo "    $svc: $(g run services describe "$svc" --region="$REGION" --format='value(status.url)')"
done
