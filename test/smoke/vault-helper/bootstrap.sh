#!/bin/bash
#
# Description: This script will bootstrap a given habitat version on to a Linux machine.  If no version is specified,
#              it installs the latest version.
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
    start_habitat "${__HAB_SUP_START}" "${__HAB_SUP_TOPOLOGY}" "${__HAB_SUP_STRATEGY}" "${__HAB_SUP_CHANNEL}" "${__HAB_PEER}" "${__SUDO}"
    load_habitat_services "${__BASE_PATH}" "${__HAB_SERVICES}" "${__HAB_BINLINK}" "${__SUDO}"

    if [[ -n "${__VAULT_NODE}" ]]; then
        bootstrap_vault "${__VAULT_NODE}"
    else
        # Hab config apply the root / unseal tokens in to the ring so others can obtain it
        __ROOT_TOKEN="$( grep -io 'root token: .*' /home/kitchen/nohup.out | awk -F ':' '{print $2}' | xargs | sed 's/[^[:print:]]//g; s/\[0m$//g;' )"
        __UNSEAL_TOKEN="$( grep -io 'unseal key: .*' /home/kitchen/nohup.out | awk -F ':' '{print $2}' | xargs | sed 's/[^[:print:]]//g; s/\[0m$//g;' )"
        printf '[config]\ntoken = "%s"\nunseal_token = "%s"' "${__ROOT_TOKEN}" "${__UNSEAL_TOKEN}" | hab config apply vault.default "$(date +%s)"
    fi
}

function get_options() {
    TEMP="$( getopt -o hsnB:bv:P:u:l:p:H:x:St:r:c:i: --long help,sudo,vault-node:,base-path:,hab-binlink,hab-version:,hab-peer:,hab-bldr-url:,hab-license:,hab-pkgs:,hab-harts:,hab-services:,hab-sup-start,hab-sup-topology:,hab-sup-strategy:,hab-sup-channel:,hab-install-url: -n 'help' -- "$@" )"

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
    __VAULT_NODE=
    __HAB_SUP_START="false"
    __HAB_SUP_TOPOLOGY="leader"
    __HAB_SUP_STRATEGY="rolling"
    __HAB_SUP_CHANNEL="stable"
    __HAB_PEER=
    __HAB_VERSION=
    __HAB_BLDR_URL=
    __HAB_LICENSE=
    __HAB_PKGS="core/jq-static"
    __HAB_HARTS=
    __HAB_BINLINK="false"
    __HAB_INSTALL_URL="https://raw.githubusercontent.com/habitat-sh/habitat/master/components/hab/install.sh"
    __BASE_PATH="/tmp/kitchen/data"

    # See what's been passed in and set specified values appropriately
    while true; do
        case "${1}" in
            -h | --help ) __HELP="true"; shift;;
            -s | --sudo ) __SUDO="true"; shift;;
            -n | --vault-node ) __VAULT_NODE="${2}"; shift 2;;
            -B | --base-path ) __BASE_PATH="${2}"; shift 2;;
            -b | --hab-binlink ) __HAB_BINLINK="true"; shift;;
            -v | --hab-version ) __HAB_VERSION="${2}"; shift 2;;
            -P | --hab-peer ) __HAB_PEER="${2}"; shift 2;;
            -u | --hab-bldr-url ) __HAB_BLDR_URL="${2}"; shift 2;;
            -l | --hab-license ) __HAB_LICENSE="${2}"; shift 2;;
            -p | --hab-pkgs ) __HAB_PKGS="${2}"; shift 2;;
            -H | --hab-harts ) __HAB_HARTS="${2}"; shift 2;;
            -x | --hab-services ) __HAB_SERVICES="${2}"; shift 2;;
            -S | --hab-sup-start ) __HAB_SUP_START="true"; shift;;
            -t | --hab-sup-topology ) __HAB_SUP_TOPOLOGY="${2}"; shift 2;;
            -r | --hab-sup-strategy ) __HAB_SUP_STRATEGY="${2}"; shift 2;;
            -c | --hab-sup-channel ) __HAB_SUP_CHANNEL="${2}"; shift 2;;
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

