package main

import (
	"github.com/dukfaar/goUtils/relay"
	"github.com/dukfaar/itemBackend/item"
)

var Schema string = `
		schema {
			query: Query
			mutation: Mutation
		}

		type Query {
			items(first: Int, last: Int, before: String, after: String, name: String): ItemConnection!
			item(id: ID!): Item!

			findItem(name: String, namespaceId: ID): Item
		}

		type Mutation {
			createItem(name: String, namespaceId: ID): Item!
			updateItem(id: ID!, name: String, namespaceId: ID): Item!
			deleteItem(id: ID!): ID

			rcItemImport(): String!
			xivdbItemImport(): String!
		}` +
	relay.PageInfoGraphQLString +
	item.GraphQLType
