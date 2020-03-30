#!/bin/bash -e

find . -type f -name '*.pb.*.go' -o -name '*.pb.go' -a ! -name 'message.pb.go' -delete
PROTOS=$(find . -type f -name '*.proto')

for PROTO in $PROTOS; do
  echo $PROTO
  protoc -I./ -I$(dirname $PROTO) --go_out=plugins=grpc,paths=source_relative:. --micro_out=paths=source_relative:. $PROTO
done
