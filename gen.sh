#!/bin/sh

set -ex
CURDIR=$(cd $(dirname $0); pwd)
export PATH=$PATH:.

# natsrpc and httprpc
protoc --proto_path=$CURDIR/rpc \
  -I=$CURDIR \
  --go_out=$CURDIR --go_opt=paths=source_relative \
  $CURDIR/pb/*.proto \