# shellcheck disable=SC1090,SC2154
function load_habitat_services() {
    local __BASE_PATH="${1}"
    local __HAB_SERVICES="${2}"
    local __HAB_BINLINK="${3}"
    local __SUDO="${4}"

    local __USE_SUDO=
    if [[ "${__SUDO}" == "true" ]]; then
        __USE_SUDO="sudo -E"
    fi

    local __USE_BINLINK=
    if [[ "${__HAB_BINLINK}" == "true" ]]; then
        __USE_BINLINK="--binlink"
    fi

    if [[ -n "${__HAB_SERVICES}" ]]; then
        emit "Starting Habitat services from ${__HAB_SERVICES}..."

        for ITEM in $( jq -r '.[] | @base64' "${__BASE_PATH}/${__HAB_SERVICES}" ); do
            __LAST_BUILD="${__BASE_PATH}/$( _jq "${ITEM}" ".last_build" )"
            __USER_TOML="${__BASE_PATH}/$( _jq "${ITEM}" ".user_toml" )"
            __PKG_IDENT="$( _jq "${ITEM}" ".pkg_ident" )"
            __CHANNEL="$( _jq "${ITEM}" ".channel" )"
            __STRATEGY="$( _jq "${ITEM}" ".strategy" )"
            __TOPOLOGY="$( _jq "${ITEM}" ".topology" )"
            __BINDS="$( _jq "${ITEM}" ".binds" )"

            __PKG_NAME=
            __PKG_ARTIFACT=
            if [[ "${__LAST_BUILD}" != "${__BASE_PATH}/null" ]]; then
                # We source the LAST_BUILD file to get build metadata
                __RESULTS_DIR="$( dirname "${__LAST_BUILD}" )"

                . "${__LAST_BUILD}"

                __PKG_IDENT="${pkg_ident}"
                __PKG_NAME="${pkg_name}"
                __PKG_ARTIFACT="${pkg_artifact}"

                # Install the package
                ${__USE_SUDO} hab pkg install "${__RESULTS_DIR}/${__PKG_ARTIFACT}" "${__USE_BINLINK}"
            else
                # If we have no __LAST_BUILD, then we assume the JSON has a .pkg_ident field instead and use that
                __PKG_NAME="$( echo "${__PKG_IDENT}" | awk -F '/' '{print $2}' )"

                # Install the package
                ${__USE_SUDO} hab pkg install "${__PKG_IDENT}" "${__USE_BINLINK}"
            fi

            __USE_BINDS=
            if [[ "${__BINDS}" != "null" ]]; then
                __USE_BINDS="--bind $( echo "${__BINDS}" | jq -r '. | join(" --bind ")' )"
            fi

            # Copy over the user.toml to the correct dir
            if [[ "$( basename "${__USER_TOML}" )" != "null" ]]; then
                if [[ -e "${__USER_TOML}" ]]; then
                    if [[ ! -d "/hab/user/${__PKG_NAME}/config" ]]; then
                        ${__USE_SUDO} mkdir -p "/hab/user/${__PKG_NAME}/config"
                    fi

                    ${__USE_SUDO} cp "${__USER_TOML}" "/hab/user/${__PKG_NAME}/config/user.toml"
                else
                    emit "User TOML path was non-empty (${__USER_TOML}), but doesn't seem to exist?" "WARN"
                fi
            fi

            # (re)start the hab svc, with binds
            hab_svc_reload "${__USE_SUDO}" "${__PKG_IDENT}" "${__CHANNEL}" "${__STRATEGY}" "${__TOPOLOGY}" "${__USE_BINDS}"
        done
    fi
}

# shellcheck disable=SC2155
function hab_svc_reload() {
    local __USE_SUDO="${1}"
    local __PKG_IDENT="${2}"
    local __CHANNEL="${3}"
    local __STRATEGY="${4}"
    local __TOPOLOGY="${5}"
    local __USE_BINDS="${6}"
    local __RUNNING="$( hab svc status | grep -c "${__PKG_IDENT}" )"

    ${__USE_SUDO} hab svc unload "${__PKG_IDENT}"
    local __COUNT=0
    while (( __RUNNING > 0 )); do
        emit "Waiting for service '${__PKG_IDENT}' to stop..."

        sleep 3

        __RUNNING="$( hab svc status | grep -c "${__PKG_IDENT}" )"

        ((__COUNT++))

        if (( __COUNT > 10 )); then
            emit "Could not stop habitat service '${__PKG_IDENT}' after 30 seconds..." "ERROR"
            exit 2
        fi
    done

    # shellcheck disable=SC2086
    ${__USE_SUDO} hab svc load "${__PKG_IDENT}" --channel "${__CHANNEL}" --strategy "${__STRATEGY}" --topology "${__TOPOLOGY}" ${__USE_BINDS}

    local __STATE="$( hab svc status | grep "${__PKG_IDENT}" | awk -F ' ' '{print $4}' )"
    local __COUNT=0
    while [[ "${__STATE}" != "up" ]]; do
        emit "Waiting for service '${__PKG_IDENT}' to start..."

        __STATE="$( hab svc status | grep "${__PKG_IDENT}" | awk -F ' ' '{print $4}' )"

        sleep 3

        ((__COUNT++))

        if (( __COUNT > 10 )); then
            emit "Habitat service '${__PKG_IDENT}' did not start after 30 seconds..." "ERROR"
            exit 2
        fi
    done
}

