#!/bin/bash
#
# This cleans out the local build env to make sure it's in a pristine state
#
# Run INSIDE habitat studio
#

#
# Remove all 'bin' files
#
if [ -d "bin" ]; then
    rm -f bin/*
fi

#
# Remove all 'pkg' files/folders
#
if [ -d "pkg" ]; then
    rm -rf pkg/*
fi

#
# Remove local results dir
#
if [ -d "results" ]; then
    rm -rf results
fi

#
# Remove go-dep vendored dirs
#
if [ -d "src" ] ; then
    for X in $( ls -1d src/* ); do
        if [ -d "${X}/vendor" ]; then
            rm -rf ${X}/vendor
        fi
    done
fi
