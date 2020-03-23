package main

import (
	"errors"

	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/validator"
	"github.com/graphql-go/graphql"
)

func NewGraphQLSchema(suggestSvc *services.SuggestSvc) (graphql.Schema, error) {

	suggestionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "suggestion",
		Fields: graphql.Fields{
			"alternatives": &graphql.Field{
				Description: "The list of alternatives. If no better match is found, the input is returned. 1 or more.",
				Type:        graphql.NewList(graphql.String),
			},

			"malformedSyntax": &graphql.Field{
				Description: "Boolean value that when true, means the address can't be valid. Conversely when false, doesn't mean it is.",
				Type:        graphql.Boolean,
			},
		},
		Description: "",
	})

	fields := graphql.Fields{
		"suggestion": &graphql.Field{
			Type: suggestionType,
			Args: graphql.FieldConfigArgument{
				"email": &graphql.ArgumentConfig{
					Type:        graphql.String,
					Description: "The e-mail address you'd like to get suggestions for",
				},
			},
			Resolve: func(p graphql.ResolveParams) (i interface{}, err error) {
				if value, ok := p.Args["email"]; ok {
					var err error
					email := value.(string)
					result, sugErr := suggestSvc.Suggest(p.Context, email)
					if sugErr != nil && sugErr != validator.ErrEmailAddressSyntax {
						err = sugErr
					}

					return erihttp.SuggestResponse{
						Alternatives:    result.Alternatives,
						MalformedSyntax: sugErr == validator.ErrEmailAddressSyntax,
					}, err
				}

				return nil, errors.New("missing required parameters")
			},
			Description: "Get suggestions",
		},
	}
	return graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "RootQuery",
			Fields: fields,
		}),
	})
}
