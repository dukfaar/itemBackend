package main

import (
	"github.com/dukfaar/goUtils/com/dukfaar/relay"
	"github.com/dukfaar/itemBackend/com/dukfaar/item"
)

var Schema string = `
		schema {
			query: Query
			mutation: Mutation
		}

		type Query {
			items(first: Int, last: Int, before: String, after: String): ItemConnection!
			item(id: ID): Item!
		}

		type Mutation {
			createItem(name: String, namespaceId: ID): Item!
		}` +
	relay.PageInfoGraphQLString +
	item.GraphQLType
