package main

import (
	"context"

	"github.com/dukfaar/goUtils/permission"

	"github.com/dukfaar/goUtils/relay"
	"github.com/dukfaar/itemBackend/item"
	graphql "github.com/graph-gophers/graphql-go"
	"gopkg.in/mgo.v2/bson"
)

type Resolver struct {
}

func (r *Resolver) Items(ctx context.Context, args struct {
	First  *int32
	Last   *int32
	Before *string
	After  *string
}) (*item.ConnectionResolver, error) {
	err := permission.Check(ctx, "query.items")
	if err != nil {
		return nil, err
	}

	itemService := ctx.Value("itemService").(item.Service)

	var totalChannel = make(chan int)
	go func() {
		var total, _ = itemService.Count()
		totalChannel <- total
	}()

	var itemsChannel = make(chan []item.Model)
	go func() {
		result, _ := itemService.List(args.First, args.Last, args.Before, args.After)
		itemsChannel <- result
	}()

	var (
		start string
		end   string
	)

	var items = <-itemsChannel

	if len(items) == 0 {
		start, end = "", ""
	} else {
		start, end = items[0].ID.Hex(), items[len(items)-1].ID.Hex()
	}

	hasPreviousPageChannel, hasNextPageChannel := relay.GetHasPreviousAndNextPage(len(items), start, end, itemService)

	return &item.ConnectionResolver{
		Models: items,
		ConnectionResolver: relay.ConnectionResolver{
			relay.Connection{
				Total:           int32(<-totalChannel),
				From:            start,
				To:              end,
				HasNextPage:     <-hasNextPageChannel,
				HasPreviousPage: <-hasPreviousPageChannel,
			},
		},
	}, nil
}

func (r *Resolver) CreateItem(ctx context.Context, args struct {
	Name        *string
	NamespaceId *string
}) (*item.Resolver, error) {
	err := permission.Check(ctx, "query.createItem")
	if err != nil {
		return nil, err
	}

	itemService := ctx.Value("itemService").(item.Service)

	newModel, err := itemService.Create(&item.Model{
		Name:        *args.Name,
		NamespaceID: bson.ObjectIdHex(*args.NamespaceId),
	})

	if err == nil {
		return &item.Resolver{
			Model: newModel,
		}, nil
	}

	return nil, err
}

func (r *Resolver) UpdateItem(ctx context.Context, args struct {
	Id          string
	Name        *string
	NamespaceId *string
}) (*item.Resolver, error) {
	err := permission.Check(ctx, "query.updateItem")
	if err != nil {
		return nil, err
	}

	itemService := ctx.Value("itemService").(item.Service)

	newModel, err := itemService.Update(args.Id, &item.Model{
		Name:        *args.Name,
		NamespaceID: bson.ObjectIdHex(*args.NamespaceId),
	})

	if err == nil {
		return &item.Resolver{
			Model: newModel,
		}, nil
	}

	return nil, err
}

func (r *Resolver) DeleteItem(ctx context.Context, args struct {
	Id string
}) (*graphql.ID, error) {
	err := permission.Check(ctx, "query.deleteItem")
	if err != nil {
		return nil, err
	}

	itemService := ctx.Value("itemService").(item.Service)

	deletedID, err := itemService.DeleteByID(args.Id)
	result := graphql.ID(deletedID)

	if err == nil {
		return &result, nil
	}

	return nil, err
}

func (r *Resolver) Item(ctx context.Context, args struct {
	Id string
}) (*item.Resolver, error) {
	err := permission.Check(ctx, "query.item")
	if err != nil {
		return nil, err
	}

	itemService := ctx.Value("itemService").(item.Service)

	queryItem, err := itemService.FindByID(args.Id)

	if err == nil {
		return &item.Resolver{
			Model: queryItem,
		}, nil
	}

	return nil, err
}
