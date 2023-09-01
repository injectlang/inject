#!/bin/bash -x
#
set -e -o pipefail

function build
{
  [[ ! -d build ]] && mkdir build
  go build -o build/add_pubkey ./cmd/add_pubkey
  go build -o build/add_secret ./cmd/add_secret
  go build -o build/decrypt_string ./cmd/decrypt_string
  go build -o build/encrypt_string ./cmd/encrypt_string
  go build -o build/entrypoint ./cmd/entrypoint
  go build -o build/injectord ./cmd/injectord
  go build -o build/injwalk ./cmd/injwalk
  go build -o build/renderinj ./cmd/renderinj
}

function test
{
  local args="$*"

  go test ${args} ./...
}

function main
{
  if [[ "$1" == "test" ]]; then
    shift
    test $*
  else
    build
  fi
}

main $*
