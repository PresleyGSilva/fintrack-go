package docs

import _ "embed"

// Spec é o conteúdo de docs/openapi.yaml servido em GET /openapi.yaml.
//
//go:embed openapi.yaml
var Spec []byte
