#!/bin/bash
declare -A ejson_keys

## -- Default Variables

# General Plugin Configuration
TMP_DIR=${TMP_DIR:-./tmp}
KUSTOMIZE_BUILD_OPTIONS=""

# Render Configuration




ROOT_DIRECTORY=${ARGOCD_ENV_ROOT_DIRECTORY:-.}
EXTRA_DIRECTORIES=${ARGOCD_ENV_EXTRA_DIRECTORIES}
EJSON_FILE_REGEX=${ARGOCD_ENV_EJSON_FILE_REGEX:-"*.ejson"}

#: ${ARGOCD_APP_NAMESPACE:?"Namespace for ArgoCD Application is required"}
EJSON_SECRET=${ARGOCD_ENV_SECRET}
EJSON_INLINE_KEYS=""

VAR_FILE_REGEX=${ARGOCD_ENV_VAR_FILE_REGEX:-"*.vars"}

## -- Help Context
show_help() {
cat << EOF
Usage: ${0##*/} [-h] [-p] "ejson_key" [-s] "directory" [-m] "merge_directory" [-k] "secret_key" [-f] "dir/filename" [-r]
    -s secret          EJSON Secret name with ejson private keys
    -n secret_ns       EJSON Secret namespace
    -p ejson_keys      Add EJSON private key to decrypt ejson files (Comma seperated for multiple keys)
    -D directory       Root directory to search for files to decrypt [${ROOT_DIRECTORY}]
    -m merge_dirs      Additional comma seperated directories to merge secrets from
    -e file_regex      Regex to match files to decrypt [Default: ".*\.ejson$"]
    -v vars_regex      Regex to match files to merge [Default: ".*\.vars$"]
    -d                 Dry Run Mode (Evaluate Build)
    -v                 Verbose Mode
    -h                 Show this context

EOF
}

# -- Script Arguments
OPTIND=1
while getopts hvdv:e:m:p:n:s: opt; do
    case $opt in
        h)
            show_help
            exit 0
            ;;
        d)  DRY_RUN=true
            ;;
        v)  DEBUG=true
            ;;
        s)  EJSON_SECRET="${OPTARG}"
            ;;
        p)  EJSON_INLINE_KEYS="${OPTARG}"
            ;;
        D)  if [ -d "${OPTARG}" ]; then
              ROOT_DIRECTORY=${OPTARG}
            else
              echo "Root directory '${OPTARG}' does not exist" && exit 1
            fi
            ;;
        *)
            show_help >&2
            exit 1
            ;;
    esac
done
shift "$((OPTIND-1))"

# Constants/States
DEBUG=${ARGOCD_ENV_DEBUG:-false}
DRY_RUN=${ARGOCD_ENV_DRY_RUN:-false}
SUBS_STRUCT="{\"subst\": { \"secrets\": {}, \"vars\": {}, \"env\": {} }}"
DATA_FILE="${TMP_DIR%/}/secrets.tmp.json"

# -- Validators

# Namespace
# ARGOCD_APP_NAMESPACE is only set, if destination.namespace is set in the ArgoCD Application

# -- Functions

# Initialize
initialize() {
  if $DEBUG; then
    rm -rf ${TMP_DIR}
  fi

  mkdir -p ${TMP_DIR%/}/build
  cat > "${DATA_FILE}" << EOF
${SUBS_STRUCT}
EOF

}

## -- Debug Logging
debug() {
  if $DEBUG; then
    echo "$(date) - $1"
  fi
}

cleanup() {
  if ! $DEBUG; then
    rm -rf ${TMP_DIR}
  fi
}
trap cleanup EXIT

