package graphqlapi

import (
	"context"

	"github.com/graphql-go/graphql"

	"endonend/api/internal/crawler"
	"endonend/protocol/validate"
)

const crawlerCtxKey ctxKey = storeCtxKey + 1

func withCrawler(ctx context.Context, c *crawler.Crawler) context.Context {
	return context.WithValue(ctx, crawlerCtxKey, c)
}

func crawlerFrom(ctx context.Context) *crawler.Crawler {
	c, _ := ctx.Value(crawlerCtxKey).(*crawler.Crawler)
	return c
}

var validationErrorType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ValidationError",
	Fields: graphql.Fields{
		"field":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"message": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var validationReportType = graphql.NewObject(graphql.ObjectConfig{
	Name: "ValidationReport",
	Fields: graphql.Fields{
		"valid":  &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errors": &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(validationErrorType)))},
	},
})

var submitResultType = graphql.NewObject(graphql.ObjectConfig{
	Name: "SubmitResult",
	Fields: graphql.Fields{
		"accepted": &graphql.Field{Type: graphql.NewNonNull(graphql.Boolean)},
		"errors":   &graphql.Field{Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String)))},
		"identity": &graphql.Field{Type: identityType},
	},
})

type submitResult struct {
	Accepted bool
	Errors   []string
	Identity any
}

type validationReport struct {
	Valid  bool
	Errors []validationError
}

type validationError struct {
	Field   string
	Message string
}

var mutationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"submitManifestUrl": &graphql.Field{
			Type: graphql.NewNonNull(submitResultType),
			Args: graphql.FieldConfigArgument{
				"url": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				identityURL := p.Args["url"].(string)
				manifestURL := identityURL + "/.well-known/endonend/manifest.json"
				c := crawlerFrom(p.Context)
				if err := c.PollOne(p.Context, manifestURL); err != nil {
					return submitResult{Accepted: false, Errors: []string{err.Error()}}, nil
				}
				id, err := storeFrom(p.Context).GetIdentity(p.Context, identityURL)
				if err != nil {
					return nil, err
				}
				return submitResult{Accepted: true, Identity: id}, nil
			},
		},
		"validateManifest": &graphql.Field{
			Type: graphql.NewNonNull(validationReportType),
			Args: graphql.FieldConfigArgument{
				"rawJson": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				raw := p.Args["rawJson"].(string)
				report, err := validate.ValidateRawJSON([]byte(raw), validate.Options{})
				if err != nil {
					return validationReport{Valid: false, Errors: []validationError{{Field: "$", Message: err.Error()}}}, nil
				}
				out := validationReport{Valid: report.Valid}
				for _, f := range report.Failures {
					out.Errors = append(out.Errors, validationError{Field: f.Field, Message: f.Message})
				}
				return out, nil
			},
		},
	},
})
