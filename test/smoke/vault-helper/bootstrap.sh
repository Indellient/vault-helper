#!/bin/bash
#
# Description: This script will bootstrap a given habitat version on to a Linux machine.  If no version is specified,
#              it installs the latest version.  It also sets up some basic data in vault for InSpec test assertions.
#
#              The only prerequisite is that curl is on the $PATH.  Everything else is managed via `hab ...` commands.
#
# See --help for more information
#
function main() {
    # Parse CLI options
    get_options "$@"

    # Set habitat global vars
    export HAB_NONINTERACTIVE="true"
    if [[ -n "${__HAB_BLDR_URL}" ]]; then
        export HAB_BLDR_URL="${__HAB_BLDR_URL}"
    fi

    if [[ -n "${__HAB_LICENSE}" ]]; then
        export HAB_LICENSE="${__HAB_LICENSE}"
    fi

    # Install habitat, packages, hart files
    install_habitat "${__HAB_INSTALL_URL}" "${__HAB_VERSION}" "${__SUDO}"
    install_habitat_packages "${__HAB_PKGS}" "${__HAB_BINLINK}" "${__SUDO}"
    install_habitat_harts "${__HAB_HARTS}" "${__HAB_BINLINK}" "${__SUDO}"
    bootstrap_vault "${__VAULT_NODE}"
}

function get_options() {
    TEMP=$( getopt -o hsn:bv:u:l:p:H:i: --long help,sudo,vault-node:,hab-binlink,hab-version:,hab-bldr-url:,hab-license:,hab-pkgs:,hab-harts:,hab-install-url: -n 'help' -- "$@" )

    # Check to see if an invalid option was passed in. If so, show a message and exit.
    # shellcheck disable=SC2181
    if [[ $? != 0 ]]; then
        echo "Try specifying '--help'" >&2
        exit
    fi

    eval set -- "${TEMP}"

    # All possible options and their default values
    __HELP="false"
    __SUDO="false"
    __VAULT_NODE="mozart"
    __HAB_VERSION=
    __HAB_BLDR_URL=
    __HAB_LICENSE=
    __HAB_PKGS="bluepipeline/jq"
    __HAB_HARTS=
    __HAB_BINLINK="false"
    __HAB_INSTALL_URL="https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"

    # See what's been passed in and set specified values appropriately
    while true; do
        case "${1}" in
            -h | --help ) __HELP="true"; shift;;
            -s | --sudo ) __SUDO="true"; shift;;
            -n | --vault-node ) __VAULT_NODE="${2}"; shift 2;;
            -b | --hab-binlink ) __HAB_BINLINK="true"; shift;;
            -v | --hab-version ) __HAB_VERSION="${2}"; shift 2;;
            -u | --hab-bldr-url ) __HAB_BLDR_URL="${2}"; shift 2;;
            -l | --hab-license ) __HAB_LICENSE="${2}"; shift 2;;
            -p | --hab-pkgs ) __HAB_PKGS="${2}"; shift 2;;
            -H | --hab-harts ) __HAB_HARTS="${2}"; shift 2;;
            -i | --hab-install-url ) __HAB_INSTALL_URL="${2}"; shift 2;;
            -- ) shift; break;;
            * ) break ;;
        esac
    done

    # Check to see if someone is asking for help
    if [[ "${__HELP}" == "true" ]]; then
        help
        exit
    fi
}

# Using `eval` below makes me weary, but it's not directly calling an evaluated
function install_habitat() {
    local __HAB_INSTALL_URL="${1}"
    local __HAB_VERSION="${2}"
    local __SUDO="${3}"

    local __USE_SUDO=
    if [[ "${__SUDO}" == "true" ]]; then
        __USE_SUDO="sudo"
    fi

    if [[ "$( command -v hab 2>/dev/null )" == "" ]]; then
        if [[ -z "${__HAB_VERSION}" ]]; then
            # Install latest
            emit "Installing latest version of hab from ${__HAB_INSTALL_URL}..."
            curl --silent "${__HAB_INSTALL_URL}" | bash -c "${__USE_SUDO} bash"
        else
            # Install specified version
            emit "Installing hab v${__HAB_VERSION} from ${__HAB_INSTALL_URL}..."
            curl --silent "${__HAB_INSTALL_URL}" | bash -c "${__USE_SUDO} bash -s -- -v ${__HAB_VERSION}"
        fi
    fi
}

function install_habitat_packages() {
    local __HAB_PKGS="${1}"
    local __HAB_BINLINK="${2}"
    local __SUDO="${3}"

    local __USE_SUDO=
    if [[ "${__SUDO}" == "true" ]]; then
        __USE_SUDO="sudo -E"
    fi

    local __USE_BINLINK=
    if [[ "${__HAB_BINLINK}" == "true" ]]; then
        __USE_BINLINK="--binlink"
    fi

    for PKG in ${__HAB_PKGS}; do
        if [[ ! -d "/hab/pkgs/${PKG}" ]]; then
            emit "Installing habitat package ${PKG}..."
            bash -c "${__USE_SUDO} hab pkg install ${__USE_BINLINK} ${PKG}"
        fi
    done
}

function install_habitat_harts() {
    local __HAB_HARTS="${1}"
    local __HAB_BINLINK="${2}"
    local __SUDO="${3}"

    local __USE_SUDO=
    if [[ "${__SUDO}" == "true" ]]; then
        __USE_SUDO="sudo -E"
    fi

    local __USE_BINLINK=
    if [[ "${__HAB_BINLINK}" == "true" ]]; then
        __USE_BINLINK="--binlink"
    fi

    if [[ -n "${__HAB_HARTS}" ]]; then
        for ITEM in ${__HAB_HARTS}; do
            find /tmp/kitchen/data -type f -name "${ITEM}" -exec bash -c "${__USE_SUDO} hab pkg install {} ${__USE_BINLINK}" \;
        done
    fi
}

