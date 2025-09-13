package resolvers

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require here.

import (
	"context"
	"main/ent"
	"main/graph/directives"
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

// getClient возвращает клиент из контекста (для правильной работы с query/mutation клиентами)
// или fallback на r.client для обратной совместимости и тестов
func (r *Resolver) getClient(ctx context.Context) *ent.Client {
	if client := ent.FromContext(ctx); client != nil {
		return client
	}
	return r.client
}

// NewSchema creates a graphql executable schema
func NewSchema(client *ent.Client) graphql.ExecutableSchema {
	return generated.NewExecutableSchema(generated.Config{
		Resolvers: &Resolver{
			client: client,
		},
		Directives: generated.DirectiveRoot{
			Auth:   directives.Auth,
			Admin:  directives.Admin,
			Member: directives.Member,
		},
	})
}

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

// Mutation returns app.MutationResolver implementation
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Subscription returns generated.SubscriptionResolver implementation
//func (r *Resolver) Subscription() generated.SubscriptionResolver { return &subscriptionResolver{r} }

// Entity returns generated.EntityResolver implementation for federation
func (r *Resolver) Entity() generated.EntityResolver { return &entityResolver{r} }

// File returns generated.FileResolver implementation
func (r *Resolver) File() generated.FileResolver {
	return &fileResolver{r}
}

type queryResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type subscriptionResolver struct{ *Resolver }
type entityResolver struct{ *Resolver }
type fileResolver struct{ *Resolver }
