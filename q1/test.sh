#!/bin/bash

trap 'echo "Cleaning up all child processes..."; pkill -P $$; exit' SIGINT SIGTERM EXIT

# Define the number of backend servers and clients
NUM_SERVERS=5
NUM_CLIENTS=10
REQUEST_TYPES=(0 1 2)  # Different request types
POLICY="PF"

# Start the Load Balancer
echo "Starting Load Balancer..."
make server POLICY=$POLICY &
LB_PID=$! 
sleep 2  # Give some time for the load balancer to initialize

# Start Backend Servers
echo "Starting $NUM_SERVERS Backend Servers..."

SERVER_PIDS=()
NUM_CPUS=8

for ((i=1; i<=NUM_SERVERS; i++)); do
    make backend &   # Start the backend server in the background
    pid=$!              # Capture the process ID correctly
    cpu=$((i % NUM_CPUS)) # Correct arithmetic for CPU assignment
    taskset -cp $cpu $pid  # Bind process to a specific CPU
    SERVER_PIDS+=($pid)  # Append the correct PID to the array
done
sleep 4  # Allow servers to initialize

# Run Clients & Measure Response Times
echo "Running $NUM_CLIENTS client requests..."

START_TIME=$(date +%s.%N)  # Start time

CLIENT_PIDS=()
for ((i=1; i<=NUM_CLIENTS; i++)); do
    REQ_TYPE=${REQUEST_TYPES[$((i % ${#REQUEST_TYPES[@]}))]}  # Rotate through 0,1,2
    make client TASK=$REQ_TYPE > /dev/null &
    CLIENT_PIDS+=($!)
done

for PID in "${CLIENT_PIDS[@]}"; do
    wait $PID
done

END_TIME=$(date +%s.%N)  # End time

# Calculate throughput
TOTAL_TIME=$(echo "($END_TIME - $START_TIME)" | bc)
THROUGHPUT=$(echo "$NUM_CLIENTS / $TOTAL_TIME" | bc -l)


echo "Load Test Results:"
echo "Total Requests: $NUM_CLIENTS"
echo "Total Time: $TOTAL_TIME sec"
echo "Throughput: $THROUGHPUT requests/sec"
# echo "Average Completion Time: $AVG_TIME sec"

# Cleanup: Stop Servers and Load Balancer
echo "Stopping Backend Servers..."

for pid in "${SERVER_PIDS[@]}"; do
   kill -s SIGINT $pid
done

echo "Stopping Load Balancer..."
kill -s SIGINT $LB_PID

wait
echo "Test Completed. Servers stopped."

# Cleanup log
rm -f times.log
