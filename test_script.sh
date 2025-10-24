#!/bin/bash

# Test script for GitHub step with hardcoded values
# Modify the values below to test with your specific configuration

# Required environment variables
export GITHUB_PAT="" #insert token
export build_number="3"
export version="5.26.1"
export github_organization="roadtrippers"
export github_repo="roadtrippers-ios"
export github_labels_to_remove="needs build"
export github_labels_to_add="needs qa"
export github_username_list="dejanristic"
export BITRISE_GIT_BRANCH="dr_bitrise_test" #insert branch
export GIT_CLONE_COMMIT_HASH="abc123abc" #insert hash

# Optional: Set to empty string if not needed
# export github_labels_to_remove=""
# export github_labels_to_add=""
# export github_username_list=""

echo "Running GitHub step with test values..."
echo "======================================"
echo "Build Number: $build_number"
echo "Version: $version"
echo "Organization: $github_organization"
echo "Repository: $github_repo"
echo "Branch: $BITRISE_GIT_BRANCH"
echo "Commit: $GIT_CLONE_COMMIT_HASH"
echo "Labels to Remove: $github_labels_to_remove"
echo "Labels to Add: $github_labels_to_add"
echo "Usernames: $github_username_list"
echo "======================================"

# Run the main script
go run main.go
