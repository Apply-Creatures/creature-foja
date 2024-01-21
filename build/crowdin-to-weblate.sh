#!/bin/bash
# Copyright 2024 The Forgejo Authors
# SPDX-License-Identifier: MIT

D=/tmp/crowdin-to-weblate
mkdir -p $D

function checkout() {
    if test -d $D/gitea ; then
        git -C $D/gitea reset --hard
        return
    fi

    git clone --depth 1 https://github.com/go-gitea/gitea $D/gitea
}

function replace() {
    go run build/merge-forgejo-locales.go $D/gitea/options/locale
    cp -a $D/gitea/options/locale/* options/locale
}

function run() {
    checkout
    replace
}

"$@"
