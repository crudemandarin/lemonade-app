#!/usr/bin/env bash
# Deletes everything bootstrap.sh created, then verifies nothing is left.
#
#   1. terraform destroy (after lifting Cloud SQL deletion protection)
#   2. gcloud sweep of the same resources by name, in case Terraform state is
#      missing or destroy failed part-way
#   3. Cloud Build source bucket(s) and, unless --keep-state, the state bucket
#   4. verification: exits non-zero if anything remains
#
# Usage: ./deploy/scripts/teardown.sh [--yes] [--keep-state]
# Env:   PROJECT_ID (default lemonade-app-dev), REGION (default us-central1)
set -uo pipefail
source "$(dirname "$0")/lib.sh"

ASSUME_YES=0
KEEP_STATE=0
for arg in "$@"; do
  case "$arg" in
    --yes) ASSUME_YES=1 ;;
    --keep-state) KEEP_STATE=1 ;;
    *) echo "unknown argument: $arg" >&2; exit 2 ;;
  esac
done

# Names must match infra/*.tf.
RUN_SERVICES=(web api)
SQL_PREFIX="lemonade-db-"
AR_REPO="app"
SECRET="db-password"
SERVICE_ACCOUNTS=(
  "api-runtime@$PROJECT_ID.iam.gserviceaccount.com"
  "web-runtime@$PROJECT_ID.iam.gserviceaccount.com"
  "github-deployer@$PROJECT_ID.iam.gserviceaccount.com"
)
WIF_PREFIX="github-"
CLOUDBUILD_ROLES=(roles/artifactregistry.writer roles/logging.logWriter roles/storage.objectViewer)
BUDGET_NAME="lemonade monthly"

require gcloud curl jq
BILLING_ACCOUNT="$(billing_account)"

# Domain mappings have no GA gcloud command, so use the Cloud Run v1 API.
DOMAIN_API="https://$REGION-run.googleapis.com/apis/domains.cloudrun.com/v1/namespaces/$PROJECT_ID/domainmappings"

domain_mappings() {
  curl -fsS -H "Authorization: Bearer $(g auth print-access-token)" "$DOMAIN_API" 2>/dev/null \
    | jq -r '.items[]?.metadata.name'
}

sql_instances() {
  g sql instances list --filter="name:$SQL_PREFIX*" --format="value(name)" 2>/dev/null
}

# The Budgets API needs a quota project with user credentials, hence --billing-project.
budgets() {
  [ -n "$BILLING_ACCOUNT" ] || return 0
  g billing budgets list --billing-account="$BILLING_ACCOUNT" --billing-project="$PROJECT_ID" \
    --filter="displayName='$BUDGET_NAME'" --format='value(name)' 2>/dev/null
}

# Only active pools; deleted ones are listed separately and expire after 30 days.
wif_pools() {
  g iam workload-identity-pools list --location=global --format='value(name)' 2>/dev/null \
    | sed 's|.*/||' | grep "^$WIF_PREFIX"
}

cloudbuild_buckets() {
  g storage buckets list --format="value(name)" 2>/dev/null | grep -E '_cloudbuild$'
}

echo "This permanently deletes, in project $PROJECT_ID ($REGION):"
echo "  - Cloud Run services: ${RUN_SERVICES[*]}, and their custom domain mappings"
echo "  - Cloud SQL instance(s) $SQL_PREFIX* and ALL their data and backups"
echo "  - Artifact Registry repo '$AR_REPO' and all images"
echo "  - Secret '$SECRET', service accounts api-runtime / web-runtime / github-deployer"
echo "  - GitHub Actions identity pool ($WIF_PREFIX*) and repo variables"
echo "  - Cloud Build source bucket(s) (*_cloudbuild)"
echo "  - Billing budget '$BUDGET_NAME'"
[ "$KEEP_STATE" -eq 1 ] || echo "  - Terraform state bucket gs://$STATE_BUCKET"
if [ "$ASSUME_YES" -ne 1 ]; then
  read -r -p "Type the project id to confirm: " answer
  [ "$answer" = "$PROJECT_ID" ] || { echo "Aborted."; exit 1; }
fi

# ---- 1. Terraform destroy -------------------------------------------------
if command -v terraform >/dev/null && g storage buckets describe "gs://$STATE_BUCKET" >/dev/null 2>&1; then
  echo "==> terraform destroy"
  tf_vars
  if tf_init; then
    # Lift deletion protection on the database only, so nothing else changes
    # before it's destroyed.
    terraform -chdir="$INFRA" apply -input=false -auto-approve "${TF_VARS[@]}" \
      -var deletion_protection=false -target=google_sql_database_instance.db \
      || echo "warning: could not lift deletion protection via Terraform; the sweep will retry"
    terraform -chdir="$INFRA" destroy -input=false -auto-approve "${TF_VARS[@]}" -var deletion_protection=false \
      || echo "warning: terraform destroy failed; continuing with the gcloud sweep"
  fi
else
  echo "==> No Terraform (or no state bucket); skipping to the gcloud sweep"
