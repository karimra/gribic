package api

import (
	"errors"

	"google.golang.org/protobuf/proto"
)

type GRIBIOption func(proto.Message) error

// ErrInvalidMsgType is returned by a GRIBIOption in case the Option is supplied
// an unexpected proto.Message
var ErrInvalidMsgType = errors.New("invalid message type")

// ErrInvalidValue is returned by a GRIBIOption in case the Option is supplied
// an unexpected value.
var ErrInvalidValue = errors.New("invalid value")

// apply is a helper function that simply applies the options to the proto.Message.
// It returns an error if any of the options fails.
func apply(m proto.Message, opts ...GRIBIOption) error {
	for _, o := range opts {
		if err := o(m); err != nil {
			return err
		}
	}
	return nil
}