# https://github.com/buttahtoast/ejsonMerger/blob/master/ejson-merger.sh
# Main Function
main() {

  # Evaluate all paths
  PATHS=". "

  # Add additional paths to decrypt
  if [[ -n "${EXTRA_DIRECTORIES}" ]]; then
    IFS=, read -ra DIR <<< "$EXTRA_DIRECTORIES"
    for v in "${DIR[@]}"; do
      path=$(echo $v | xargs)
      if [ -d "$path" ]; then
        PATHS="${PATHS} $v"
      fi
    done
  fi

  # Read Kustomization Paths
  if [ -f ./kustomization.yaml ]; then
    while read -r path; do
      if [ -d "$path" ]; then
        PATHS="${PATHS} $path"
      fi
    done <<< "$(spruce json ./kustomization.yaml | jq -r '.resources[]')"
  fi

  debug "Lookup Paths: ${PATHS}"

  ## --- Read EJSON Files
  if [ -x "$(command -v ejson)" ]; then

    # Key Array
    KEYS=()

    if ! [ -z "${EJSON_INLINE_KEYS}" ]; then
      IFS=, read -ra INLINE <<< "$EJSON_INLINE_KEYS"
      for k in "${INLINE[@]}"; do
        KEYS+=($k)
      done
    else
      # Resolve Private Keys from Secret
      if ! [ -z "${EJSON_SECRET}" ]; then
        while read -r key; do
          KEYS+=($(echo "$key" | base64 -d))
        done <<< "$(kubectl get secret test -o go-template='{{range .data}}{{.}}{{"\n"}}{{end}}')"
      fi
    fi

    SECRET_DATA="{}"
    if [ "${#KEYS[@]}" -eq 0 ]; then
      debug "No EJSON Keys given"
    else
      for secret in $(find $PATHS -mindepth 1 -maxdepth 1 -type f -name "${EJSON_FILE_REGEX}"); do
        debug "Attempt to decrypt '${secret}'";
        JSON_RESULT=$(cat "${secret}" | jq -r '.data')
        if [[ $? -eq 0 ]] && [[ $JSON_RESULT != "null" ]]; then
          decrypted=0
          for key in "${KEYS[@]}"; do
            if echo "${key}" | ejson decrypt -key-from-stdin "${secret}" &> /dev/null; then
              debug "Decrypted '${secret}'";
              data=$(echo "${key}" | ejson decrypt -key-from-stdin "${secret}" | jq -r --argjson struct "$SUBS_STRUCT"  '.data as $d | $struct | .subst.secrets = $d' | spruce merge --skip-eval "${DATA_FILE}" - | spruce json)
              echo $data > "${DATA_FILE}"
              decrypted=1;
              break;
            fi
          done
          if [ $decrypted -eq 0 ]; then
            debug "Unable to decrypt '${secret}'";
          fi
        else
          debug "Invalid JSON '${file}'. Does it have '.data' in structure (required)?";
          continue;
        fi
      done
    fi
  fi


  # -- Read Var Files
  VAR_FILES=()
  readarray -d '' VAR_FILES < <(find $PATHS -mindepth 1 -maxdepth 1 -type f -name "${VARS_FILE_REGEX}")
  if ! [ ${#VAR_FILES[@]} -eq 0 ]; then
    data=$(spruce merge --skip-eval $(printf '${DATA_FILE} %s ' "${VAR_FILES[@]}") - | spruce json | jq -r --argjson struct "$SUBS_STRUCT"  '. as $d | $struct | .subst.vars = $d')
    echo $data > "${DATA_FILE}"

    #| jq '. | del(.SECRETS) | del(.VARS)')

    # Redirect Variables to DATA
    #DATA=$(echo "$DATA" | jq --argjson vars "$vars" '.VARS = $vars')
  fi

  # -- Add Environment Variables
  ENV="{}"
  while read -r env; do
    convert_env=$(echo "$env" | sed "s/ARGOCD_ENV_//")
    ENV=$(echo $ENV | jq --arg env "$(echo $convert_env| cut -d "=" -f 1)" --arg value "$(echo $convert_env| cut -d "=" -f 2)" '.[$env] = $value')
  done <<< "$(env | grep ARGOCD_ENV_)"
  data=$(cat "${DATA_FILE}" | jq  -r --argjson ev "$ENV" '.subst.env=$ev')
  echo $data > "${DATA_FILE}"


  # -- Merge Files
  debug "Available for subsitution $(cat "${DATA_FILE}")"

  # Envsubstitution allows only
  if kustomize build ${ROOT_DIRECTORY} --load-restrictor LoadRestrictionsNone ${KUSTOMIZE_BUILD_OPTIONS} -o ${TMP_DIR%/}/build/; then
    debug "Kustomize build succeeded"
    for manifest in ${TMP_DIR%/}/build/*.yaml; do
      debug "Run Substitution for build ${manifest}"
      spruce merge "${DATA_FILE}" $manifest | spruce json | jq 'del(.subst)'
    done
  else
    debug "Kustomize build failed"
    exit 1
  fi

}

# -- Main
initialize && main