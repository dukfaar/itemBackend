package item

import "github.com/dukfaar/goUtils/com/dukfaar/relay"

type ConnectionResolver struct {
	Models []Model
	relay.ConnectionResolver
}

func (r *ConnectionResolver) Edges() *[]*EdgeResolver {
	l := make([]*EdgeResolver, len(r.Models))
	for i := range l {
		l[i] = &EdgeResolver{
			model: &r.Models[i],
		}
	}
	return &l
}
