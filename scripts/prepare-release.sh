#!/usr/bin/env bash
#
# prepare-release.sh
# (Note: This should be called by `make release/prepare` because it depends
#   on several variables set by the Makefile)
#
# Performs release preparation tasks:
#   - Creates a release branch
#   - Renames "LATEST" section to the new version number
#   - Adds new "LATEST" entry to the changelog
#
##############################################
set -Eeuo pipefail

if [[ -z "${NEW_VERSION:-}" ]]; then
    echo "[ERROR] NEW_VERSION environment variable not defined." >&2
    exit 1
fi

# Script called from within a git repo?
if [[ $(git rev-parse --is-inside-work-tree &>/dev/null) -ne 0 ]]; then
    echo "[ERROR] Current directory (${SRCDIR}) is not a git repository" >&2
    exit 1
fi

REPO_ROOT=$(git rev-parse --show-toplevel)
CHANGELOG_FILENAME=${CHANGELOG:-"CHANGELOG.md"}

# normalize version by removing `v` prefix
VERSION_NUM=${NEW_VERSION/#v/}
RELEASE_BRANCH=$(printf "release/v%s" "${VERSION_NUM}")

function updateChangelog() {
    local tmpfile

    trap '[ -e "${tmpfile}" ] && rm "${tmpfile}"' RETURN

    local changelogFile
    changelogFile=$(printf "%s/%s" "${REPO_ROOT}" "${CHANGELOG_FILENAME}")

    # create Changelog file if not exists
    if ! [[ -f "${REPO_ROOT}/${CHANGELOG_FILENAME}" ]]; then
        touch "${REPO_ROOT}/${CHANGELOG_FILENAME}" && \
        git add "${REPO_ROOT}/${CHANGELOG_FILENAME}"
    fi

    tmpfile=$(mktemp)

    # Replace "Latest" in the top-most changelog block with new version
    # Then push a new "latest" block to top of the changelog
    awk 'NR==1, /---/{ sub(/START\/LATEST/, "START/v'${VERSION_NUM}'"); sub(/# Latest/, "# v'${VERSION_NUM}'") } {print}' \
     "${changelogFile}" > "${tmpfile}"

    # Inserts "Latest" changelog HEREDOC at the top of the file
    cat - "${tmpfile}" << EOF > "${REPO_ROOT}/${CHANGELOG_FILENAME}"
[//]: # (START/LATEST)
# Latest

## Features
  * A user-friendly description of a new feature. {issue-number}

## Fixes
 * A user-friendly description of a fix. {issue-number}

## Security
 * A user-friendly description of a security fix. {issue-number}

---

EOF
}

function _main() {

    # Stash version changes
    git stash push &>/dev/null

    if ! git checkout -b "${RELEASE_BRANCH}" origin/"${MAIN_BRANCH:-main}"; then
        echo "[ERROR] Could not check out release branch." >&2
        git stash pop &>/dev/null
        exit 1
    fi

    # Add the version changes to release branch
    git stash pop &>/dev/null

    updateChangelog

    cat << EOF

[SUCCESS] Changelog updated & release branch created:
    New Version:    ${NEW_VERSION}
    Release Branch: ${RELEASE_BRANCH}

Next steps:
    1. Edit the changelog notes in ${CHANGELOG_FILENAME}
    2. Commit changes to the release branch
    3. Push changes to remote => git push origin ${RELEASE_BRANCH}

EOF
    exit 0
}

_main
