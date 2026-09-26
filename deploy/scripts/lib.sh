# Shared settings and helpers for bootstrap.sh, deploy.sh and teardown.sh.
# Sourced, not run. Every gcloud call goes through g(), which passes --project
# explicitly, so the scripts never change your gcloud default project.

PROJECT_ID="${PROJECT_ID:-lemonade-app-509618}"
REGION="${REGION:-us-central1}"
STATE_BUCKET="${PROJECT_ID}-tfstate"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
INFRA="$ROOT/deploy/infra"
# Must match the Artifact Registry repo in infra/main.tf.
REGISTRY="$REGION-docker.pkg.dev/$PROJECT_ID/app"

g() { gcloud --project="$PROJECT_ID" --quiet "$@"; }

require() {
  local cmd
  for cmd in "$@"; do
    command -v "$cmd" >/dev/null || { echo "error: $cmd is not installed" >&2; exit 1; }
  done
}

# Billing account linked to the project (e.g. 015E56-C42150-9C05C1), empty if none.
billing_account() {
  g billing projects describe "$PROJECT_ID" --format='value(billingAccountName)' 2>/dev/null \
    | sed 's|^billingAccounts/||'
}

# owner/name of the GitHub origin remote, empty if it isn't on GitHub.
github_repo() {
  git -C "$ROOT" remote get-url origin 2>/dev/null \
    | sed -nE 's#^(https://github\.com/|git@github\.com:)([^/]+/[^/]+)$#\2#p' | sed 's/\.git$//'
}

tf_init() {
  terraform -chdir="$INFRA" init -input=false -reconfigure -backend-config="bucket=$STATE_BUCKET" >/dev/null
}

# Common -var flags for terraform apply/destroy.
tf_vars() {
  TF_VARS=(-var "project_id=$PROJECT_ID" -var "region=$REGION" -var "billing_account=$(billing_account)" -var "github_repo=$(github_repo)")
}

# Timestamp, plus the git commit when run from a git checkout.
new_tag() {
  local tag sha
  tag="$(date +%Y%m%d-%H%M%S)"
  sha="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null)" && tag="$tag-$sha"
  echo "$tag"
}

source_dir() {
  case "$1" in
    api) echo "$ROOT/lemonade-api" ;;
    web) echo "$ROOT/lemonade-web" ;;
  esac
}

# build_one <service> <tag>: builds and pushes $REGISTRY/<service>:<tag> on Cloud Build.
# On a fresh project, the Cloud Build API and IAM grants can take a few minutes
# to take effect and fail with PERMISSION_DENIED meanwhile. The first build also
# creates the <project>_cloudbuild source bucket, which may not be visible yet
# ("bucket does not exist"). Retry those only.
build_one() {
  local svc="$1" tag="$2" attempt out
  out="$(mktemp)"
  for attempt in 1 2 3 4 5; do
    g builds submit "$(source_dir "$svc")" --region="$REGION" --tag "$REGISTRY/$svc:$tag" 2>&1 | tee "$out" && return 0
    grep -q -E 'PERMISSION_DENIED|bucket does not exist' "$out" || return 1
    echo "attempt $attempt: new project not ready yet; retrying in 30s"
    sleep 30
  done
  return 1
}

# build_images <tag> <service>...: builds the services in parallel.
build_images() {
  local tag="$1" svc log_dir failed=0
  shift
  log_dir="$(mktemp -d)"
  local -a pids=()
  for svc in "$@"; do
    build_one "$svc" "$tag" >"$log_dir/$svc.log" 2>&1 &
    pids+=($!)
  done
  local i=0
  for svc in "$@"; do
    wait "${pids[$i]}" || { echo "$svc build failed:"; tail -n 40 "$log_dir/$svc.log"; failed=1; }
    i=$((i + 1))
  done
  [ "$failed" -eq 0 ] || return 1
  echo "    built $* at tag $tag"
}

# deploy_image <service> <tag>: points the Cloud Run service at the image.
# Skipped when it already runs that image, so no empty revision is created.
deploy_image() {
  local svc="$1" image="$REGISTRY/$1:$2" current
  current="$(g run services describe "$svc" --region="$REGION" \
    --format='value(spec.template.spec.containers[0].image)')"
  if [ "$current" = "$image" ]; then
    echo "    $svc already runs $image"
    return 0
  fi
  echo "    deploying $image"
  g run services update "$svc" --region="$REGION" --image="$image" >/dev/null
}
