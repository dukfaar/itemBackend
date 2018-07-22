package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dukfaar/goUtils/env"
	"github.com/dukfaar/goUtils/eventbus"
	dukGraphql "github.com/dukfaar/goUtils/graphql"
	dukHttp "github.com/dukfaar/goUtils/http"
	"github.com/dukfaar/itemBackend/item"
	"github.com/dukfaar/itemBackend/permission"

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

func JSTimestampToTime(timestamp int64) time.Time {
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

type LoginSuccessMsg struct {
	UserID               string `json:"userId,omitempty"`
	AccessToken          string `json:"accessToken,omitempty"`
	AccessTokenExpiresAt string `json:"accessTokenExpiresAt,omitempty"`
}

type RoleMsg struct {
	ID          string   `json:"_id,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

type PermissionMsg struct {
	ID   string `json:"_id,omitempty"`
	Name string `json:"name,omitempty"`
}

type UserMsg struct {
	ID    string   `json:"_id,omitempty"`
	Roles []string `json:"roles,omitempty"`
}

func AddAuthEventsHandlers(nsqEventbus *eventbus.NsqEventBus, permissionService *permission.Service) {
	hostname, err := os.Hostname()
	if err != nil {
		panic("Could not determine Hostname")
	}

	channelName := "item_" + hostname

	permissionHandler := func(msg []byte) error {
		permissionMsg := PermissionMsg{}

		err := json.Unmarshal(msg, &permissionMsg)
		if err != nil {
			err = fmt.Errorf("Error unmarshaling: %v: %v", string(msg), err)
			fmt.Println(err)
			return err
		}

		permissionService.SetPermission(permissionMsg.ID, permissionMsg.Name)
		permissionService.BuildUserPermissionData()

		return nil
	}

	roleHandler := func(msg []byte) error {
		roleMsg := RoleMsg{}

		err := json.Unmarshal(msg, &roleMsg)
		if err != nil {
			err = fmt.Errorf("Error unmarshaling: %v: %v", string(msg), err)
			fmt.Println(err)
			return err
		}

		permissionService.SetRole(roleMsg.ID, roleMsg.Permissions)
		permissionService.BuildUserPermissionData()

		return nil
	}

	userHandler := func(msg []byte) error {
		userMsg := UserMsg{}

		err := json.Unmarshal(msg, &userMsg)
		if err != nil {
			err = fmt.Errorf("Error unmarshaling: %v: %v", string(msg), err)
			fmt.Println(err)
			return err
		}

		permissionService.SetUser(userMsg.ID, userMsg.Roles)
		permissionService.BuildUserPermissionData()

		return nil
	}

	nsqEventbus.On("permission.created", channelName, permissionHandler)
	nsqEventbus.On("permission.updated", channelName, permissionHandler)

	nsqEventbus.On("role.created", channelName, roleHandler)
	nsqEventbus.On("role.updated", channelName, roleHandler)

	nsqEventbus.On("user.created", channelName, userHandler)
	nsqEventbus.On("user.updated", channelName, userHandler)

	nsqEventbus.On("login.success", channelName, func(msg []byte) error {
		loginSuccess := LoginSuccessMsg{}

		err := json.Unmarshal(msg, &loginSuccess)
		if err != nil {
			err = fmt.Errorf("Error unmarshaling: %v: %v", string(msg), err)
			fmt.Println(err)
			return err
		}

		expiresAt, err := time.Parse(time.RFC3339Nano, loginSuccess.AccessTokenExpiresAt)
		if err != nil {
			err = fmt.Errorf("Error parsing time: %v: %v", string(msg), err)
			fmt.Println(err)
			return err
		}

		permissionService.SetToken(loginSuccess.UserID, loginSuccess.AccessToken, expiresAt)
		permissionService.BuildUserPermissionData()
		return nil
	})
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

	tokenEdges := queryResult.GetObject("tokens").GetArray("edges")
	for j := 0; j < tokenEdges.Len(); j++ {
		tokenEdge := tokenEdges.Get(j)
		token := tokenEdge.GetObject("node")
		accessTokenExpiresAt, _ := token.GetInt64("accessTokenExpiresAt")
		expiresAt := JSTimestampToTime(accessTokenExpiresAt)

		permissionService.SetToken(token.GetString("accessToken"), token.GetString("userId"), expiresAt)
	}

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

		permissionService.SetUser(id, userRoles)
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

		permissionService.SetRole(id, rolePermissions)
	}

	permissionEdges := queryResult.GetObject("permissions").GetArray("edges")
	for j := 0; j < permissionEdges.Len(); j++ {
		permissionEdge := permissionEdges.Get(j)
		permission := permissionEdge.GetObject("node")
		permissionService.SetPermission(permission.GetString("_id"), permission.GetString("name"))
	}

	permissionService.BuildUserPermissionData()

	nsqEventbus.On("service.up", "item", func(msg []byte) error {
		newService := eventbus.ServiceInfo{}
		json.Unmarshal(msg, &newService)

		if newService.Name == "apigateway" {
			nsqEventbus.Emit("service.up", serviceInfo)
		}

		return nil
	})

	AddAuthEventsHandlers(nsqEventbus, permissionService)

	log.Fatal(http.ListenAndServe(":"+env.GetDefaultEnvVar("PORT", "8080"), nil))
}
