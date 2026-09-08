package graphqlapi

import (
	"github.com/graphql-go/graphql"

	"endonend/api/internal/store"
)

func intArg(p graphql.ResolveParams, name string, fallback int) int {
	v, ok := p.Args[name]
	if !ok || v == nil {
		return fallback
	}
	return v.(int)
}

// albumsFromSummaries loads the full Album detail for each summary, since
// this API exposes one Album type everywhere rather than a separate
// lighter-weight summary type for the browse view.
func albumsFromSummaries(p graphql.ResolveParams, s *store.Store, summaries []store.AlbumSummary) ([]*store.Album, error) {
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
}

var queryType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Query",
	Fields: graphql.Fields{
		"artists": &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(identityType))),
			Args: graphql.FieldConfigArgument{
				"limit":  &graphql.ArgumentConfig{Type: graphql.Int},
				"offset": &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return storeFrom(p.Context).ListArtists(p.Context, intArg(p, "limit", 50), intArg(p, "offset", 0))
			},
		},
		"labels": &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(identityType))),
			Args: graphql.FieldConfigArgument{
				"limit":  &graphql.ArgumentConfig{Type: graphql.Int},
				"offset": &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return storeFrom(p.Context).ListLabels(p.Context, intArg(p, "limit", 50), intArg(p, "offset", 0))
			},
		},
		"identity": &graphql.Field{
			Type: identityType,
			Args: graphql.FieldConfigArgument{
				"url": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return storeFrom(p.Context).GetIdentity(p.Context, p.Args["url"].(string))
			},
		},
		"albums": &graphql.Field{
			Type: graphql.NewNonNull(graphql.NewList(graphql.NewNonNull(albumType))),
			Args: graphql.FieldConfigArgument{
				"limit":  &graphql.ArgumentConfig{Type: graphql.Int},
				"offset": &graphql.ArgumentConfig{Type: graphql.Int},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				s := storeFrom(p.Context)
				summaries, err := s.ListAlbums(p.Context, intArg(p, "limit", 50), intArg(p, "offset", 0))
				if err != nil {
					return nil, err
				}
				return albumsFromSummaries(p, s, summaries)
			},
		},
		"album": &graphql.Field{
			Type: albumType,
			Args: graphql.FieldConfigArgument{
				"identityUrl": &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
				"albumId":     &graphql.ArgumentConfig{Type: graphql.NewNonNull(graphql.String)},
			},
			Resolve: func(p graphql.ResolveParams) (any, error) {
				return storeFrom(p.Context).GetAlbum(p.Context, p.Args["identityUrl"].(string), p.Args["albumId"].(string))
			},
		},
	},
})
