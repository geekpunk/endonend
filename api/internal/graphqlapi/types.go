// Package graphqlapi implements the versioned GraphQL API defined in
// KB/0010-mvp-scope.md's "API (GraphQL)" section: a public, read-only
// Query root served at /v1/graphql, and an admin-only Mutation root served
// separately at /v1/admin/graphql, never linked from any public page.
package graphqlapi

import (
	"context"
	"encoding/json"

	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"

	"endonend/api/internal/store"
	"endonend/protocol/manifest"
)

type ctxKey int

const storeCtxKey ctxKey = iota

func withStore(ctx context.Context, s *store.Store) context.Context {
	return context.WithValue(ctx, storeCtxKey, s)
}

func storeFrom(ctx context.Context) *store.Store {
	s, _ := ctx.Value(storeCtxKey).(*store.Store)
	return s
}

// jsonScalar passes an already-decoded Go value (map, slice, string, ...)
// through to the response as JSON, for manifest fields (credits, presentation
// links) whose shape is defined by KB/0003-manifest.md rather than by this
// schema. Resolvers are responsible for decoding raw jsonb bytes into a
// plain Go value before returning it; this scalar does no decoding itself.
var jsonScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "An arbitrary JSON value, shaped per KB/0003-manifest.md rather than this schema.",
	Serialize:   func(value any) any { return value },
	ParseValue:  func(value any) any { return value },
	ParseLiteral: func(valueAST ast.Value) any {
		return nil // only ever used as output in this API; input literals aren't needed.
	},
})

func decodeJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil
	}
	return out
}

var verificationStatusEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "VerificationStatus",
	Values: graphql.EnumValueConfigMap{
		"VERIFIED":   &graphql.EnumValueConfig{Value: store.Verified},
		"UNVERIFIED": &graphql.EnumValueConfig{Value: store.Unverified},
		"DISPUTED":   &graphql.EnumValueConfig{Value: store.Disputed},
		"PENDING":    &graphql.EnumValueConfig{Value: store.Pending},
	},
})

var identityTypeEnum = graphql.NewEnum(graphql.EnumConfig{
	Name: "IdentityType",
	Values: graphql.EnumValueConfigMap{
		"ARTIST": &graphql.EnumValueConfig{Value: "artist"},
		"LABEL":  &graphql.EnumValueConfig{Value: "label"},
	},
})

var splitType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Split",
	Fields: graphql.Fields{
		"artist": &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
		"label":  &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
	},
})

type splitValue struct{ Artist, Label float64 }

var merchLinkType = graphql.NewObject(graphql.ObjectConfig{
	Name: "MerchLink",
	Fields: graphql.Fields{
		"label": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"url":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var purchaseLinkType = graphql.NewObject(graphql.ObjectConfig{
	Name: "PurchaseLink",
	Fields: graphql.Fields{
		"format": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"url":    &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
	},
})

var trackType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Track",
	Fields: graphql.Fields{
		"trackId":  &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"side":     &graphql.Field{Type: graphql.String},
		"number":   &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"name":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"duration": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"file":     &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"lyrics":   &graphql.Field{Type: graphql.String},
	},
})

var colorsType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Colors",
	Fields: graphql.Fields{
		"primary":        &graphql.Field{Type: graphql.String},
		"heroBackground": &graphql.Field{Type: graphql.String},
		"background":     &graphql.Field{Type: graphql.String},
		"text":           &graphql.Field{Type: graphql.String},
	},
})

var presentationType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Presentation",
	Fields: graphql.Fields{
		"colors": &graphql.Field{Type: colorsType},
		"links": &graphql.Field{
			Type: jsonScalar,
			Resolve: func(p graphql.ResolveParams) (any, error) {
				pres := p.Source.(*manifest.Presentation)
				if pres.Links == nil {
					return nil, nil
				}
				return pres.Links, nil
			},
		},
		"footer": &graphql.Field{Type: graphql.String},
	},
})

var rosterEntryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "RosterEntry",
	Fields: graphql.Fields{
		"artist": &graphql.Field{
			Type: graphql.NewNonNull(identityType),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				r := p.Source.(store.RosterEntry)
				return storeFrom(p.Context).GetIdentity(p.Context, r.ArtistURL)
			},
		},
		"split": &graphql.Field{
			Type: graphql.NewNonNull(splitType),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				r := p.Source.(store.RosterEntry)
				return splitValue{Artist: r.SplitArtistPct, Label: r.SplitLabelPct}, nil
			},
		},
		"verified": &graphql.Field{
			Type: graphql.NewNonNull(verificationStatusEnum),
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return p.Source.(store.RosterEntry).Verified, nil
			},
		},
	},
})

var albumSplitType = graphql.NewObject(graphql.ObjectConfig{
	Name: "AlbumSplit",
	Fields: graphql.Fields{
		"manifestUrl": &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"role":        &graphql.Field{Type: graphql.NewNonNull(graphql.String)},
		"percentage":  &graphql.Field{Type: graphql.NewNonNull(graphql.Float)},
		"verified":    &graphql.Field{Type: graphql.NewNonNull(verificationStatusEnum)},
	},
})

// identityType and albumType reference each other (Identity.albums,
// Album.identity), so both are declared empty first and their fields
// filled in via AddFieldConfig once both exist.
var identityType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "Identity",
	Fields: graphql.Fields{},
})

