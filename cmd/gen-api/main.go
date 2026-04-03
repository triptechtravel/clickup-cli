// Command gen-api reads an OpenAPI spec and generates typed Go wrapper
// functions for each operation. The wrappers use the auto-generated types
// from clickupv2/clickupv3 packages and route through api.Client.
//
// Usage:
//
//	go run ./cmd/gen-api -spec api/specs/clickup-v2.json -pkg apiv2 -types-pkg clickupv2 -out internal/apiv2/operations.gen.go
//	go run ./cmd/gen-api -spec api/specs/clickup-v3.yaml -pkg apiv3 -types-pkg clickupv3 -out internal/apiv3/operations.gen.go
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

type spec struct {
	Paths map[string]map[string]json.RawMessage `json:"paths"`
}

type operation struct {
	OperationID string     `json:"operationId"`
	Summary     string     `json:"summary"`
	Parameters  []param    `json:"parameters"`
	RequestBody *struct{}  `json:"requestBody"`
	Responses   map[string]struct {
		Content map[string]struct{} `json:"content"`
	} `json:"responses"`
}

type param struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
	Schema   struct {
		Type string `json:"type"`
	} `json:"schema"`
}

type opInfo struct {
	FuncName    string
	Method      string
	PathPattern string // e.g., "task/%s/comment"
	PathArgs    []pathArg
	QueryParams []queryParam
	HasReqBody  bool
	HasRespBody bool
	ReqType     string // e.g., "clickupv2.UpdateTaskJSONRequest"
	RespType    string // e.g., "clickupv2.GetTaskJSONResponse"
	Summary     string
}

type pathArg struct {
	Name     string // Go param name
	SpecName string // original spec name
}

type queryParam struct {
	Name     string
	SpecName string
	Type     string
	Required bool
}

var (
	specPath = flag.String("spec", "", "path to OpenAPI spec (JSON)")
	pkg      = flag.String("pkg", "apiv2", "output Go package name")
	typesPkg = flag.String("types-pkg", "clickupv2", "types package name")
	typesImp = flag.String("types-import", "", "types package import path (auto-detected if empty)")
	outPath  = flag.String("out", "", "output file path")
	module   = flag.String("module", "github.com/triptechtravel/clickup-cli", "Go module path")
	fixMode  = flag.Bool("fix", false, "fix self-referencing types in generated code instead of generating wrappers")
	fixGen   = flag.String("fix-gen", "", "path to generated types file (for -fix mode)")
	fixOut   = flag.String("fix-out", "", "path to write fixes file (for -fix mode)")
	fixPkg   = flag.String("fix-pkg", "", "package name for fixes file (for -fix mode)")
)

var pathParamRe = regexp.MustCompile(`\{([^}]+)\}`)

func main() {
	flag.Parse()

	if *fixMode {
		if *specPath == "" || *fixGen == "" || *fixOut == "" || *fixPkg == "" {
			log.Fatal("-fix mode requires -spec, -fix-gen, -fix-out, and -fix-pkg")
		}
		if err := fixSelfRefs(*specPath, *fixGen, *fixOut, *fixPkg); err != nil {
			log.Fatalf("fix: %v", err)
		}
		return
	}

	if *specPath == "" || *outPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	data, err := os.ReadFile(*specPath)
	if err != nil {
		log.Fatalf("read spec: %v", err)
	}

	var s spec
	if err := json.Unmarshal(data, &s); err != nil {
		log.Fatalf("parse spec: %v", err)
	}

	if *typesImp == "" {
		*typesImp = *module + "/api/" + *typesPkg
	}

	// Read generated types file to know which types actually exist.
	typesFile := "api/" + *typesPkg + "/client.gen.go"
	existingTypes := readExistingTypes(typesFile)

	ops := extractOperations(s, *typesPkg, existingTypes)
	sort.Slice(ops, func(i, j int) bool { return ops[i].FuncName < ops[j].FuncName })

	if err := writeFile(*outPath, ops); err != nil {
		log.Fatalf("write: %v", err)
	}

	fmt.Printf("Generated %d operations → %s\n", len(ops), *outPath)
}

func readExistingTypes(path string) map[string]bool {
	types := make(map[string]bool)
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("warning: cannot read types file %s: %v", path, err)
		return types
	}
	// Match "type FooBar struct" or "type FooBar = ..."
	re := regexp.MustCompile(`(?m)^type (\w+) `)
	for _, m := range re.FindAllStringSubmatch(string(data), -1) {
		types[m[1]] = true
	}
	return types
}