function _jq() {
    local __PAYLOAD="${1}"
    local __FILTER="${2}"

    echo "${__PAYLOAD}" | jq -Rr '@base64d' | jq -r "${__FILTER}"
}

function start_habitat() {
    local __HAB_SUP_START="${1}"
    local __HAB_SUP_TOPOLOGY="${2}"
    local __HAB_SUP_STRATEGY="${3}"
    local __HAB_SUP_CHANNEL="${4}"
    local __HAB_PEER="${5}"
    local __SUDO="${6}"

    local __USE_SUDO=
    if [[ "${__SUDO}" == "true" ]]; then
        __USE_SUDO="sudo -E"
    fi

    local __USE_HAB_PEER=
    if [[ -n "${__HAB_PEER}" ]]; then
        __USE_HAB_PEER="--peer ${__HAB_PEER}"
    fi

    if [[ "${__HAB_SUP_START}" == "true" ]]; then
        if [[ "$( pgrep -c --full 'hab sup run' )" == "0" ]]; then
            emit "Waiting for habitat supervisor to start..."
            nohup bash -c "${__USE_SUDO} hab sup run ${__USE_HAB_PEER} --topology ${__HAB_SUP_TOPOLOGY} --strategy ${__HAB_SUP_STRATEGY} --channel ${__HAB_SUP_CHANNEL} &"
            wait_for_path "/hab/sup/default/CTL_SECRET" 30

            # Fix permissions and ownership so 'vagrant' user can interact with habitat supervisor
            ${__USE_SUDO} chmod 440 /hab/sup/default/CTL_SECRET
            ${__USE_SUDO} chown root:vagrant /hab/sup/default/CTL_SECRET
        fi
    fi
}

function wait_for_path() {
    local __PATH="${1}"
    local __MAX_WAIT="${2}"

    local COUNT=0
    while [[ ! -e ${__PATH} ]]; do
        sleep 1

        if (( COUNT >= __MAX_WAIT )); then
            emit "Path ${__PATH} did not appear after ${__MAX_WAIT} seconds..."
            break
        fi

        ((COUNT++))
    done
}

function install_habitat() {
    local __HAB_INSTALL_URL="${1}"
    local __HAB_VERSION="${2}"
    local __SUDO="${3}"

    local __USE_SUDO=
    if [[ "${__SUDO}" == "true" ]]; then
        __USE_SUDO="sudo -E"
    fi

    # Install binary
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

    # Setup hab user
    if ! grep "hab" /etc/passwd > /dev/null; then
        useradd hab && true
    fi

    # Setup hab group
    if ! grep "hab" /etc/group > /dev/null; then
        groupadd hab && true
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
            # shellcheck disable=SC2016
            find /tmp/kitchen/data -type f -name "${ITEM}" -exec bash -c "$( printf 'FILE="$1"; %s hab pkg install ${FILE} %s' "${__USE_SUDO}" "${__USE_BINLINK}" )" _ {} \;
        done
    fi
}

