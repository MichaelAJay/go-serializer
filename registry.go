package serializer

// DefaultRegistry is a pre-configured registry with common serializers
var DefaultRegistry = func() *Registry {
	r := NewRegistry()
	r.Register(JSON, NewJSONSerializer(32*1024))
	r.Register(Msgpack, NewMsgpackSerializer())
	return r
}()
