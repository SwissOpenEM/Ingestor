#!/bin/bash

set -euo pipefail

cd "$(dirname "$0")/.."
[ -d internal ] || { echo "This script should be run from the project root"; exit 1; }

echo "Updating extglobusservice"
curl -o internal/extglobusservice/openapi.yaml https://raw.githubusercontent.com/SwissOpenEM/scicat-globus-proxy/refs/heads/main/internal/api/openapi.yaml

echo "Updating s3upload"
curl -o internal/s3upload/openapi.yaml https://raw.githubusercontent.com/SwissOpenEM/ScopeMArchiver/c6284916d9b180b3d33dbfde2f1ae1072d135488/backend/api/openapi.yaml
# Patch OpenAPI spec for oapi-gen compatibility
sed -i '' 's/openapi: 3\.1\.[0-9]/openapi: 3.0.3/' internal/s3upload/openapi.yaml
