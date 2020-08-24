# shellcheck shell=bash disable=SC2034,SC2155,SC2154
pkg_name=vault-helper
pkg_origin=indellient
pkg_version="0.1.7"
pkg_bin_dirs=(bin)
pkg_deps=(core/glibc)
pkg_build_deps=(core/go/1.12.9 core/which core/gcc core/glibc core/curl core/git core/shellcheck)

do_download(){
    return 0
}

do_before() {
    # Setup environment
    export GOROOT="$(pkg_path_for go)"
    export GOPATH="$(pwd)/../"
    export DEP_VERSION="0.5.0"
    export GOMETALINTER_VERSION="2.0.11"

    # Run shellcheck first
    shellcheck ./*.sh || return 1
    shellcheck ../test/smoke/vault-helper/*.sh || return 1

    pushd "../bin" > /dev/null || return 1
    # Download dep
    if [[ ! -e "dep" ]]; then
        build_line "Setting up dep..."
        curl --silent -LL -X GET https://github.com/golang/dep/releases/download/v${DEP_VERSION}/dep-linux-amd64 -o dep
        chmod +x dep
    fi

    # Download gometalinter
    if [[ ! -e "gometalinter" ]]; then
        build_line "Setting up gometalinter..."
        curl --silent -LL -X GET https://github.com/alecthomas/gometalinter/releases/download/v${GOMETALINTER_VERSION}/gometalinter-${GOMETALINTER_VERSION}-linux-amd64.tar.gz -O
        tar --strip-components=1 -zxf gometalinter-${GOMETALINTER_VERSION}-linux-amd64.tar.gz
        rm -f gometalinter-${GOMETALINTER_VERSION}-linux-amd64.tar.gz
    fi
    popd > /dev/null || return 1

}

do_build() {
    # Setup environment
    export GOROOT="$(pkg_path_for go)"
    export GOPATH="$(pwd)"
    export BUILD_VERSION="${pkg_version:-99.99.999}"
    export BUILD_TIMESTAMP="${VAULT_HELPER_BUILD_TIMESTAMP:-$( date --rfc-email )}"
    export BUILD_OS="linux windows"
    export BUILD_ARCH="amd64"
    export BINARY_NAME="$pkg_name"

    # Build time LD Flags
    __GO_LDFLAGS="$( printf -- '-X "main.BuildVersion=%s" -X "main.BuildTimestamp=%s"' "${BUILD_VERSION}" "${BUILD_TIMESTAMP}" )"

    # Setup dependencies
    build_line "Setting up package dependencies ..."
    for _GOPKGDIR in src/* ; do
        [[ -d "${_GOPKGDIR}" ]] || break    # Break out early if there's no matches

        _GOPKGNAME="$( basename "${_GOPKGDIR}" )"

        pushd "${_GOPKGDIR}" > /dev/null || return 1
        if [[ -e "Gopkg.toml" ]]; then
            build_line "    --> dep ensure ${_GOPKGDIR} ..."
            ../../bin/dep ensure
        else
            build_line "    --> dep init ${_GOPKGDIR} ..."
            ../../bin/dep init
        fi
        popd > /dev/null || return 1
    done

    # Run gometalinter.v2 --fast
    build_line "Running gometalinter in $(pwd) ..."
    bin/gometalinter --fast src

    # Perform unit tests
    build_line "Running go unit tests ..."
    for _GOPKGDIR in src/*; do
        [[ -d "${_GOPKGDIR}" ]] || break    # Break out early if there's no matches

        _GOPKGNAME="$( basename "${_GOPKGDIR}" )"

        build_line "    --> go test -race ${_GOPKGNAME} -v ..."

        if ! go test -race "${_GOPKGNAME}" -v; then
            exit $?
        fi
    done

    # Perform debug build with -race
    build_line "Performing debug build(s) ..."
    build_line "    --> go build linux amd64 -race ${BINARY_NAME}-linux-amd64-race ..."
    GOOS=linux GOARCH=amd64 go build             \
        -o="bin/${BINARY_NAME}-linux-amd64-race" \
        -pkgdir="./pkg"                          \
        -compiler='gc'                           \
        -ldflags="${__GO_LDFLAGS}"               \
        -race                                    \
        main.go

    # Perform the build
    build_line "Performing build(s) ..."
    for OS in ${BUILD_OS}; do
        export GOOS="${OS}"

        for ARCH in ${BUILD_ARCH}; do
            export GOARCH="${ARCH}"

            OUT="${BINARY_NAME}-${OS}-${ARCH}"
            if [[ "${OS}" == "windows" ]]; then
                OUT="${OUT}.exe"
            fi

            build_line "    --> go build ${OS} ${ARCH} ${OUT} ..."
            go build                       \
                -o="bin/${OUT}"            \
                -pkgdir="./pkg"            \
                -compiler='gc'             \
                -ldflags="${__GO_LDFLAGS}" \
                main.go
        done
    done
}

# I am lazy, so if DO_INSTALL is false, we skip installing binaries so we don't have to wait for habitat to tar it up
# after a build is done, when all I need is access to the built binary.
do_install() {
    if [[ "${DO_INSTALL}" == "true" ]] || [[ -z "${DO_INSTALL}" ]]; then
        build_line "Installing $pkg_name{.exe,-race} binaries in habitat pkg ..."
        install -D "$PLAN_CONTEXT/../bin/$pkg_name-linux-amd64" "$pkg_prefix/bin/$pkg_name"
        install -D "$PLAN_CONTEXT/../bin/$pkg_name-linux-amd64-race" "$pkg_prefix/bin/$pkg_name-race"
        install -D "$PLAN_CONTEXT/../bin/$pkg_name-windows-amd64.exe" "$pkg_prefix/bin/$pkg_name.exe"
    else
        build_line "Skipping install of $pkg_name{.exe,-race} binaries ..."
    fi
}
