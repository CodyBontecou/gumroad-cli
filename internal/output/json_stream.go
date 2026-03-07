package output

import (
	"encoding/json"
	"errors"
	"io"
)

type jsonStreamInputIter struct {
	reader io.ReadCloser
	stream *jsonTokenStream
	err    error
}

func newJSONStreamInputIter(reader io.ReadCloser) *jsonStreamInputIter {
	dec := json.NewDecoder(reader)
	dec.UseNumber()
	return &jsonStreamInputIter{
		reader: reader,
		stream: newJSONTokenStream(dec),
	}
}

func (i *jsonStreamInputIter) Next() (any, bool) {
	if i.err != nil {
		return nil, false
	}

	value, err := i.stream.next()
	if err != nil {
		if errors.Is(err, io.EOF) {
			i.err = err
			return nil, false
		}
		i.err = err
		return err, true
	}
	return value, true
}

func (i *jsonStreamInputIter) Close() error {
	i.err = io.EOF
	if i.reader == nil {
		return nil
	}
	return i.reader.Close()
}

type jsonTokenStream struct {
	dec    *json.Decoder
	path   []any
	states []int
}

func newJSONTokenStream(dec *json.Decoder) *jsonTokenStream {
	return &jsonTokenStream{
		dec:    dec,
		path:   []any{},
		states: []int{jsonStateTopValue},
	}
}

const (
	jsonStateTopValue = iota
	jsonStateArrayStart
	jsonStateArrayValue
	jsonStateArrayEnd
	jsonStateArrayEmptyEnd
	jsonStateObjectStart
	jsonStateObjectKey
	jsonStateObjectValue
	jsonStateObjectEnd
	jsonStateObjectEmptyEnd
)

func (s *jsonTokenStream) next() (any, error) {
	switch s.states[len(s.states)-1] {
	case jsonStateArrayEnd, jsonStateObjectEnd:
		s.path = s.path[:len(s.path)-1]
		fallthrough
	case jsonStateArrayEmptyEnd, jsonStateObjectEmptyEnd:
		s.states = s.states[:len(s.states)-1]
	}

	if s.dec.More() {
		switch s.states[len(s.states)-1] {
		case jsonStateArrayValue:
			s.path[len(s.path)-1] = s.path[len(s.path)-1].(int) + 1
		case jsonStateObjectValue:
			s.path = s.path[:len(s.path)-1]
		}
	}

	for {
		token, err := s.dec.Token()
		if err != nil {
			if err == io.EOF && s.states[len(s.states)-1] != jsonStateTopValue {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}

		if delim, ok := token.(json.Delim); ok {
			switch delim {
			case '[', '{':
				switch s.states[len(s.states)-1] {
				case jsonStateArrayStart:
					s.states[len(s.states)-1] = jsonStateArrayValue
				case jsonStateObjectKey:
					s.states[len(s.states)-1] = jsonStateObjectValue
				}

				if delim == '[' {
					s.states = append(s.states, jsonStateArrayStart)
					s.path = append(s.path, 0)
				} else {
					s.states = append(s.states, jsonStateObjectStart)
				}
			case ']':
				if s.states[len(s.states)-1] == jsonStateArrayStart {
					s.states[len(s.states)-1] = jsonStateArrayEmptyEnd
					s.path = s.path[:len(s.path)-1]
					return []any{s.copyPath(), []any{}}, nil
				}
				s.states[len(s.states)-1] = jsonStateArrayEnd
				return []any{s.copyPath()}, nil
			case '}':
				if s.states[len(s.states)-1] == jsonStateObjectStart {
					s.states[len(s.states)-1] = jsonStateObjectEmptyEnd
					return []any{s.copyPath(), map[string]any{}}, nil
				}
				s.states[len(s.states)-1] = jsonStateObjectEnd
				return []any{s.copyPath()}, nil
			}
			continue
		}

		switch s.states[len(s.states)-1] {
		case jsonStateArrayStart:
			s.states[len(s.states)-1] = jsonStateArrayValue
			fallthrough
		case jsonStateArrayValue:
			return []any{s.copyPath(), token}, nil
		case jsonStateObjectStart, jsonStateObjectValue:
			s.states[len(s.states)-1] = jsonStateObjectKey
			s.path = append(s.path, token)
		case jsonStateObjectKey:
			s.states[len(s.states)-1] = jsonStateObjectValue
			return []any{s.copyPath(), token}, nil
		default:
			s.states[len(s.states)-1] = jsonStateTopValue
			return []any{s.copyPath(), token}, nil
		}
	}
}

func (s *jsonTokenStream) copyPath() []any {
	path := make([]any, len(s.path))
	copy(path, s.path)
	return path
}
