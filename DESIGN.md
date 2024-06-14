## blobcache metadata storage

Blobcache consists of a series of in-memory caches that store content addressed binary blobs across nodes. The metadata server encodes the location and some other information specific objects.

---
### redis key structure

blobcache:entry:<HASH> -->
    {
        "hash": <HASH>,
        "size:": uint64,
        "source": s3://oci-images-stage/myfile.bin,0-1000
        "content": []bytes OR nil
    }

blobcache:location:<HASH> -->
    SET {blobcache-host-9f5739.tailc480d.ts.net, blobcache-host-22222.tailc480d.ts.net}

blobcache:ref:hash -->
    INT (inc/dec) ref on SET/EVICT

---
### client requests

on insert:
    - pick a shard (maybe the closest), and store the content
    - when done, we will add a ref to that particular shard
    - we will also INC the ref count

on evict:
    - convert key to hash
    - decrement ref 
    - srem location
    - evict all other chunks

on retrieve:
    - TODO