package item

import graphql "github.com/graph-gophers/graphql-go"

type EdgeResolver struct {
	model *Model
}

func (r *EdgeResolver) Node() *Resolver {
	return &Resolver{Model: r.model}
}

func (r *EdgeResolver) Cursor() graphql.ID {
	return graphql.ID(r.model.ID.Hex())
}
