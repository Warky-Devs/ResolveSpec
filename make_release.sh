#!/bin/bash

# Ask if the user wants to make a release version
read -p "Do you want to make a release version? (y/n): " make_release

if [[ $make_release =~ ^[Yy]$ ]]; then
    # Get the latest tag from git
    latest_tag=$(git describe --tags --abbrev=0 2>/dev/null)

    if [ -z "$latest_tag" ]; then
        # No tags exist yet, start with v1.0.0
        suggested_version="v1.0.0"
        echo "No existing tags found. Starting with $suggested_version"
    else
        echo "Latest tag: $latest_tag"

        # Remove 'v' prefix if present
        version_number="${latest_tag#v}"

        # Split version into major.minor.patch
        IFS='.' read -r major minor patch <<< "$version_number"

        # Increment patch version
        patch=$((patch + 1))

        # Construct new version
        suggested_version="v${major}.${minor}.${patch}"
        echo "Suggested next version: $suggested_version"
    fi

    # Ask the user for the version number with the suggested version as default
    read -p "Enter the version number (press Enter for $suggested_version): " version

    # Use suggested version if user pressed Enter without input
    if [ -z "$version" ]; then
        version="$suggested_version"
    fi

    # Prepend 'v' to the version if it doesn't start with it
    if ! [[ $version =~ ^v ]]; then
        version="v$version"
    fi

    # Get commit logs since the last tag
    if [ -z "$latest_tag" ]; then
        # No previous tag, get all commits
        commit_logs=$(git log --pretty=format:"- %s" --no-merges)
    else
        # Get commits since the last tag
        commit_logs=$(git log "${latest_tag}..HEAD" --pretty=format:"- %s" --no-merges)
    fi

    # Create the tag message
    if [ -z "$commit_logs" ]; then
        tag_message="Release $version"
    else
        tag_message="Release $version

${commit_logs}"
    fi

    # Create an annotated tag with the commit logs
    git tag -a "$version" -m "$tag_message"

    # Push the tag to the remote repository
    git push origin "$version"

    echo "Tag $version created and pushed to the remote repository."
else
    echo "No release version created."
fi
