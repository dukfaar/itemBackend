package main

import (
	"context"
	"encoding/json"
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

type GraphQLResponseArray struct {
	response []interface{}
}

type GraphQLResponse struct {
	response interface{}
}

func (r *GraphQLResponseArray) Get(index int) *GraphQLResponse {
	return &GraphQLResponse{r.response[index]}
}

func (r *GraphQLResponseArray) Len() int {
	return len(r.response)
}

func (r *GraphQLResponse) GetObject(key string) *GraphQLResponse {
	return &GraphQLResponse{(r.response.(map[string]interface{}))[key]}
}

func (r *GraphQLResponse) GetArray(key string) *GraphQLResponseArray {
	return &GraphQLResponseArray{r.response.(map[string]interface{})[key].([]interface{})}
}

func (r *GraphQLResponse) GetString(key string) string {
	return (r.response.(map[string]interface{}))[key].(string)
}

func (r *GraphQLResponse) GetInt64(key string) (int64, error) {
	return strconv.ParseInt(r.GetString(key), 10, 64)
}

func JSTimestampToTime(timestamp string) time.Time {
	seconds := timestamp / 1000
	nSeconds := (timestamp % 1000) * 1e6

	return time.Unix(seconds, nSeconds)
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

	clientlogin := GraphQLResponse{result}
	token := clientlogin.GetObject("clientlogin")
	f.accessToken = token.GetString("accessToken")
	accessTokenExpiresAt, _ := token.GetInt64("accessTokenExpiresAt")

	f.accessTokenExpiresAt = JSTimestampToTime(accessTokenExpiresAt)

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
								}
							}
						}
					}
				}
			}
			roles {
				edges {
					node {
						_id
						permissions {
							edges {
								node {
									_id
								}
							}
						}
					}
				}
			}
			permissions {
				edges {
					node {
						_id
						name
					}
				}
			}
			tokens {
				edges {
					node {
						accessToken
						accessTokenExpiresAt
						userId
					}
				}
			}
		}`,
	})
	queryResult := GraphQLResponse{result}

	type TokenData struct {
		userId               string
		accessTokenExpiresAt time.Time
	}

	tokenData := make(map[string]TokenData)
	userRoleData := make(map[string][]string)
	rolePermissionData := make(map[string][]string)
	permissionData := make(map[string]string)

	userEdges := queryResult.GetObject("users").GetArray("edges")
	for i := 0; i < userEdges.Len(); i++ {
		userEdge := userEdges.Get(i)
		user := userEdge.GetObject("node")
		id := user.GetString("_id")

		roleEdges := user.GetObject("roles").GetArray("edges")
		userRoles := make([]string, roleEdges.Len())
		for j := 0; j < roleEdges.Len(); j++ {
			roleEdge := roleEdges.Get(j)
			role := roleEdge.GetObject("node")
			userRoles[j] = role.GetString("_id")
		}

		userRoleData[id] = userRoles
	}

	roleEdges := queryResult.GetObject("roles").GetArray("edges")
	for i := 0; i < roleEdges.Len(); i++ {
		roleEdge := roleEdges.Get(i)
		role := roleEdge.GetObject("node")
		id := role.GetString("_id")

		permissionEdges := role.GetObject("permissions").GetArray("edges")
		rolePermissions := make([]string, permissionEdges.Len())
		for j := 0; j < permissionEdges.Len(); j++ {
			permissionEdge := permissionEdges.Get(j)
			permission := permissionEdge.GetObject("node")
			rolePermissions[j] = permission.GetString("_id")
		}

		rolePermissionData[id] = rolePermissions
	}

	permissionEdges := queryResult.GetObject("permissions").GetArray("edges")
	for j := 0; j < permissionEdges.Len(); j++ {
		permissionEdge := permissionEdges.Get(j)
		permission := permissionEdge.GetObject("node")
		permissionData[permission.GetString("_id")] = permission.GetString("name")
	}

	tokenEdges := queryResult.GetObject("tokens").GetArray("edges")
	for j := 0; j < tokenEdges.Len(); j++ {
		tokenEdge := tokenEdges.Get(j)
		token := tokenEdge.GetObject("node")
		accessTokenExpiresAt, _ := token.GetInt64("accessTokenExpiresAt")
		expiresAt := JSTimestampToTime(accessTokenExpiresAt)

		tokenData[permission.GetString("accessToken")] = TokenData{
			userId: permission.GetString("userId"),
			accessTokenExpiresAt: expiresAt,
	}

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
