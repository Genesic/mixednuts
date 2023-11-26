package mixednuts

import (
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	"github.com/smarty/assertions"
)

// ShouldResembleProto asserts that given two values that contain proto messages
// are equal by comparing their types and ensuring they serialize to the same
// text representation.
//
// Values can either each be a proto.Message or a slice of values that each
// implement proto.Message interface.
func ShouldResembleProto(actual interface{}, expected ...interface{}) string {
	if len(expected) != 1 {
		return fmt.Sprintf("ShouldResembleProto expects 1 value, got %d", len(expected))
	}
	exp := expected[0]
	// This is very crude... We want to be able to see a diff between expected
	// and actual protos, so we just serialize them into a string and compare
	// strings. This is much simpler than trying to achieve the same using
	// reflection, clearing of XXX_*** fields and ShouldResemble.
	if am, ok := protoMessage(actual); ok {
		if err := assertions.ShouldHaveSameTypeAs(actual, exp); err != "" {
			return err
		}
		em, _ := protoMessage(exp)
		return assertions.ShouldEqual(textPBMultiline.Format(am), textPBMultiline.Format(em))
	}
	lVal := reflect.ValueOf(actual)
	rVal := reflect.ValueOf(exp)
	if lVal.Kind() == reflect.Slice {
		if rVal.Kind() != reflect.Slice {
			return "ShouldResembleProto is expecting both arguments to be a slice if first one is a slice"
		}
		if err := assertions.ShouldHaveLength(actual, rVal.Len()); err != "" {
			return err
		}
		var left, right strings.Builder
		for i := 0; i < lVal.Len(); i++ {
			l := lVal.Index(i).Interface()
			r := rVal.Index(i).Interface()
			if err := assertions.ShouldHaveSameTypeAs(l, r); err != "" {
				return err
			}
			if i != 0 {
				left.WriteString("---\n")
				right.WriteString("---\n")
			}
			lm, _ := protoMessage(l)
			rm, _ := protoMessage(r)
			left.WriteString(textPBMultiline.Format(lm))
			right.WriteString(textPBMultiline.Format(rm))
		}
		return assertions.ShouldEqual(left.String(), right.String())
	}
	return fmt.Sprintf(
		"ShouldResembleProto doesn't know how to handle values of type %T, "+
			"expecting a proto.Message or a slice of thereof", actual)
}

var textPBMultiline = prototext.MarshalOptions{
	Multiline: true,
}

// protoMessage returns V2 proto message, converting v1 on the fly.
func protoMessage(a interface{}) (proto.Message, bool) {
	if m, ok := a.(proto.Message); ok {
		return m, true
	}
	return nil, false
}
