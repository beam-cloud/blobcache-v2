# blobcache

## Overview
A very simple in-memory cache used as a content-addresse storage system. Exposes a GRPC server that can be embedded directly in a golang application. Persistence is just backed by disk. There is no data replication - the main use for this cache is to store large blobs of data for fast lookup by a distributed filesystem.