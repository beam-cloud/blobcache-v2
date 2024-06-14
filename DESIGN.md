blobcache consists of a series of in-memory caches that store content addressed binary blobs

the metadata server encodes the location and some other information specific objects
metadata structure:

blobcache:entry:<HASH> -->
    {
        "hash": <HASH>,
        "size:": uint64,
        "source": s3://oci-images-stage/myfile.bin:0,1000
        "content": []bytes OR nil
    }

blobcache:ref:<HASH> -->
    SET {blobcache-host-9f5739.tailc480d.ts.net, blobcache-host-22222.tailc480d.ts.net}

on insert, the goal is to:
    - pick a shard (maybe the closest), and store the content
    - when done, we will add a ref to that particular shard
    - we will also INC the ref count