func extractOperations(s spec, typesPkg string, existingTypes map[string]bool) []opInfo {
	var ops []opInfo

	for path, methods := range s.Paths {
		for method, raw := range methods {
			if method == "parameters" {
				continue
			}
			var op operation
			if err := json.Unmarshal(raw, &op); err != nil {
				continue // skip non-operation entries
			}
			if op.OperationID == "" {
				continue
			}

			info := opInfo{
				FuncName:   cleanFuncName(op.OperationID),
				Method:     strings.ToUpper(method),
				Summary:    op.Summary,
				HasReqBody: op.RequestBody != nil,
			}

			// Check if 200 response has JSON content.
			if resp, ok := op.Responses["200"]; ok {
				if _, ok := resp.Content["application/json"]; ok {
					info.HasRespBody = true
				}
			}

			// Build path pattern: /v2/task/{task_id} → "task/%s"
			trimmed := strings.TrimPrefix(path, "/v2/")
			var pathArgs []pathArg
			pattern := pathParamRe.ReplaceAllStringFunc(trimmed, func(match string) string {
				name := match[1 : len(match)-1]
				pathArgs = append(pathArgs, pathArg{
					Name:     toCamelParam(name),
					SpecName: name,
				})
				return "%s"
			})
			info.PathPattern = pattern
			info.PathArgs = pathArgs

			// Extract query params.
			for _, p := range op.Parameters {
				if p.In == "query" {
					info.QueryParams = append(info.QueryParams, queryParam{
						Name:     toCamelParam(p.Name),
						SpecName: p.Name,
						Type:     goType(p.Schema.Type),
						Required: p.Required,
					})
				}
			}

			// Map to auto-gen type names. Look up which name actually exists
			// in the generated types file (codegen may clean names differently).
			cleanedOpID := cleanFuncName(op.OperationID)
			skip := false

			if info.HasReqBody {
				reqName := cleanedOpID + "JSONRequest"
				if existingTypes[reqName] {
					info.ReqType = typesPkg + "." + reqName
				} else {
					// Try original operationId (some have special chars removed differently)
					altName := op.OperationID + "JSONRequest"
					if existingTypes[altName] {
						info.ReqType = typesPkg + "." + altName
					} else {
						skip = true // can't find request type
					}
				}
			}
			if info.HasRespBody {
				respName := cleanedOpID + "JSONResponse"
				if existingTypes[respName] {
					info.RespType = typesPkg + "." + respName
				} else {
					altName := op.OperationID + "JSONResponse"
					if existingTypes[altName] {
						info.RespType = typesPkg + "." + altName
					} else {
						// No response type — generate with any as response
						info.HasRespBody = false
					}
				}
			}

			if skip {
				continue
			}

			ops = append(ops, info)
		}
	}

	return ops
}

func cleanFuncName(opID string) string {
	// Remove apostrophes, spaces, etc.
	cleaned := strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return -1
	}, opID)
	if cleaned == "" {
		return opID
	}
	// Ensure first letter is uppercase.
	return strings.ToUpper(cleaned[:1]) + cleaned[1:]
}

func toCamelParam(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	result := strings.Join(parts, "")
	// Lowercase first letter for param names.
	if len(result) > 0 {
		result = strings.ToLower(result[:1]) + result[1:]
	}
	return result
}

func goType(schemaType string) string {
	switch schemaType {
	case "integer":
		return "int"
	case "number":
		return "float64"
	case "boolean":
		return "bool"
	default:
		return "string"
	}
}

func writeFile(path string, ops []opInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, map[string]any{
		"Pkg":        *pkg,
		"TypesPkg":   *typesPkg,
		"TypesImport": *typesImp,
		"Ops":        ops,
	})
}

var tmpl = template.Must(template.New("").Funcs(template.FuncMap{
	"upper": strings.ToUpper,
}).Parse(`// Code generated by gen-api from OpenAPI spec; DO NOT EDIT.

package {{ .Pkg }}

import (
	"context"
	"fmt"

	"github.com/triptechtravel/clickup-cli/internal/api"
	{{ .TypesPkg }} "{{ .TypesImport }}"
)

// Ensure imports are used.
var _ = fmt.Sprintf
var _ {{ .TypesPkg }}.Nullable[string]
{{ range .Ops }}
// {{ .FuncName }} — {{ .Summary }}
// {{ .Method }} /v2/{{ .PathPattern }}
func {{ .FuncName }}(ctx context.Context, client *api.Client{{ range .PathArgs }}, {{ .Name }} string{{ end }}{{ if .HasReqBody }}, req *{{ .ReqType }}{{ end }}) {{ if .HasRespBody }}(*{{ .RespType }}, error){{ else }}error{{ end }} {
	{{- if .HasRespBody }}
	var resp {{ .RespType }}
	{{- end }}
	path := {{ if .PathArgs }}fmt.Sprintf("{{ .PathPattern }}"{{ range .PathArgs }}, {{ .Name }}{{ end }}){{ else }}"{{ .PathPattern }}"{{ end }}
	{{- if .HasRespBody }}
	if err := do(ctx, client, "{{ .Method }}", path, {{ if .HasReqBody }}req{{ else }}nil{{ end }}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
	{{- else }}
	return do(ctx, client, "{{ .Method }}", path, {{ if .HasReqBody }}req{{ else }}nil{{ end }}, nil)
	{{- end }}
}
{{ end }}
`))
