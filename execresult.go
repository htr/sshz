package main

import "errors"

type StreamType int

const (
	Stdout StreamType = iota
	Stderr
)

func (s StreamType) String() string {
	switch s {
	case Stdout:
		return "stdout"
	case Stderr:
		return "stderr"
	}
	panic("invalid stream type")
}

func (s StreamType) MarshalJSON() ([]byte, error) {
	switch s {
	case Stdout:
		return []byte("\"stdout\""), nil
	case Stderr:
		return []byte("\"stderr\""), nil
	default:
		return nil, errors.New("invalid StreamType")
	}
}

type OutErr struct {
	Stream    StreamType
	Line      string
	Timestamp int64
}

type ExecResult struct {
	Host                string
	Output              []OutErr
	Error               error `json:",omitempty"`
	ExecutionTimeMicros int64
	SeqNum              int
}
