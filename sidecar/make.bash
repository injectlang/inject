#!/bin/bash -x
#
set -e -o pipefail

function build
{
  [[ ! -d build ]] && mkdir build
  go build -o build/configd ./cmd/configd
  go build -o build/decrypt_string ./cmd/decrypt_string
  go build -o build/encrypt_string ./cmd/encrypt_string
  go build -o build/add_pubkey_to_config ./cmd/add_pubkey_to_config
  go build -o build/yamlwalk ./cmd/yamlwalk
  go build -o build/hcltool ./cmd/hcltool
}

function main
{
  build
}

main $*
