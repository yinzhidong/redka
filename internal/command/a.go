package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/nalgeon/redka"
)

var ErrInvalidInt = errors.New("ERR value is not an integer or out of range")
var ErrNestedMulti = errors.New("ERR MULTI calls can not be nested")
var ErrNotInMulti = errors.New("ERR EXEC without MULTI")
var ErrSyntax = errors.New("ERR syntax error")

func ErrInvalidArgNum(cmd string) error {
	return fmt.Errorf("ERR wrong number of arguments for '%s' command", cmd)
}
func ErrInvalidExpireTime(cmd string) error {
	return fmt.Errorf("ERR invalid expire time in '%s' command", cmd)
}
func ErrUnknownCmd(cmd string) error {
	return fmt.Errorf("ERR unknown command '%s'", cmd)
}
func ErrUnknownSubcmd(cmd, subcmd string) error {
	return fmt.Errorf("ERR unknown subcommand '%s %s'", cmd, subcmd)
}

// Redka is a Redis-like repository.
type Redka interface {
	Key() redka.Keys
	Str() redka.Strings
}

type Writer interface {
	WriteError(msg string)
	WriteString(str string)
	WriteBulk(bulk []byte)
	WriteBulkString(bulk string)
	WriteInt(num int)
	WriteInt64(num int64)
	WriteUint64(num uint64)
	WriteArray(count int)
	WriteNull()
	WriteRaw(data []byte)
	WriteAny(v any)
}

// Cmd is a Redis-compatible command.
type Cmd interface {
	Name() string
	Err() error
	String() string

	Run(w Writer, red Redka) (any, error)
}

type baseCmd struct {
	name string
	args [][]byte
	err  error
}

func newBaseCmd(args [][]byte) baseCmd {
	return baseCmd{
		name: strings.ToLower(string(args[0])),
		args: args[1:],
	}
}

func (cmd baseCmd) Name() string {
	return cmd.name
}
func (cmd baseCmd) String() string {
	var b strings.Builder
	for i, arg := range cmd.args {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.Write(arg)
	}
	return b.String()
}
func (cmd baseCmd) Err() error {
	return cmd.err
}

// Parse parses a text representation of a command into a Cmd.
func Parse(args [][]byte) (Cmd, error) {
	name := strings.ToLower(string(args[0]))
	b := newBaseCmd(args)
	switch name {
	// server
	case "command":
		return parseOK(b)
	case "info":
		return parseOK(b)
	// connection
	case "echo":
		return parseEcho(b)
	// string
	case "get":
		return parseGet(b)
	case "set":
		return parseSet(b)
	default:
		return parseUnknown(b)
	}
}
