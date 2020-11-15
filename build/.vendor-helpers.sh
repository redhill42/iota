#!/usr/bin/env bash

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
    local target="vendor/$pkg"

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