fi

# ---- 2. gcloud sweep ------------------------------------------------------
echo "==> Sweeping leftovers"
for domain in $(domain_mappings); do
  echo "    deleting domain mapping $domain"
  curl -fsS -X DELETE -H "Authorization: Bearer $(g auth print-access-token)" "$DOMAIN_API/$domain" >/dev/null
done

for budget in $(budgets); do
  echo "    deleting budget $budget"
  g billing budgets delete "$budget" --billing-project="$PROJECT_ID"
done

for pool in $(wif_pools); do
  echo "    deleting workload identity pool $pool"
  g iam workload-identity-pools delete "$pool" --location=global
done

GH_REPO="$(github_repo)"
if [ -n "$GH_REPO" ] && command -v gh >/dev/null; then
  for var in GCP_WORKLOAD_IDENTITY_PROVIDER GCP_SERVICE_ACCOUNT; do
    gh variable delete "$var" --repo "$GH_REPO" >/dev/null 2>&1 && echo "    deleted GitHub variable $var"
  done
fi

for svc in "${RUN_SERVICES[@]}"; do
  if g run services describe "$svc" --region="$REGION" >/dev/null 2>&1; then
    echo "    deleting Cloud Run service $svc"
    g run services delete "$svc" --region="$REGION"
  fi
done

for inst in $(sql_instances); do
  echo "    deleting Cloud SQL instance $inst (takes a few minutes)"
  g sql instances patch "$inst" --no-deletion-protection >/dev/null 2>&1
  g sql instances delete "$inst"
done

if g artifacts repositories describe "$AR_REPO" --location="$REGION" >/dev/null 2>&1; then
  echo "    deleting Artifact Registry repo $AR_REPO"
  g artifacts repositories delete "$AR_REPO" --location="$REGION"
fi

if g secrets describe "$SECRET" >/dev/null 2>&1; then
  echo "    deleting secret $SECRET"
  g secrets delete "$SECRET"
fi

for sa in "${SERVICE_ACCOUNTS[@]}"; do
  if g iam service-accounts describe "$sa" >/dev/null 2>&1; then
    echo "    deleting service account $sa"
    g iam service-accounts delete "$sa"
  fi
done

PROJECT_NUMBER="$(g projects describe "$PROJECT_ID" --format='value(projectNumber)' 2>/dev/null)"
if [ -n "$PROJECT_NUMBER" ]; then
  for role in "${CLOUDBUILD_ROLES[@]}"; do
    g projects remove-iam-policy-binding "$PROJECT_ID" \
      --member="serviceAccount:$PROJECT_NUMBER-compute@developer.gserviceaccount.com" \
      --role="$role" --condition=None >/dev/null 2>&1 || true
  done
fi

# ---- 3. Buckets -----------------------------------------------------------
for bucket in $(cloudbuild_buckets); do
  echo "    deleting Cloud Build bucket gs://$bucket"
  g storage rm -r "gs://$bucket"
done

if [ "$KEEP_STATE" -ne 1 ] && g storage buckets describe "gs://$STATE_BUCKET" >/dev/null 2>&1; then
  echo "    deleting state bucket gs://$STATE_BUCKET"
  # --all-versions: the bucket is versioned, so old state versions must go too.
  g storage rm -r --all-versions "gs://$STATE_BUCKET"
fi
rm -rf "$INFRA/.terraform" "$INFRA/.terraform.lock.hcl"

# ---- 4. Verify ------------------------------------------------------------
echo "==> Verifying"
remaining=0
check() {
  local label="$1" found="$2"
  if [ -z "$found" ]; then
    echo "  ✅ $label"
  else
    echo "  ❌ $label: $(echo "$found" | tr '\n' ' ')"
    remaining=1
  fi
}

check "Domain mappings" "$(domain_mappings)"
check "Cloud Run services" "$(g run services list --region="$REGION" --format='value(metadata.name)' 2>/dev/null | grep -xE 'api|web')"
check "Cloud SQL instances" "$(sql_instances)"
check "Artifact Registry repo" "$(g artifacts repositories list --location="$REGION" --format='value(name)' 2>/dev/null | grep -E "/$AR_REPO\$")"
check "Secrets" "$(g secrets list --format='value(name)' 2>/dev/null | grep -E "(^|/)$SECRET\$")"
check "Service accounts" "$(g iam service-accounts list --format='value(email)' 2>/dev/null | grep -E '^((api|web)-runtime|github-deployer)@')"
check "Workload identity pools" "$(wif_pools)"
check "Billing budget" "$(budgets)"
check "Cloud Build buckets" "$(cloudbuild_buckets)"
if [ "$KEEP_STATE" -ne 1 ]; then
  check "Terraform state bucket" "$(g storage buckets list --format='value(name)' 2>/dev/null | grep -x "$STATE_BUCKET")"
fi

if [ "$remaining" -ne 0 ]; then
  echo "Some resources remain. Re-run this script, or delete them in the console."
  exit 1
fi
echo "All resources deleted."
