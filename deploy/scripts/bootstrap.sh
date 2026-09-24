#!/usr/bin/env bash
# Creates the whole stack on Google Cloud: Terraform state bucket, Artifact
# Registry, images (built with Cloud Build), Cloud SQL, both Cloud Run
# services, custom domains, a billing budget and GitHub Actions access.
# Safe to re-run.
# For routine redeploys, use deploy.sh instead.
#
# Usage: ./deploy/scripts/bootstrap.sh
# Env:   PROJECT_ID (default numeric-lemonade-app), REGION (default us-central1)
set -euo pipefail
source "$(dirname "$0")/lib.sh"

require gcloud terraform jq
TAG="$(new_tag)"

echo "==> Project $PROJECT_ID, region $REGION, image tag $TAG"
g services enable serviceusage.googleapis.com cloudresourcemanager.googleapis.com storage.googleapis.com

BILLING_ACCOUNT="$(billing_account)"
[ -n "$BILLING_ACCOUNT" ] || { echo "error: no billing account linked to $PROJECT_ID" >&2; exit 1; }
tf_vars

echo "==> Terraform state bucket gs://$STATE_BUCKET"
if ! g storage buckets describe "gs://$STATE_BUCKET" >/dev/null 2>&1; then
  g storage buckets create "gs://$STATE_BUCKET" \
    --location="$REGION" --uniform-bucket-level-access --public-access-prevention
  g storage buckets update "gs://$STATE_BUCKET" --versioning
fi
tf_init

# Phase 1: everything the image build needs. Cloud Run can't be created until
# the images exist, and the images need the registry.
echo "==> Phase 1: APIs, registry, Cloud Build permissions"
terraform -chdir="$INFRA" apply -input=false -auto-approve -compact-warnings "${TF_VARS[@]}" \
  -target=google_project_service.services \
  -target=google_artifact_registry_repository.app \
  -target=google_project_iam_member.cloudbuild

echo "==> Phase 2: building images"
build_images "$TAG" api web

# Phase 3: everything else. image_tag only matters when the services are first
# created; afterwards Terraform ignores the image and deploy_image sets it.
echo "==> Phase 3: Cloud SQL, Cloud Run, domains, budget (first run takes ~10 minutes)"
terraform -chdir="$INFRA" apply -input=false -auto-approve "${TF_VARS[@]}" -var "image_tag=$TAG"

# deploy.yml reads these to authenticate. Not secret: they only work for jobs
# on this repo's main branch.
GH_REPO="$(github_repo)"
if [ -n "$GH_REPO" ] && command -v gh >/dev/null; then
  echo "==> GitHub Actions variables on $GH_REPO"
  gh variable set GCP_WORKLOAD_IDENTITY_PROVIDER --repo "$GH_REPO" \
    --body "$(terraform -chdir="$INFRA" output -raw github_workload_identity_provider)"
  gh variable set GCP_SERVICE_ACCOUNT --repo "$GH_REPO" \
    --body "$(terraform -chdir="$INFRA" output -raw github_service_account)"
elif [ -n "$GH_REPO" ]; then
  echo "note: gh not installed; set GCP_WORKLOAD_IDENTITY_PROVIDER and GCP_SERVICE_ACCOUNT on $GH_REPO from 'terraform output'"
fi

echo "==> Deploying images"
deploy_image api "$TAG"
deploy_image web "$TAG"

echo
echo "Done."
echo "  web: $(terraform -chdir="$INFRA" output -raw web_url)"
echo "  api: $(terraform -chdir="$INFRA" output -raw api_url)"
echo
echo "Custom domains: create these DNS records (DNS only, not proxied):"
terraform -chdir="$INFRA" output -json dns_records | jq -r 'to_entries[] | "  \(.key): \(.value | join(", "))"'
