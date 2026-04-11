#!/bin/bash
set -e

PROJECT_VERSION="${PROJECT_VERSION:-1.0.0.1}"
PROJECT_VERSION_GIT="${PROJECT_VERSION_GIT:-unknown}"
SHARE_INSTALL_DESTINATION="${SHARE_INSTALL_DESTINATION:-/usr/share/keydesk}"
RUN_DIR_PATH="${RUN_DIR_PATH:-/var/run/keydesk}"
PIDFILE_PATH="${PIDFILE_PATH:-/var/run/keydesk/keydesk.pid}"
CONFIG_PATH="${CONFIG_PATH:-/etc/keydesk.conf}"

if [ -z "$PROJECT_NAME_LOWERCASE" ]; then
    echo "Error: PROJECT_NAME_LOWERCASE must be set" >&2
    exit 1
fi

clean_version=$(echo "$PROJECT_VERSION" | sed 's/^v//' | sed 's/-.*$//')
IFS='.' read -r major minor patch tweak <<< "$clean_version"

PROJECT_VERSION_SHORT="${major}.${minor}.${patch}"
PROJECT_VERSION_INTEGER="${major}${minor}${patch}${tweak}"

OUTPUT_FILE="${OUTPUT_FILE:-src/app/version/version.go}"

cat > "$OUTPUT_FILE" << EOF
package version

const ProjectName = "$PROJECT_NAME_LOWERCASE"
const VersionApp = "$PROJECT_VERSION Revision: $PROJECT_VERSION_GIT"
const VersionAppShort = "$PROJECT_VERSION_SHORT"
const VersionAppNumber = $PROJECT_VERSION_INTEGER

const ShareFolderPath = "$SHARE_INSTALL_DESTINATION"
const RunDirPath = "$RUN_DIR_PATH"
const PidFilePath = "$PIDFILE_PATH"
const ConfigPath = "$CONFIG_PATH"
EOF

echo "Generated $OUTPUT_FILE with:"
echo "  ProjectName: $PROJECT_NAME_LOWERCASE"
echo "  VersionApp: $PROJECT_VERSION Revision: $PROJECT_VERSION_GIT"
echo "  ShareFolderPath: $SHARE_INSTALL_DESTINATION"
echo "  ConfigPath: $CONFIG_PATH"
