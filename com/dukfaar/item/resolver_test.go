package item

import (
	"reflect"
	"testing"

	graphql "github.com/graph-gophers/graphql-go"
	"gopkg.in/mgo.v2/bson"
)

func TestResolver_ID(t *testing.T) {
	tests := []struct {
		name string
		r    *Resolver
		want graphql.ID
	}{
		{
			"",
			&Resolver{
				&Model{
					bson.ObjectIdHex("00112233445566778899aabb"),
					"def",
					bson.ObjectIdHex("10112233445566778899aabb"),
				},
			},
			"00112233445566778899aabb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := *tt.r.ID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.ID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_Name(t *testing.T) {
	tests := []struct {
		name string
		r    *Resolver
		want string
	}{
		{
			"",
			&Resolver{
				&Model{
					bson.ObjectIdHex("00112233445566778899aabb"),
					"def",
					bson.ObjectIdHex("10112233445566778899aabb"),
				},
			},
			"def",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := *tt.r.Name(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResolver_NamespaceID(t *testing.T) {
	tests := []struct {
		name string
		r    *Resolver
		want graphql.ID
	}{
		{
			"",
			&Resolver{
				&Model{
					bson.ObjectIdHex("00112233445566778899aabb"),
					"def",
					bson.ObjectIdHex("10112233445566778899aabb"),
				},
			},
			"10112233445566778899aabb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := *tt.r.NamespaceID(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Resolver.NamespaceID() = %v, want %v", got, tt.want)
			}
		})
	}
}
