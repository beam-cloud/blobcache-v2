# Blobcache

## Overview
Blobcache is a distributed, in-memory cache used to cache objects near workloads. It does not require an external metadata server
for each region, making it easy to spin up in various regions without performance penalties.

## Features
- **Tiered Storage**: Fast and efficient caching mechanism - both in-memory and on disk
- **Content-Addressed**: Stores data based on content hash, so workloads sharing the same data (i.e. model weights), can benefit from the distributed cache.
- **GRPC Interface**: Easily embeddable in Golang applications for seamless integration
