package item

import (
	"context"

	"github.com/dukfaar/goUtils/permission"
	graphql "github.com/graph-gophers/graphql-go"
)

type Resolver struct {
	Model *Model
}

func (r *Resolver) ID(ctx context.Context) (*graphql.ID, error) {
	err := permission.Check(ctx, "item._id.read")
	if err != nil {
		return nil, err
	}

	id := graphql.ID(r.Model.ID.Hex())
	return &id, nil
}

func (r *Resolver) Name(ctx context.Context) (*string, error) {
	err := permission.Check(ctx, "item.name.read")
	if err != nil {
		return nil, err
	}

	return &r.Model.Name, nil
}

func (r *Resolver) NamespaceID(ctx context.Context) (*graphql.ID, error) {
	err := permission.Check(ctx, "item.namespaceId.read")
	if err != nil {
		return nil, err
	}

	id := graphql.ID(r.Model.NamespaceID.Hex())
	return &id, nil
}
