package serializer

// DefaultRegistry is a pre-configured registry with common serializers
var DefaultRegistry = func() *Registry {
	r := NewRegistry()
	r.Register(JSON, NewJSONSerializer())
	r.Register(Msgpack, NewMsgpackSerializer())
	return r
}()
