package test

import (
	"testing"

	eyev1 "github.com/arashrasoulzadeh/eye/gen/go/eyev1"
	"google.golang.org/protobuf/proto"
)

// FuzzEventProtoUnmarshal ensures arbitrary wire input cannot panic the
// protobuf decoder used at the gRPC boundary.
func FuzzEventProtoUnmarshal(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x0a, 0x03, 't', '1', 'x'})
	f.Add([]byte{0xff, 0xff, 0xff, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		msg := new(eyev1.IngestRequest)
		_ = proto.Unmarshal(data, msg)
	})
}
