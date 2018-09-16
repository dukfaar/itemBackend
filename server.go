package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/dukfaar/goUtils/env"
	"github.com/dukfaar/goUtils/eventbus"
	dukGraphql "github.com/dukfaar/goUtils/graphql"
	dukHttp "github.com/dukfaar/goUtils/http"
	"github.com/dukfaar/goUtils/permission"
	"github.com/dukfaar/itemBackend/item"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"

	"github.com/gorilla/websocket"

	graphql "github.com/graph-gophers/graphql-go"
	graphqlRelay "github.com/graph-gophers/graphql-go/relay"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func createApiGatewayFetcher() dukGraphql.Fetcher {
	url := env.GetDefaultEnvVar("API_GATEWAY_HOST", "localhost") + ":" + env.GetDefaultEnvVar("API_GATEWAY_PORT", "8090")
	path := env.GetDefaultEnvVar("API_GATEWAY_PATH", "/graphql")

	apiGatewayFetcher, err := dukGraphql.NewHttpFetcher(url, path)

	if err != nil {
		panic(err)
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	loginApiGatewayFetcher := dukGraphql.NewClientLoginHttpFetcher(apiGatewayFetcher, clientID, clientSecret)

	return loginApiGatewayFetcher
}

func main() {
	dbSession, err := mgo.Dial(env.GetDefaultEnvVar("DB_HOST", "localhost"))
	if err != nil {
		panic(err)
	}
	defer dbSession.Close()

	db := dbSession.DB("item")

	nsqEventbus := eventbus.NewNsqEventBus(env.GetDefaultEnvVar("NSQD_TCP_URL", "localhost:4150"), env.GetDefaultEnvVar("NSQLOOKUP_HTTP_URL", "localhost:4161"))
	permissionService := permission.NewService()

	itemService := item.NewMgoService(db, nsqEventbus)

	loginApiGatewayFetcher := createApiGatewayFetcher()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, "itemService", itemService)
	ctx = context.WithValue(ctx, "permissionService", permissionService)
	ctx = context.WithValue(ctx, "eventbus", nsqEventbus)
	ctx = context.WithValue(ctx, "apigatewayfetcher", loginApiGatewayFetcher)

	resolver := &Resolver{}
	schema := graphql.MustParseSchema(Schema, resolver)

	http.Handle("/graphql", dukHttp.AddContext(ctx, dukHttp.Authenticate(&graphqlRelay.Handler{
		Schema: schema,
	})))

	http.Handle("/socket", dukHttp.AddContext(ctx, &dukGraphql.SocketHandler{
		Schema: schema,
		Upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}))

	serviceInfo := eventbus.ServiceInfo{
		Name:                  "item",
		Hostname:              env.GetDefaultEnvVar("PUBLISHED_HOSTNAME", "itembackend"),
		Port:                  env.GetDefaultEnvVar("PUBLISHED_PORT", "8080"),
		GraphQLHttpEndpoint:   "/graphql",
		GraphQLSocketEndpoint: "/socket",
	}

	result, err := loginApiGatewayFetcher.Fetch(dukGraphql.Request{
		Query: permission.Query,
	})
	if err != nil {
		panic(err)
	}
	queryResult := dukGraphql.Response{result}

	permission.ParseQueryResponse(queryResult, permissionService)
	permissionService.BuildAllUserPermissionData()

	nsqEventbus.On("service.up", "item", func(msg []byte) error {
		newService := eventbus.ServiceInfo{}
		json.Unmarshal(msg, &newService)

		if newService.Name == "apigateway" {
			nsqEventbus.Emit("service.up", serviceInfo)
		}

		return nil
	})

	nsqEventbus.On("import.item.by.rcname", "item", func(msg []byte) error {
		var itemData struct {
			Name        string `json:"name"`
			NamespaceID string `json:"namespace"`
			//Add other vars here
		}

		err := json.Unmarshal(msg, &itemData)

		if err != nil {
			fmt.Printf("Error(%v) unmarshaling event data: %v\n", err, string(msg))
			return err
		}

		if itemData.Name == "" {
			fmt.Printf("Cant import an item without a name\n")
			return nil
		}

		itemModel, err := itemService.FindByName(itemData.Name)

		if err != nil {
			if err.Error() == "not found" {

				var newItemModel = item.Model{}
				newItemModel.Name = itemData.Name
				newItemModel.NamespaceID = bson.ObjectIdHex(itemData.NamespaceID)

				_, err := itemService.Create(&newItemModel)

				if itemData.Name == "Hardsilver Staff" {
					fmt.Printf("%v %v\n", itemData.Name, newItemModel.Name)
				}

				if err != nil {
					fmt.Printf("Error(%v) saving new item: %v\n", err, newItemModel)
				}
			} else {
				fmt.Printf("Unknown error: %v\n", err)
			}

			return err
		}

		if itemModel == nil {
			fmt.Println("Model not found")
		} else {
			if itemData.Name == "Hardsilver Staff" {
				fmt.Printf("%v %v\n", itemData.Name, itemModel.Name)
			}
			itemModel.Name = itemData.Name
			itemModel.NamespaceID = bson.ObjectIdHex(itemData.NamespaceID)

			_, err := itemService.Update(itemModel.ID.Hex(), &itemModel)

			if err != nil {
				fmt.Printf("Error(%v) updating item: %v\n", err, itemModel)
			}
		}

		return nil
	})

	nsqEventbus.On("import.item.by.xivdbid", "item", func(msg []byte) error {
		fmt.Println(string(msg))
		return nil
	})

	nsqEventbus.Emit("service.up", serviceInfo)

	permission.AddAuthEventsHandlers(nsqEventbus, permissionService)

	http.Handle("/metrics", promhttp.Handler())

	dukGraphql.EmitRegisterEvents("registerQuery", schema.Inspect().QueryType(), nsqEventbus)
	dukGraphql.EmitRegisterEvents("registerMutation", schema.Inspect().MutationType(), nsqEventbus)
	dukGraphql.EmitRegisterEvents("registerSubscription", schema.Inspect().SubscriptionType(), nsqEventbus)
	dukGraphql.EmitRegisterTypeEvents("registerType", schema.Inspect().Types(), nsqEventbus)

	log.Fatal(http.ListenAndServe(":"+env.GetDefaultEnvVar("PORT", "8080"), nil))
}
