#!/bin/bash

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=./ --go-grpc_opt=paths=source_relative ./blobcache.proto

flatc --go --grpc --gen-object-api -o . ./blobcache.fbs
