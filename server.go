package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/dukfaar/goUtils/env"
	"github.com/dukfaar/goUtils/eventbus"
	dukGraphql "github.com/dukfaar/goUtils/graphql"
	dukHttp "github.com/dukfaar/goUtils/http"
	"github.com/dukfaar/itemBackend/item"

	"gopkg.in/mgo.v2"

	"github.com/gorilla/websocket"

	graphql "github.com/graph-gophers/graphql-go"
	graphqlRelay "github.com/graph-gophers/graphql-go/relay"
)

type ClientLoginHttpFetcher struct {
	clientID             string
	clientSecret         string
	fetcher              *dukGraphql.HttpFetcher
	accessToken          string
	accessTokenExpiresAt time.Time
}

func NewClientLoginHttpFetcher(fetcher *dukGraphql.HttpFetcher, clientID string, clientSecret string) *ClientLoginHttpFetcher {
	return &ClientLoginHttpFetcher{
		fetcher:              fetcher,
		clientID:             clientID,
		clientSecret:         clientSecret,
		accessTokenExpiresAt: time.Now(),
	}
}

func (f *ClientLoginHttpFetcher) doLogin() error {
	result, err := f.fetcher.Fetch(dukGraphql.Request{
		Query: `query {
			clientlogin(clientId: "` + f.clientID + `", clientSecret: "` + f.clientSecret + `") {
				accessToken
				accessTokenExpiresAt
			}
		}`,
	})

	if err != nil {
		return err
	}

	clientlogin := result.(map[string]interface{})
	token := clientlogin["clientlogin"].(map[string]interface{})
	f.accessToken = token["accessToken"].(string)
	accessTokenExpiresAt, _ := strconv.ParseInt(token["accessTokenExpiresAt"].(string), 10, 64)
	accessTokenExpiresAtSeconds := accessTokenExpiresAt / 1000
	accessTokenExpiresAtNSeconds := (accessTokenExpiresAt % 1000) * 1e6

	f.accessTokenExpiresAt = time.Unix(accessTokenExpiresAtSeconds, accessTokenExpiresAtNSeconds)

	f.fetcher.SetHeader("Authentication", "Bearer "+f.accessToken)
	return nil
}

func (f *ClientLoginHttpFetcher) Fetch(request dukGraphql.Request) (interface{}, error) {
	now := time.Now()
	if f.accessTokenExpiresAt.After(now) || f.accessTokenExpiresAt.Equal(now) {
		err := f.doLogin()

		if err != nil {
			return nil, err
		}
	}
	return f.fetcher.Fetch(request)
}

func main() {
	dbSession, err := mgo.Dial(env.GetDefaultEnvVar("DB_HOST", "localhost"))
	if err != nil {
		panic(err)
	}
	defer dbSession.Close()

	db := dbSession.DB("item")

	nsqEventbus := eventbus.NewNsqEventBus(env.GetDefaultEnvVar("NSQD_TCP_URL", "localhost:4150"), env.GetDefaultEnvVar("NSQLOOKUP_HTTP_URL", "localhost:4161"))

	ctx := context.Background()
	ctx = context.WithValue(ctx, "db", db)
	ctx = context.WithValue(ctx, "itemService", item.NewMgoService(db, nsqEventbus))

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

	apiGatewayFetcher, err := dukGraphql.NewHttpFetcher(
		env.GetDefaultEnvVar("API_GATEWAY_HOST", "localhost")+":"+env.GetDefaultEnvVar("API_GATEWAY_PORT", "8090"),
		env.GetDefaultEnvVar("API_GATEWAY_PATH", "/graphql"),
	)

	loginApiGatewayFetcher := NewClientLoginHttpFetcher(apiGatewayFetcher, "dukfaar-cloud-internal", "i am a ninja cat")

	result, err := loginApiGatewayFetcher.Fetch(dukGraphql.Request{
		Query: `query {
			users {
				edges {
					node {
						_id
						roles {
							edges {
								node {
									_id
									name
								}
							}
						}
					}
				}
			}
		}`,
	})

	fmt.Println(result)

	nsqEventbus.On("service.up", "item", func(msg []byte) error {
		newService := eventbus.ServiceInfo{}
		json.Unmarshal(msg, &newService)

		if newService.Name == "apigateway" {
			nsqEventbus.Emit("service.up", serviceInfo)
		}

		return nil
	})

	log.Fatal(http.ListenAndServe(":"+env.GetDefaultEnvVar("PORT", "8080"), nil))
}
