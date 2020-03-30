#!/bin/bash -e

PROTOS=$(find . -type f -name '*.proto')

for PROTO in $PROTOS; do
  echo $PROTO
  protoc -I./ -I$(dirname $PROTO) --micro_out=paths=source_relative:. $PROTO
done
