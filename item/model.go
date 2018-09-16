package item

import (
	"github.com/dukfaar/goUtils/relay"
	"github.com/globalsign/mgo/bson"
)

type Model struct {
	ID          bson.ObjectId `json:"_id,omitempty" bson:"_id,omitempty"`
	Name        string        `json:"name,omitempty" bson:"name,omitempty"`
	NamespaceID bson.ObjectId `json:"namespaceId,omitempty" bson:"namespaceId,omitempty"`
	XivdbID     *int32        `json:"xivdbid,omitempty" bson:"xivdbid,omitempty"`
}

var GraphQLType = `
type Item {
	_id: ID
	name: String
	namespaceId: ID
	xivdbid: Int
}
` +
	relay.GenerateConnectionTypes("Item")
