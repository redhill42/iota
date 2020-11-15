#!/usr/bin/env bash

PROJECT=github.com/cloudway/platform

# Downloads dependencies into vendor/ directory
mkdir -p vendor
export GOPATH="$GOPATH:${PWD}/vendor"

find='find'
if [ "$(go env GOHOSTOS)" = 'windows' ]; then
    find='/usr/bin/find'
fi

clone() {
    local vcs="$1"
    local pkg="$2"
    local rev="$3"
    local url="$4"

    : ${url:=https://$pkg}
    local target="vendor/src/$pkg"

    echo -n "$pkg @ $rev: "

    if [ -d "$target" ]; then
        echo -n 'rm old, '
        rm -rf "$target"
    fi

    echo -n 'clone, '
    case "$vcs" in
      git)
        if [ ${#rev} -ne 40 ]; then
          git clone --depth=1 --branch="$rev" --no-checkout "$url" "$target"
        else
          git clone --no-checkout "$url" "$target"
        fi
        ( cd "$target" && git checkout --quiet "$rev" && git reset --quiet --hard "$rev" )
        ;;
      hg)
        hg clone --updaterev "$rev" "$url" "$target"
        ;;
    esac

    echo -n 'rm VCS, '
    ( cd "$target" && rm -rf .{git,hg} )

    echo -n 'rm vendor, '
    ( cd "$target" && rm -rf vendor Godeps/_workspace )

    echo done
}

# get an ENV from the Dockerfile with support for multiline values
_dockerfile_env() {
    local e="$1"
    awk '
        $1 == "ENV" && $2 == "'"$e"'" {
            sub(/^ENV +([^ ]+) +/, "");
            inEnv = 1;
        }
        inEnv {
            if (sub(/\\$/, "")) {
                printf "%s", $0;
                next;
            }
            print;
            exit;
        }
    ' ${DOCKER_FILE:="Dockerfile"}
}

clean() {
    local packages=(
        "${PROJECT}/cmd/cwman"
        "${PROJECT}/cmd/cwctl"
        "${PROJECT}/cmd/cwcli"
    )
    local platforms=( ${CLOUDWAY_ENGINE_OSARCH:="linux/amd64"} $(_dockerfile_env CLOUDWAY_CROSSPLATFORMS) )
    local buildTags="$(_dockerfile_env CLOUDWAY_BUILDTAGS)"
    local buildTagsCombos=(
        ''
        "$buildTags"
    )

    echo

    echo -n 'collecting import graph, '
    local IFS=$'\n'
    local imports=( $(
        for platform in "${platforms[@]}"; do
            export GOOS="${platform%/*}";
            export GOARCH="${platform##*/}";
            for buildTags in "${buildTagsCombos[@]}"; do
                go list -e -tags "$buildTags" -f '{{join .Deps "\n"}}' "${packages[@]}"
                go list -e -tags "$buildTags" -f '{{join .TestImports "\n"}}' "${packages[@]}"
            done
        done | grep -vE "^${PROJECT}" | sort -u
    ) )
    imports=( $(go list -e -f '{{if not .Standard}}{{.ImportPath}}{{end}}' "${imports[@]}") )
    unset IFS

    echo -n 'pruning unused packages, '
    for import in "${imports[@]}"; do
        [ "${#findArgs[@]}" -eq 0 ] || findArgs+=( -or )
        findArgs+=( -path "vendor/src/$import" )
    done

    local IFS=$'\n'
    local prune=( $($find vendor -depth -type d -not '(' "${findArgs[@]}" ')') )
    unset IFS
    for dir in "${prune[@]}"; do
        $find "$dir" -maxdepth 1 -not -type d -not -name 'LICENSE*' -not -name 'COPYING*' -exec rm -f '{}' ';'
        rmdir "$dir" 2>/dev/null || true
    done

    echo -n 'pruning unused files, '
    $find vendor -type f -name '*_test.go' -exec rm '{}' ';'
    $find vendor -type f -name 'Vagrantfile' -exec rm '{}' ';'

    # These are the files that are left over after fix_rewritten_imports is run
    echo -n 'pruning .orig files, '
    $find vendor -type f -name '*.orig' -exec rm '{}' ';'

    mv vendor/src/* vendor
    rm -rf vendor/src
    echo done
}

# Fix up hard-coded imports that refer to Godeps paths so they'll work with our vendoring
fix_rewritten_imports () {
    local pkg="$1"
    local remove="${pkg}/Godeps/_workspace/src"
    local target="vendor/src/$pkg"

    echo "$pkg: fixing rewritten imports"
    $find "$target" -name \*.go -exec sed -i'.orig' -e "s|\"${remove}|\"|g" {} \;
}
