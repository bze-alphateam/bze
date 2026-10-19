// Package cli holds the x/dao CLI pieces autocli cannot express on its own.
package cli

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"cosmossdk.io/client/v2/autocli/flag"
	"cosmossdk.io/math"
	cosmos_proto "github.com/cosmos/cosmos-proto"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

// decScalar is the cosmos_proto scalar of LegacyDec fields.
const decScalar = "cosmos.Dec"

var isJSONFile = regexp.MustCompile(`\.json$`)

// DecJSONMessageType is an autocli flag type for a message given as JSON, or
// as a .json file holding it, like autocli's own JSON message flags, except
// that its cosmos.Dec fields take decimals: "0.5" is sent as
// 500000000000000000, LegacyDec's 18-decimal integer form. Plain autocli
// passes the string through untouched, so the chain would read "0.5" as an
// integer and refuse the tx.
type DecJSONMessageType struct {
	// Name is the full name of the message, e.g. bze.dao.DaoConfig.
	Name protoreflect.FullName
}

// NewValue implements flag.Type.
func (t DecJSONMessageType) NewValue(_ *context.Context, builder *flag.Builder) flag.Value {
	return &decJSONMessageValue{name: t.Name, builder: builder}
}

// DefaultValue implements flag.Type.
func (DecJSONMessageType) DefaultValue() string {
	return ""
}

type decJSONMessageValue struct {
	name    protoreflect.FullName
	builder *flag.Builder
	message proto.Message
}

func (v *decJSONMessageValue) Get(protoreflect.Value) (protoreflect.Value, error) {
	if v.message == nil {
		return protoreflect.Value{}, nil
	}

	return protoreflect.ValueOfMessage(v.message.ProtoReflect()), nil
}

func (v *decJSONMessageValue) String() string {
	if v.message == nil {
		return ""
	}

	bz, err := protojson.MarshalOptions{Resolver: v.builder.TypeResolver}.Marshal(v.message)
	if err != nil {
		return err.Error()
	}

	return string(bz)
}

func (v *decJSONMessageValue) Set(s string) error {
	messageType, err := v.messageType()
	if err != nil {
		return err
	}

	bz := []byte(s)
	if isJSONFile.MatchString(s) {
		if bz, err = os.ReadFile(s); err != nil {
			return err
		}
	}

	message := messageType.New().Interface()
	if err := (protojson.UnmarshalOptions{Resolver: v.builder.TypeResolver}).Unmarshal(bz, message); err != nil {
		return err
	}

	if err := decimalsToDecInts(message.ProtoReflect()); err != nil {
		return err
	}

	v.message = message

	return nil
}

func (v *decJSONMessageValue) Type() string {
	return fmt.Sprintf("%s (json)", v.name)
}

// messageType finds the registered type of the message, or builds a dynamic
// one from its descriptor.
func (v *decJSONMessageValue) messageType() (protoreflect.MessageType, error) {
	if typ, err := v.builder.TypeResolver.FindMessageByName(v.name); err == nil {
		return typ, nil
	}

	desc, err := v.builder.FileResolver.FindDescriptorByName(v.name)
	if err != nil {
		return nil, fmt.Errorf("message %s not found: %w", v.name, err)
	}

	messageDesc, ok := desc.(protoreflect.MessageDescriptor)
	if !ok {
		return nil, fmt.Errorf("%s is not a message", v.name)
	}

	return dynamicpb.NewMessageType(messageDesc), nil
}

// decimalsToDecInts rewrites every set cosmos.Dec field of the message, nested
// messages included, from a decimal to LegacyDec's 18-decimal integer form.
func decimalsToDecInts(message protoreflect.Message) error {
	var err error
	message.Range(func(fd protoreflect.FieldDescriptor, value protoreflect.Value) bool {
		switch {
		case fd.IsMap():
			// no map in the messages this type is used for
		case fd.Kind() == protoreflect.MessageKind && fd.IsList():
			list := value.List()
			for i := 0; i < list.Len() && err == nil; i++ {
				err = decimalsToDecInts(list.Get(i).Message())
			}
		case fd.Kind() == protoreflect.MessageKind:
			err = decimalsToDecInts(value.Message())
		case fd.Kind() == protoreflect.StringKind && !fd.IsList() && isDecField(fd):
			var dec math.LegacyDec
			if dec, err = math.LegacyNewDecFromStr(value.String()); err != nil {
				err = fmt.Errorf("%s: %q is not a decimal: %w", fd.Name(), value.String(), err)
				break
			}

			message.Set(fd, protoreflect.ValueOfString(dec.BigInt().String()))
		}

		return err == nil
	})

	return err
}

// isDecField reports whether the field is annotated as a cosmos.Dec scalar.
func isDecField(fd protoreflect.FieldDescriptor) bool {
	opts := fd.Options()
	if opts == nil || !proto.HasExtension(opts, cosmos_proto.E_Scalar) {
		return false
	}

	scalar, _ := proto.GetExtension(opts, cosmos_proto.E_Scalar).(string)

	return scalar == decScalar
}
