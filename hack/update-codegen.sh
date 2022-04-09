#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

bash "${CODEGEN_PKG}"/generate-groups.sh "deepcopy,client,informer,lister" \
  kunstack.com/pharos/pkg/client kunstack.com/pharos/pkg/apis \
  publish:v1beta1 \
  --output-base "./" \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt

rm -fr ./pkg/client/{clientset,informers,listers}
cp -r ./kunstack.com/pharos/pkg/ ./pkg/
rm -fr ./kunstack.com
