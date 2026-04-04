// Package api provides auto-generated ClickUp API clients.
//
// Generated code lives in clickupv2/ and clickupv3/ subdirectories.
// Specs are fetched on demand from developer.clickup.com — run `make api-gen`.
//
// Uses oapi-codegen-exp (github.com/oapi-codegen/oapi-codegen-exp) which
// natively supports OpenAPI 3.1 via libopenapi.
package api

//go:generate oapi-codegen -package clickupv2 -output clickupv2/client.gen.go specs/clickup-v2.json
//go:generate oapi-codegen -package clickupv3 -output clickupv3/client.gen.go specs/clickup-v3.yaml
