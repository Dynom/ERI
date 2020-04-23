package main

import (
	"errors"

	"github.com/Dynom/ERI/cmd/web/config"
	"github.com/Dynom/ERI/cmd/web/services"

	"github.com/Dynom/ERI/cmd/web/erihttp"
	"github.com/Dynom/ERI/validator"
	"github.com/graphql-go/graphql"
)

func NewGraphQLSchema(conf config.Config, suggestSvc *services.SuggestSvc, autocompleteSvc *services.AutocompleteSvc) (graphql.Schema, error) {

	suggestionType := graphql.NewObject(graphql.ObjectConfig{
		Name: "suggestion",
		Fields: graphql.Fields{
			"alternatives": &graphql.Field{
				Description: "The list of alternatives. If no better match is found, the input is returned. 1 or more.",
				Type:        graphql.NewList(graphql.NewNonNull(graphql.String)),
			},

			"malformedSyntax": &graphql.Field{
				Description: "Boolean value that when true, means the address can't be valid. Conversely when false, doesn't mean it is.",
				Type:        graphql.NewNonNull(graphql.Boolean),
			},
		},
		Description: "",
	})

	autocompleteType := graphql.NewObject(graphql.ObjectConfig{
		Name: "autocomplete",
		Fields: graphql.Fields{
			"suggestions": &graphql.Field{
				Description: "The list of domains matching the prefix. 0 or more are returned",
				Type:        graphql.NewList(graphql.NewNonNull(graphql.String)),
			},
		},
		Description: "",
	})

	fields := graphql.Fields{
		"suggestion": &graphql.Field{
			Type: suggestionType,
			Args: graphql.FieldConfigArgument{
				"email": &graphql.ArgumentConfig{
					Type:        graphql.NewNonNull(graphql.String),
					Description: "The e-mail address you'd like to get suggestions for",
				},
			},
			Resolve: func(p graphql.ResolveParams) (i interface{}, err error) {
				i = erihttp.SuggestResponse{
					Alternatives:    []string{},
					MalformedSyntax: false,
					Error:           "",
				}

				value, ok := p.Args["email"]
				if !ok {
					return i, errors.New("missing required parameters")
				}

				email := value.(string)
				result, sugErr := suggestSvc.Suggest(p.Context, email)
				if sugErr != nil && sugErr != validator.ErrEmailAddressSyntax {
					err = sugErr
				}

				return erihttp.SuggestResponse{
					Alternatives:    result.Alternatives,
					MalformedSyntax: sugErr == validator.ErrEmailAddressSyntax,
				}, err
			},
			Description: "Get suggestions",
		},
		"autocomplete": &graphql.Field{
			Type: autocompleteType,
			Args: graphql.FieldConfigArgument{
				"domain": &graphql.ArgumentConfig{
					Type:        graphql.NewNonNull(graphql.String),
					Description: "",
				},
			},
			Resolve: func(p graphql.ResolveParams) (i interface{}, err error) {
				i = erihttp.AutoCompleteResponse{
					Suggestions: []string{},
					Error:       "",
				}

				value, ok := p.Args["domain"]
				if !ok {
					return i, errors.New("missing required parameters")
				}

				domain := value.(string)
				result, err := autocompleteSvc.Autocomplete(p.Context, domain, conf.Server.Services.Autocomplete.MaxSuggestions)
				if err != nil {
					return i, err
				}

				return erihttp.AutoCompleteResponse{
					Suggestions: result.Suggestions,
				}, err
			},
		},
	}

	return graphql.NewSchema(graphql.SchemaConfig{
		Query: graphql.NewObject(graphql.ObjectConfig{
			Name:   "RootQuery",
			Fields: fields,
		}),
	})
}
