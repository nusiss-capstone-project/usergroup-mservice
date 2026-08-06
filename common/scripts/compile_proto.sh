#!/bin/bash
set -euo pipefail
protoc --go_out=. --go-grpc_out=. "proto/usergroup.proto"
