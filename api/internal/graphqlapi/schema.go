package graphqlapi

import "github.com/graphql-go/graphql"

// PublicSchema serves /v1/graphql: read-only, safe for any listener.
// AdminSchema serves /v1/admin/graphql: read plus the onboarding mutations,
// gated separately per KB/0010-mvp-scope.md and never linked from a public
// page. Both are built at the end of types.go's init(), once identityType
// and albumType have their fields populated: building them here instead,
// as ordinary package-level vars, would race Go's initialization order
// against that init() and construct them from still-empty types.
var (
	PublicSchema graphql.Schema
	AdminSchema  graphql.Schema
)

func mustSchema(cfg graphql.SchemaConfig) graphql.Schema {
	s, err := graphql.NewSchema(cfg)
	if err != nil {
		panic("graphqlapi: invalid schema: " + err.Error())
	}
	return s
}
