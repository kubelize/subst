#!/bin/bash

set -e

echo "================================"
echo "Subst Performance Comparison Test"
echo "================================"
echo ""
echo "Comparing:"
echo "  Old: ghcr.io/kubelize/subst:v1.0.1 (render + spruce on testing/legacy)"
echo "  New: kubelize/subst-cmp:v1.1.0 (render + gomplate on testing/new)"
echo ""

# Test directories - old version tests legacy, new version tests new
OLD_TEST_DIRS=(
  "examples/basic-substitution"
  "examples/ejson-substitution"
  "testing/legacy/clusters/vclusters/prod/ci-common-prod"
)

NEW_TEST_DIRS=(
  "examples/basic-substitution"
  "examples/ejson-substitution"
  "testing/new/clusters/vclusters/prod/ci-common-prod"
)

# Pull images first
echo "Pulling Docker images..."
docker pull ghcr.io/kubelize/subst:v1.0.1 >/dev/null 2>&1 || echo "Warning: Could not pull v1.0.1"
docker pull kubelize/subst-cmp:v1.1.0 >/dev/null 2>&1 || echo "Warning: Could not pull v1.1.0"
echo ""

# Function to run old version
run_old_version() {
  local dir=$1
  local runs=$2
  local total_time=0
  
  for i in $(seq 1 $runs); do
    start=$(date +%s%N)
    docker run --rm -v "$(pwd)/$dir:/workspace" -w /workspace \
      ghcr.io/kubelize/subst:v1.0.1 render . > /dev/null 2>&1 || true
    end=$(date +%s%N)
    elapsed=$(( (end - start) / 1000000 ))  # Convert to milliseconds
    total_time=$((total_time + elapsed))
  done
  
  echo $((total_time / runs))
}

# Function to run new version
run_new_version() {
  local dir=$1
  local runs=$2
  local total_time=0
  
  for i in $(seq 1 $runs); do
    start=$(date +%s%N)
    docker run --rm -v "$(pwd)/$dir:/workspace" -w /workspace \
      kubelize/subst-cmp:v1.1.0 render . > /dev/null 2>&1 || true
    end=$(date +%s%N)
    elapsed=$(( (end - start) / 1000000 ))  # Convert to milliseconds
    total_time=$((total_time + elapsed))
  done
  
  echo $((total_time / runs))
}

# Run tests
RUNS=5
echo "Running $RUNS iterations per test directory..."
echo ""

for i in "${!OLD_TEST_DIRS[@]}"; do
  old_dir="${OLD_TEST_DIRS[$i]}"
  new_dir="${NEW_TEST_DIRS[$i]}"
  
  if [ ! -d "$old_dir" ]; then
    echo "⚠️  Skipping (old dir not found): $old_dir"
    echo ""
    continue
  fi
  
  if [ ! -d "$new_dir" ]; then
    echo "⚠️  Skipping (new dir not found): $new_dir"
    echo ""
    continue
  fi
  
  echo "📁 Testing: ${old_dir} vs ${new_dir}"
  echo "   Running old version (v1.0.1) on $old_dir..."
  old_time=$(run_old_version "$old_dir" $RUNS)
  
  echo "   Running new version (v1.1.0) on $new_dir..."
  new_time=$(run_new_version "$new_dir" $RUNS)
  
  # Calculate difference
  if [ $old_time -gt 0 ]; then
    diff=$((new_time - old_time))
    percent=$(( (diff * 100) / old_time ))
    
    echo ""
    echo "   Results (average of $RUNS runs):"
    echo "   Old (v1.0.1): ${old_time}ms"
    echo "   New (v1.1.0): ${new_time}ms"
    
    if [ $diff -lt 0 ]; then
      echo "   ✅ Improvement: ${diff#-}ms faster (${percent#-}% faster)"
    elif [ $diff -gt 0 ]; then
      echo "   ⚠️  Regression: ${diff}ms slower (${percent}% slower)"
    else
      echo "   ✅ Same performance"
    fi
  else
    echo "   ⚠️  Old version failed or returned 0ms"
    echo "   New (v1.1.0): ${new_time}ms"
  fi
  
  echo ""
done

echo "================================"
echo "Test Complete"
echo "================================"
