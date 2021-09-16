# shellcheck shell=bash disable=SC1008,SC1090,SC1091,SC1117,SC2034,SC2039,SC2140,SC2148,SC2153,SC2154,SC2164,SC2239
pkg_name=vault-helper
pkg_origin=indellient
pkg_license=('Indellient Proprietary')
pkg_version="0.1.7"
pkg_bin_dirs=(bin)
pkg_deps=(core/glibc)
pkg_build_deps=(
  core/go
  core/which
  core/gcc
  core/glibc
  core/curl
  core/git
  core/shellcheck
)

do_setup_environment() {
  GOPATH="${CACHE_PATH}"
  set_runtime_env GOPATH "${CACHE_PATH}"
  GOBIN="${pkg_prefix}/bin"
  set_runtime_env GOBIN "${GOBIN}"

  REPO_PATH="${GOPATH}/src/github.com/Indellient/${pkg_name}"
  set_runtime_env REPO_PATH "${REPO_PATH}"
  __GO_LDFLAGS="-X \"main.BuildVersion=${pkg_version}\" -X \"main.BuildTimestamp=$( date --rfc-email )\""
  set_runtime_env __GO_LDFLAGS "${__GO_LDFLAGS}"
}

do_prepare() {
  mkdir -p "${REPO_PATH}"
  cp -r "${PLAN_CONTEXT}"/../* "${REPO_PATH}"/
}

do_clean() {
  rm -rf "${REPO_PATH}"
}

do_build() {
  pushd "${REPO_PATH}" &>/dev/null || exit 1
    go build                     \
      -compiler='gc'             \
      -ldflags="${__GO_LDFLAGS}" \
      main.go
  popd &>/dev/null || exit 1
}

do_check() {
  shellcheck "${PLAN_CONTEXT}"/../test/smoke/vault-helper/*.sh
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.42.1
  PATH="${PATH}:$(go env GOPATH)/bin" golangci-lint run

  # Perform unit tests
  build_line "Running go unit tests for vault..."
  go test -race github.com/Indellient/vault-helper/pkg/vault
}

do_install() {
  pushd "${REPO_PATH}" &>/dev/null || exit 1
    go install
  popd &>/dev/null || exit 1
}
