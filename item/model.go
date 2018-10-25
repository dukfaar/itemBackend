package item

import (
	"reflect"

	"github.com/dukfaar/goUtils/graphql"
	"github.com/dukfaar/goUtils/relay"
	"github.com/globalsign/mgo/bson"
)

type UnspoiledNodeTime struct {
	Time           *int32  `json:"time,omitempty" bson:"time,omitempty" gql:"time"`
	Duration       *int32  `json:"duration,omitempty" bson:"duration,omitempty" gql:"duration"`
	AmPm           *string `json:"ampm,omitempty" bson:"ampm,omitempty" gql:"ampm"`
	FolkloreNeeded *string `json:"folkloreNeeded,omitempty" bson:"folkloreNeeded,omitempty" gql:"folkloreNeeded"`
}

type Model struct {
	ID                bson.ObjectId      `json:"_id,omitempty" bson:"_id,omitempty" gql:"_id"`
	Name              string             `json:"name,omitempty" bson:"name,omitempty" gql:"name"`
	NamespaceID       bson.ObjectId      `json:"namespaceId,omitempty" bson:"namespaceId,omitempty" gql:"namespaceId"`
	XivdbID           *int32             `json:"xivdbid,omitempty" bson:"xivdbid,omitempty" gql:"xivdbId"`
	GatheringLevel    *int32             `json:"gatheringLevel,omitempty" bson:"gatheringLevel,omitempty" gql:"gatheringLevel"`
	GatheringJobID    *bson.ObjectId     `json:"gatheringJobId,omitempty" bson:"gatheringJobId,omitempty" gql:"gatheringJobId"`
	GatheringEffort   *int32             `json:"gatheringEffort,omitempty" bson:"gatheringEffort,omitempty" gql:"gatheringEffort"`
	Price             *int32             `json:"price,omitempty" bson:"price,omitempty" gql:"price"`
	PriceHQ           *int32             `json:"priceHQ,omitempty" bson:"priceHQ,omitempty" gql:"priceHq"`
	UnspoiledNode     *bool              `json:"unspoiledNode,omitempty" bson:"unspoiledNode,omitempty" gql:"unspoiledNode"`
	UnspoiledNodeTime *UnspoiledNodeTime `json:"unspoiledNodeTime,omitempty" bson:"unspoiledNodeTime,omitempty" gql:"unspoiledNodeTime"`
	AvailableFromNpc  *bool              `json:"availableFromNpc,omitempty" bson:"availableFromNpc,omitempty" gql:"availableFromNpc"`
}

var GraphQLType = graphql.Build(reflect.TypeOf((*UnspoiledNodeTime)(nil)).Elem(), "UnspoiledNodeTime") +
	graphql.Build(reflect.TypeOf((*Model)(nil)).Elem(), "Item") +
	relay.GenerateConnectionTypes("Item")
