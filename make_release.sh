#!/bin/bash

# Ask if the user wants to make a release version
read -p "Do you want to make a release version? (y/n): " make_release

if [[ $make_release =~ ^[Yy]$ ]]; then
    # Ask the user for the version number
    read -p "Enter the version number : " version

    # Prepend 'v' to the version if it doesn't start with it
    if ! [[ $version =~ ^v ]]; then
        version="v$version"
    else
        echo "Version already starts with 'v'."
    fi

    # Create an annotated tag
    git tag -a "$version" -m "Released Core $version"

    # Push the tag to the remote repository
    git push origin "$version"

    echo "Tag $version created for Core and pushed to the remote repository."
else
    echo "No release version created."
fi