var albumType = graphql.NewObject(graphql.ObjectConfig{
	Name:   "Album",
	Fields: graphql.Fields{},
})

func init() {
	identityType.AddFieldConfig("url", &graphql.Field{Type: graphql.NewNonNull(graphql.String)})
	identityType.AddFieldConfig("type", &graphql.Field{
		Type: graphql.NewNonNull(identityTypeEnum),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Identity).Type, nil
		},
	})
	identityType.AddFieldConfig("name", &graphql.Field{Type: graphql.NewNonNull(graphql.String)})
	identityType.AddFieldConfig("contactEmail", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Identity).ContactEmail, nil
		},
	})
	identityType.AddFieldConfig("affiliatedLabel", &graphql.Field{
		Type: identityType,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			id := p.Source.(*store.Identity)
			if id.AffiliatedLabelURL == "" {
				return nil, nil
			}
			return storeFrom(p.Context).GetIdentity(p.Context, id.AffiliatedLabelURL)
		},
	})
	identityType.AddFieldConfig("labelSplit", &graphql.Field{
		Type: splitType,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			id := p.Source.(*store.Identity)
			if id.SplitArtistPct == nil || id.SplitLabelPct == nil {
				return nil, nil
			}
			return splitValue{Artist: *id.SplitArtistPct, Label: *id.SplitLabelPct}, nil
		},
	})
	identityType.AddFieldConfig("roster", &graphql.Field{
		Type: graphql.NewList(graphql.NewNonNull(rosterEntryType)),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			id := p.Source.(*store.Identity)
			if id.Type != "label" {
				return nil, nil
			}
			return storeFrom(p.Context).RosterForLabel(p.Context, id.URL)
		},
	})
	identityType.AddFieldConfig("merch", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(merchLinkType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			id := p.Source.(*store.Identity)
			return storeFrom(p.Context).MerchForIdentity(p.Context, id.URL)
		},
	})
	identityType.AddFieldConfig("albums", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(albumType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			id := p.Source.(*store.Identity)
			if id.Type != "artist" {
				return []*store.Album{}, nil
			}
			s := storeFrom(p.Context)
			summaries, err := s.AlbumsForIdentity(p.Context, id.URL)
			if err != nil {
				return nil, err
			}
			albums := make([]*store.Album, 0, len(summaries))
			for _, sum := range summaries {
				a, err := s.GetAlbum(p.Context, sum.IdentityURL, sum.AlbumID)
				if err != nil {
					return nil, err
				}
				if a != nil {
					albums = append(albums, a)
				}
			}
			return albums, nil
		},
	})

	albumType.AddFieldConfig("identity", &graphql.Field{
		Type: graphql.NewNonNull(identityType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			a := p.Source.(*store.Album)
			return storeFrom(p.Context).GetIdentity(p.Context, a.IdentityURL)
		},
	})
	albumType.AddFieldConfig("albumId", &graphql.Field{Type: graphql.NewNonNull(graphql.String)})
	albumType.AddFieldConfig("albumVersion", &graphql.Field{Type: graphql.NewNonNull(graphql.Int)})
	albumType.AddFieldConfig("albumName", &graphql.Field{Type: graphql.NewNonNull(graphql.String)})
	albumType.AddFieldConfig("pageTitle", &graphql.Field{Type: graphql.String})
	albumType.AddFieldConfig("releaseDate", &graphql.Field{Type: graphql.NewNonNull(graphql.String)})
	albumType.AddFieldConfig("imagesFront", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Album).ImagesFront, nil
		},
	})
	albumType.AddFieldConfig("imagesBack", &graphql.Field{
		Type: graphql.NewNonNull(graphql.String),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Album).ImagesBack, nil
		},
	})
	albumType.AddFieldConfig("imagesInsert", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(graphql.String))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Album).ImagesInsert, nil
		},
	})
	albumType.AddFieldConfig("downloadZip", &graphql.Field{
		Type: graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			a := p.Source.(*store.Album)
			if a.DownloadZip == "" {
				return nil, nil
			}
			return a.DownloadZip, nil
		},
	})
	albumType.AddFieldConfig("credits", &graphql.Field{
		Type: jsonScalar,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return decodeJSON(p.Source.(*store.Album).CreditsJSON), nil
		},
	})
	albumType.AddFieldConfig("splits", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(albumSplitType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Album).Splits, nil
		},
	})
	albumType.AddFieldConfig("purchaseLinks", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(purchaseLinkType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Album).PurchaseLinks, nil
		},
	})
	albumType.AddFieldConfig("presentation", &graphql.Field{
		Type: presentationType,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			raw := p.Source.(*store.Album).PresentationJSON
			if len(raw) == 0 {
				return nil, nil
			}
			var pres manifest.Presentation
			if err := json.Unmarshal(raw, &pres); err != nil {
				return nil, nil
			}
			return &pres, nil
		},
	})
	albumType.AddFieldConfig("tracks", &graphql.Field{
		Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(trackType))),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return p.Source.(*store.Album).Tracks, nil
		},
	})

	// Built here, not as package-level vars, so identityType and albumType
	// are guaranteed fully populated first; see schema.go.
	PublicSchema = mustSchema(graphql.SchemaConfig{Query: queryType})
	AdminSchema = mustSchema(graphql.SchemaConfig{Query: queryType, Mutation: mutationType})
}
