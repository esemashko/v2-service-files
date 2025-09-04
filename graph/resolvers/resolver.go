package resolvers

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"main/ent"
	"main/graph/generated"

	"github.com/99designs/gqlgen/graphql"
)

// Resolver is the resolver root
type Resolver struct {
	client *ent.Client
}

// SetClient устанавливает клиент для тестов
func (r *Resolver) SetClient(client *ent.Client) {
	r.client = client
}

// NewSchema creates a graphql executable schema
func NewSchema(client *ent.Client) graphql.ExecutableSchema {
	return generated.NewExecutableSchema(generated.Config{
		Resolvers: &Resolver{
			client: client,
		},
		Directives: generated.DirectiveRoot{},
	})
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Mutation returns app.MutationResolver implementation
//func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation
//func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

// Entity returns generated.EntityResolver implementation for federation
//func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

type queryResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
type entityResolver struct{ *Resolver }
