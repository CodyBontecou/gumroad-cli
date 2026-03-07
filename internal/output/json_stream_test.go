package output

import (
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestJSONStreamInputIter_UnexpectedEOF(t *testing.T) {
	iter := newJSONStreamInputIter(io.NopCloser(strings.NewReader("{")))

	value, ok := iter.Next()
	if !ok {
		t.Fatal("expected iterator error value")
	}

	err, isErr := value.(error)
	if !isErr {
		t.Fatalf("got %T, want error", value)
	}
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("got %v, want %v", err, io.ErrUnexpectedEOF)
	}

	if value, ok := iter.Next(); ok || value != nil {
		t.Fatalf("expected iterator to stop after error, got value=%v ok=%v", value, ok)
	}
}

func TestJSONStreamInputIter_CloseWithoutReader(t *testing.T) {
	iter := &jsonStreamInputIter{}

	if err := iter.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	if value, ok := iter.Next(); ok || value != nil {
		t.Fatalf("expected closed iterator to stop, got value=%v ok=%v", value, ok)
	}
}

func TestJSONStreamInputIter_EmptyContainers(t *testing.T) {
	iter := newJSONStreamInputIter(io.NopCloser(strings.NewReader(`{"items":[],"meta":{}}`)))
	defer iter.Close()

	var values []any
	for {
		value, ok := iter.Next()
		if !ok {
			break
		}
		values = append(values, value)
	}

	want := []any{
		[]any{[]any{"items"}, []any{}},
		[]any{[]any{"meta"}, map[string]any{}},
		[]any{[]any{"meta"}},
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("got %#v, want %#v", values, want)
	}
}

func TestJSONTokenStream_ArrayAndObjectValues(t *testing.T) {
	dec := json.NewDecoder(strings.NewReader(`[{"id":1},2]`))
	dec.UseNumber()
	stream := newJSONTokenStream(dec)

	var values []any
	for {
		value, err := stream.next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("next returned error: %v", err)
		}
		values = append(values, value)
	}

	want := []any{
		[]any{[]any{0, "id"}, json.Number("1")},
		[]any{[]any{0, "id"}},
		[]any{[]any{1}, json.Number("2")},
		[]any{[]any{1}},
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("got %#v, want %#v", values, want)
	}
}
