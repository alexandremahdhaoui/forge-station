#!/bin/bash
# Copyright 2024 Alexandre Mahdhaoui
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script adds license headers to generated oapi-codegen files
# that don't support header injection during generation.

set -euo pipefail

LICENSE_HEADER='// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

'

# Use a lockfile to prevent parallel execution issues
LOCKFILE="/tmp/add-license-headers.lock"
exec 200>"$LOCKFILE"
flock -x 200

# Find all oapi-codegen generated files
GENERATED_FILES=$(find ./internal/driver/rest -name 'zz_generated.oapi-codegen.go' 2>/dev/null || true)

for file in $GENERATED_FILES; do
    # Check if file already has license header
    if ! head -1 "$file" | grep -q "^// Copyright"; then
        echo "Adding license header to: $file"
        # Create unique temp file with header + original content
        tmpfile=$(mktemp)
        echo "$LICENSE_HEADER" > "$tmpfile"
        cat "$file" >> "$tmpfile"
        mv "$tmpfile" "$file"
    fi
done

echo "License headers added to generated files"
