#!/bin/bash

SERVER="10.0.0.99"
PORT="1111"
DATA_SIZE_BYTES=$((5 * 1024 * 1024 * 1024)) # 5 GiB in bytes

start_time=$(date +%s%N)
socat -u TCP:$SERVER:$PORT CREATE:/tmp/big_file
end_time=$(date +%s%N)

# Calculate the elapsed time in seconds with decimal places
elapsed_time=$(echo "scale=3; ($end_time - $start_time) / 1000000000" | bc)

# Log the time taken
echo "Time taken: $elapsed_time seconds"

# Convert data size to MiB (1 MiB = 1024 * 1024 bytes)
data_size_mib=$(echo "scale=2; $DATA_SIZE_BYTES / (1024 * 1024)" | bc)

# Calculate transfer rate in MiB/s
transfer_rate=$(echo "scale=2; $data_size_mib / $elapsed_time" | bc)

# Log the transfer rate
echo "Transfer rate: $transfer_rate MiB/s"