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

	"gopkg.in/mgo.v2"

	"github.com/gorilla/websocket"

	graphql "github.com/graph-gophers/graphql-go"
	graphqlRelay "github.com/graph-gophers/graphql-go/relay"
)

func main() {
	dbSession, err := mgo.Dial(env.GetDefaultEnvVar("DB_HOST", "localhost"))
	if err != nil {
		panic(err)
	}
	defer dbSession.Close()

	db := dbSession.DB("item")

	nsqEventbus := eventbus.NewNsqEventBus(env.GetDefaultEnvVar("NSQD_TCP_URL", "localhost:4150"), env.GetDefaultEnvVar("NSQLOOKUP_HTTP_URL", "localhost:4161"))
	permissionService := permission.NewService()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, "itemService", item.NewMgoService(db, nsqEventbus))
	ctx = context.WithValue(ctx, "permissionService", permissionService)

	schema := graphql.MustParseSchema(Schema, &Resolver{})

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

	nsqEventbus.Emit("service.up", serviceInfo)

	url := env.GetDefaultEnvVar("API_GATEWAY_HOST", "localhost") + ":" + env.GetDefaultEnvVar("API_GATEWAY_PORT", "8090")
	path := env.GetDefaultEnvVar("API_GATEWAY_PATH", "/graphql")

	fmt.Printf("gatewayFetcherUrl: %v%v", url, path)

	apiGatewayFetcher, err := dukGraphql.NewHttpFetcher(url, path)

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")

	loginApiGatewayFetcher := dukGraphql.NewClientLoginHttpFetcher(apiGatewayFetcher, clientID, clientSecret)

	result, err := loginApiGatewayFetcher.Fetch(dukGraphql.Request{
		Query: permission.Query,
	})
	queryResult := dukGraphql.Response{result}

	permission.ParseQueryResponse(queryResult, permissionService)
	permissionService.BuildUserPermissionData()

	nsqEventbus.On("service.up", "item", func(msg []byte) error {
		newService := eventbus.ServiceInfo{}
		json.Unmarshal(msg, &newService)

		if newService.Name == "apigateway" {
			nsqEventbus.Emit("service.up", serviceInfo)
		}

		return nil
	})

	permission.AddAuthEventsHandlers(nsqEventbus, permissionService)

	log.Fatal(http.ListenAndServe(":"+env.GetDefaultEnvVar("PORT", "8080"), nil))
}