# This function bootstraps some secrets in to vault, and takes as input the hostname of a ring member (to obtain VAULT_TOKEN)
# and the VAULT_ADDR (to perform requests against).
#
# shellcheck disable=SC2155
function bootstrap_vault() {
    local __NODE="${1}"
    local __RING_ADDR="http://${__NODE}:9631"
    local __VAULT_ADDR="http://${__NODE}:8200/v1"
    local __VAULT_TOKEN="$( curl --silent -X GET "${__RING_ADDR}/census" | jq -r '.census_groups | .["vault.default"] | .service_config | .value | .config | .token' )"

    # Make sure vault is initialized
    local __VAULT_INITIALIZED="$( curl --silent -X GET "${__VAULT_ADDR}/sys/health" | jq -r '.initialized' )"
    if [[ "${__VAULT_INITIALIZED}" != "true" ]]; then
        emit "Vault does not appear to be initialized, cannot continue" "ERROR"
        exit 127
    fi

    # Make sure vault is unsealed
    local __VAULT_SEALED="$( curl --silent -X GET "${__VAULT_ADDR}/sys/health" | jq -r '.sealed' )"
    if [[ "${__VAULT_SEALED}" != "false" ]]; then
        emit "Vault appears to be sealed, cannot continue" "ERROR"
        exit 127
    fi

    # See if secret engine is enabled
    if [[ "$( curl --silent -X GET "${__VAULT_ADDR}/sys/mounts" -H "X-Vault-Token: ${__VAULT_TOKEN}" | jq -r '.["vault-helper/"]' )" == "null" ]]; then
        emit "Enabling default secrets kv store at ${__VAULT_ADDR}/sys/mounts/vault-helper ..."
        curl --silent -X POST "${__VAULT_ADDR}/sys/mounts/vault-helper" -H "X-Vault-Token: ${__VAULT_TOKEN}" -d '{ "type": "kv", "version": 1 }'
    fi

    # See if secrets are written to path
    __USERNAME="$( curl --silent -X GET "${__VAULT_ADDR}/vault-helper/credentials" -H "X-Vault-Token: ${__VAULT_TOKEN}" | jq -r '.data.username' )"
    __PASSWORD="$( curl --silent -X GET "${__VAULT_ADDR}/vault-helper/credentials" -H "X-Vault-Token: ${__VAULT_TOKEN}" | jq -r '.data.password' )"
    if [[ "${__USERNAME}" == "null" ]] || [[ "${__PASSWORD}" == "null" ]]; then
        emit "Writing secrets (kevin/bacon) to ${__VAULT_ADDR}/vault-helper/credentials ..."
        curl --silent -X POST "${__VAULT_ADDR}/vault-helper/credentials" -H "X-Vault-Token: ${__VAULT_TOKEN}" -d '{ "username": "kevin", "password": "bacon" }'
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
    echo "    Bootstraps habitat at a specific version on to a running system.  Requires 'curl' somewhere on the \$PATH."
    echo "    Installs specified habitat pkgs from --hab-pkgs=\"origin/pkg1 origin/pkg2 ...\" and any .hart files from"
    echo "    --hab-harts=\"results/*.hart\".  Can also take in a service file to build a collection of services"
    echo "    to run on a given machine, utilizing the last_build.env file, a user_toml setting, and specifying"
    echo "    the channel/strategy/topology and (optionally) any binds necessary. Can also specify pkg_ident if loading"
    echo "    services from a bldr directly."
    echo
    echo "Where:"
    echo "      --vault-node: The node name (hostname or IP) of the Vault node in Test Kitchen                               Default: ${__VAULT_NODE}"
    echo " --hab-install-url: A URL to a habitat install.sh script to perform installation                                   Default: ${__HAB_INSTALL_URL}"
    echo "     --hab-version: The Habitat version to install on the system (if not specified, uses latest release)           Default: ${__HAB_VERSION}"
    echo "        --hab-peer: Peer with this IP address or hostname                                                          Default: ${__HAB_PEER}"
    echo "    --hab-bldr-url: The Habitat builder URL to use (https://bldr.habitat.sh)                                       Default: ${__HAB_BLDR_URL}"
    echo "     --hab-license: Accept the habitat license using the specified license acceptor                                Default: ${__HAB_LICENSE}"
    echo "        --hab-pkgs: A space-separated list of habitat pkgs to install on the system once Habitat is installed      Default: ${__HAB_PKGS}"
    echo "       --hab-harts: A space-separated list of habitat .hart files to install on the system                         Default: ${__HAB_HARTS}"
    echo "     --hab-binlink: When install habitat pkgs from --hab-pkgs, also binlink them                                   Default: ${__HAB_BINLINK}"
    echo "   --hab-sup-start: After Habitat is installed, start the supervisor?                                              Default: ${__HAB_SUP_START}"
    echo "--hab-sup-topology: Topology for the supervisor                                                                    Default: ${__HAB_SUP_TOPOLOGY}"
    echo "--hab-sup-strategy: Update strategy for the supervisor packages                                                    Default: ${__HAB_SUP_STRATEGY}"
    echo " --hab-sup-channel: Channel to install packages from for the supervisor packages                                   Default: ${__HAB_SUP_CHANNEL}"
    echo "    --hab-services: A space-separated list of service file JSON's to load with service definitions                 Default: ${__HAB_SERVICES}"
    echo "       --base-path: The base path to where the JSON service files are located                                      Default: ${__BASE_PATH}"
    echo "            --sudo: Habitat installation script and resulting 'hab pkg install origin/pkg1' should use sudo        Default: ${__SUDO}"
}

main "$@"
