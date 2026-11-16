package valid

import "example.com/consumer/api"

type Server struct{ cache api.Handle }

// OnFailure implements the ordinary callback interface.
func (*Server) OnFailure(error) {}

var _ interface{ OnFailure(error) } = (*Server)(nil)
var nilReference = (*api.Handle)(nil)
var nilReferences = []api.Handle(nil)
var nilMap = map[string]api.Handle(nil)
var nilChannel = chan api.Handle(nil)
var nilCallback = (func() api.Handle)(nil)

type ServerPointer *Server
type Copy api.Handle
type Box[T any] struct{ Value T }

// OrdinaryConversions checks typed nils and conversions of reference holders.
func OrdinaryConversions(ref api.Handle, server *Server) api.Handle {
	pointer := ServerPointer(server)
	_ = (*Server)(pointer)
	restored := api.Handle(Copy(ref))
	restored.Borrow().Count()
	holder := Box[api.Handle]{Value: restored}
	return identity(holder).Value
}

// indirect uses a service pointer from a generic method contract locally.
func indirect[T interface{ Borrow() *api.Counter }](value T) int {
	worker := value.Borrow()
	return worker.Count()
}

// GenericCall uses ordinary Go dispatch through a type parameter.
func GenericCall(ref api.Handle) int { return indirect(ref) }
