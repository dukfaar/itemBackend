package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dukfaar/goUtils/eventbus"
	dukgraphql "github.com/dukfaar/goUtils/graphql"
	"github.com/dukfaar/goUtils/permission"
	"github.com/dukfaar/goUtils/relay"
	"github.com/dukfaar/itemBackend/item"
	"github.com/globalsign/mgo/bson"
	graphql "github.com/graph-gophers/graphql-go"
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

func (r *Resolver) SearchItems(ctx context.Context, args struct {
	First  *int32
	Last   *int32
	Before *string
	After  *string
	Name   *string
}) (*item.ConnectionResolver, error) {
	err := permission.Check(ctx, "query.searchItems")
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
		options := "i"
		result, _ := itemService.FindByRegexName(args.First, args.Last, args.Before, args.After, *args.Name, options)
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

func fetchFFXIVNamespace(ctx context.Context) (string, error) {
	fetcher := ctx.Value("apigatewayfetcher").(dukgraphql.Fetcher)

	namespaceResult, err := fetcher.Fetch(dukgraphql.Request{
		Query: "query { namespaceByName(name: \"FFXIV\") { _id name } }",
	})

	if err != nil {
		fmt.Printf("Error fetching namespace: %v\n", err)
		return "", err
	}

	namespaceResponse := dukgraphql.Response{namespaceResult}

	return namespaceResponse.GetObject("namespaceByName").GetString("_id"), nil
}

func (r *Resolver) RcItemImport(ctx context.Context) (string, error) {
	err := permission.Check(ctx, "mutation.rcItemImport")
	if err != nil {
		return "No Permission", err
	}

	rcItemResponse, err := http.Get("https://rc.dukfaar.com/api/item")

	if err != nil {
		fmt.Printf("Error getting items: %v\n", err)
		return "Error reading from RC", err
	}
	defer rcItemResponse.Body.Close()

	var itemsData struct {
		Count int                      `json:"count"`
		List  []map[string]interface{} `json:"list"`
	}
	err = json.NewDecoder(rcItemResponse.Body).Decode(&itemsData)

	if err != nil {
		fmt.Printf("Error reading items: %v\n", err)
		return "Error parsing data from RC", err
	}

	eventbus := ctx.Value("eventbus").(eventbus.EventBus)
	namespaceId, err := fetchFFXIVNamespace(ctx)
	if err != nil {
		return "Error fetching namespace", err
	}

	go func() {
		for index := range itemsData.List {
			item := itemsData.List[index]
			item["namespace"] = namespaceId
			eventbus.Emit("import.item.by.rcname", item)
		}
	}()

	return "OK", nil
}

func (r *Resolver) XivdbItemImport(ctx context.Context) (string, error) {
	err := permission.Check(ctx, "mutation.xivdbItemImport")
	if err != nil {
		return "No Permission", err
	}

	itemListResponse, err := http.Get("https://api.xivdb.com/item?columns=id")

	if err != nil {
		fmt.Errorf("Error getting item list: %v", err)
		return "", err
	}
	defer itemListResponse.Body.Close()

	itemList := make([]XivdbItemListResponse, 0)
	err = json.NewDecoder(itemListResponse.Body).Decode(&itemList)

	if err != nil {
		fmt.Errorf("Error reading item list: %v", err)
		return "", err
	}

	eventbus := ctx.Value("eventbus").(eventbus.EventBus)

	go func() {
		namespaceId, err := fetchFFXIVNamespace(ctx)
		if err != nil {
			return
		}

		for index, _ := range itemList {
			item := itemList[index]
			itemData, err := FetchXivdbItemData(item.ID)

			if err != nil {
				fmt.Printf("Skipping item with id: %v\n", item.ID)
				time.Sleep(time.Millisecond * 200)
				continue
			}

			resultItem := make(map[string]interface{})
			json.Unmarshal(itemData, &resultItem)
			resultItem["namespace"] = namespaceId

			delete(resultItem, "special_shops_obtain")
			delete(resultItem, "special_shops_currency")

			err = eventbus.Emit("import.item.by.xivdbid", resultItem)

			if err != nil {
				fmt.Println(err)
			}

			time.Sleep(time.Millisecond * 200)
		}
	}()

	return "OK", nil
}