# This function bootstraps some secrets in to vault, and takes as input the hostname of a ring member (to obtain VAULT_TOKEN)
# and the VAULT_ADDR (to perform requests against).  Most of the time they are the same, but they don't necessarily have to be.
# shellcheck disable=SC2155
function bootstrap_vault() {
    local __NODE="${1}"
    local __RING_ADDR="http://${__NODE}:9631"
    local __VAULT_ADDR="https://${__NODE}/v1"
    local __VAULT_TOKEN="$( curl --silent -X GET "${__RING_ADDR}/census" | jq -r '.census_groups | .["vault.default"] | .service_config | .value | .config | .token' )"
    local __VAULT_SKIP_VERIFY="true"

    # Setup curl --insecure flag
    __CURL_INSECURE=
    if [[ "${__VAULT_SKIP_VERIFY}" == "true" ]]; then
        __CURL_INSECURE="--insecure"
    fi

    # Make sure vault is initialized
    local __VAULT_INITIALIZED="$( curl --silent "${__CURL_INSECURE}" -X GET "${__VAULT_ADDR}/sys/health" | jq -r '.initialized' )"
    if [[ "${__VAULT_INITIALIZED}" != "true" ]]; then
        emit "Vault does not appear to be initialized, cannot continue" "ERROR"
        exit 127
    fi

    # Make sure vault is unsealed
    local __VAULT_SEALED="$( curl --silent "${__CURL_INSECURE}" -X GET "${__VAULT_ADDR}/sys/health" | jq -r '.sealed' )"
    if [[ "${__VAULT_SEALED}" != "false" ]]; then
        emit "Vault appears to be sealed, cannot continue" "ERROR"
        exit 127
    fi

    # See if secret engine is enabled
    if [[ "$( curl --silent "${__CURL_INSECURE}" -X GET "${__VAULT_ADDR}/sys/mounts" -H "X-Vault-Token: ${__VAULT_TOKEN}" | jq -r '.["secret/"]' )" == "null" ]]; then
        emit "Enabling default secrets kv store at ${__VAULT_ADDR}/sys/mounts/secret ..."
        curl --silent "${__CURL_INSECURE}" -X POST "${__VAULT_ADDR}/sys/mounts/secret" -H "X-Vault-Token: ${__VAULT_TOKEN}" -d '{ "type": "kv", "version": 1 }'
    fi

    # See if secrets are written to path
    __USERNAME="$( curl --silent "${__CURL_INSECURE}" -X GET "${__VAULT_ADDR}/secret/credentials" -H "X-Vault-Token: ${__VAULT_TOKEN}" | jq -r '.data.username' )"
    __PASSWORD="$( curl --silent "${__CURL_INSECURE}" -X GET "${__VAULT_ADDR}/secret/credentials" -H "X-Vault-Token: ${__VAULT_TOKEN}" | jq -r '.data.password' )"
    if [[ "${__USERNAME}" == "null" ]] || [[ "${__PASSWORD}" == "null" ]]; then
        emit "Writing secrets (kevin/bacon) to ${__VAULT_ADDR}/secret/credentials ..."
        curl --silent "${__CURL_INSECURE}" -X POST "${__VAULT_ADDR}/secret/credentials" -H "X-Vault-Token: ${__VAULT_TOKEN}" -d '{ "username": "kevin", "password": "bacon" }'
    fi
}

# shellcheck disable=SC2155
function emit() {
    local __MESSAGE="${1}"
    local __LEVEL="${2:-INFO}"
    local __TIMESTAMP="$( date --rfc-3339=ns )"

    printf "%s - %s - %s\n" "${__TIMESTAMP}" "${__LEVEL}" "${__MESSAGE}"
}

function help() {
    echo "Usage: $( basename "${0}" ) [--sudo] --hab-pkgs=\"core/jq-static ...\" [--hab-version=\"0.79.1\" [--hab-install-url=...] [...]"
    echo
    echo "Summary:"
    echo "    Bootstraps habitat at a specific version on to a running system.  Requires 'curl' somewhere on the \$PATH, and"
    echo "    also installs specified habitat pkgs from --hab-pkgs=\"origin/pkg1 origin/pkg2 ...\""
    echo
    echo "Where:"
    echo " --hab-install-url: A URL to a habitat install.sh script to perform installation                                   Default: ${__HAB_INSTALL_URL}"
    echo "     --hab-version: The Habitat version to install on the system (if not specified, uses latest release)           Default: ${__HAB_VERSION}"
    echo "    --hab-bldr-url: The Habitat builder URL to use (https://bldr.habitat.sh)                                       Default: ${__HAB_BLDR_URL}"
    echo "     --hab-license: Accept the habitat license using the specified license acceptor                                Default: ${__HAB_LICENSE}"
    echo "        --hab-pkgs: A space-separated list of habitat pkgs to install on the system once Habitat is installed      Default: ${__HAB_PKGS}"
    echo "       --hab-harts: A space-separated list of habitat .hart files to install on the system                         Default: ${__HAB_HARTS}"
    echo "     --hab-binlink: When install habitat pkgs from --hab-pkgs, also binlink them                                   Default: ${__HAB_BINLINK}"
    echo "      --vault-node: The node name (hostname or IP) of the Vault node in Test Kitchen                               Default: ${__VAULT_NODE}"
    echo "            --sudo: Habitat installation script and resulting 'hab pkg install origin/pkg1' should use sudo        Default: ${__SUDO}"
}

main "$@"
