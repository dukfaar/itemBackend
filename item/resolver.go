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
	err := permission.Check(ctx, "Item._id.read")
	if err != nil {
		return nil, err
	}

	id := graphql.ID(r.Model.ID.Hex())
	return &id, nil
}

func (r *Resolver) Name(ctx context.Context) (*string, error) {
	err := permission.Check(ctx, "Item.name.read")
	if err != nil {
		return nil, err
	}

	return &r.Model.Name, nil
}

func (r *Resolver) XivdbID(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "Item.xivdbid.read")
	if err != nil {
		return nil, err
	}

	return r.Model.XivdbID, nil
}

func (r *Resolver) NamespaceID(ctx context.Context) (*graphql.ID, error) {
	err := permission.Check(ctx, "Item.namespaceId.read")
	if err != nil {
		return nil, err
	}

	id := graphql.ID(r.Model.NamespaceID.Hex())
	return &id, nil
}

func (r *Resolver) GatheringLevel(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "Item.gatheringLevel.read")
	if err != nil {
		return nil, err
	}

	return r.Model.GatheringLevel, nil
}

func (r *Resolver) GatheringJob(ctx context.Context) (*graphql.ID, error) {
	err := permission.Check(ctx, "Item.gatheringJob.read")
	if err != nil {
		return nil, err
	}

	id := graphql.ID(r.Model.GatheringJob.Hex())
	return &id, nil
}

func (r *Resolver) GatheringEffort(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "Item.gatheringEffort.read")
	if err != nil {
		return nil, err
	}

	return r.Model.GatheringEffort, nil
}

func (r *Resolver) Price(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "Item.price.read")
	if err != nil {
		return nil, err
	}

	return r.Model.Price, nil
}

func (r *Resolver) PriceHQ(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "Item.priceHq.read")
	if err != nil {
		return nil, err
	}

	return r.Model.PriceHQ, nil
}

func (r *Resolver) UnspoiledNode(ctx context.Context) (*bool, error) {
	err := permission.Check(ctx, "Item.unspoiledNode.read")
	if err != nil {
		return nil, err
	}

	return r.Model.UnspoiledNode, nil
}

func (r *Resolver) AvailableFromNpc(ctx context.Context) (*bool, error) {
	err := permission.Check(ctx, "Item.availableFromNpc.read")
	if err != nil {
		return nil, err
	}

	return r.Model.AvailableFromNpc, nil
}

type UnspoiledNodeTimeResolver struct {
	time *UnspoiledNodeTime
}

func (r *Resolver) UnspoiledNodeTime(ctx context.Context) (*UnspoiledNodeTimeResolver, error) {
	err := permission.Check(ctx, "Item.unspoiledNodeTime.read")
	if err != nil {
		return nil, err
	}

	if r.Model.UnspoiledNodeTime == nil {
		return nil, nil
	}

	return &UnspoiledNodeTimeResolver{r.Model.UnspoiledNodeTime}, nil
}

func (r *UnspoiledNodeTimeResolver) Time(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "UnspoiledNodeTime.time.read")
	if err != nil {
		return nil, err
	}

	return r.time.Time, nil
}

func (r *UnspoiledNodeTimeResolver) Duration(ctx context.Context) (*int32, error) {
	err := permission.Check(ctx, "UnspoiledNodeTime.duration.read")
	if err != nil {
		return nil, err
	}

	return r.time.Duration, nil
}

func (r *UnspoiledNodeTimeResolver) AmPm(ctx context.Context) (*string, error) {
	err := permission.Check(ctx, "UnspoiledNodeTime.ampm.read")
	if err != nil {
		return nil, err
	}

	return r.time.AmPm, nil
}

func (r *UnspoiledNodeTimeResolver) FolkloreNeeded(ctx context.Context) (*string, error) {
	err := permission.Check(ctx, "UnspoiledNodeTime.folkloreNeeded.read")
	if err != nil {
		return nil, err
	}

	return r.time.FolkloreNeeded, nil
}
