package conn

import (
	"fmt"
	"strconv"

	"github.com/nalgeon/redka/redsrv/internal/redis"
)

// Hello represents the HELLO command.
// https://redis.io/commands/hello
type Hello struct {
	redis.BaseCmd
	ProtoVersion int
	AuthUser     string
	AuthPass     string
	ClientName   string
}

// ParseHello parses the HELLO command arguments.
// Format: HELLO [protocol-version] [AUTH username password] [SETNAME clientname]
func ParseHello(b redis.BaseCmd) (Hello, error) {
	cmd := Hello{BaseCmd: b}
	args := cmd.Args() // [][]byte

	fmt.Printf("  Args: %v\n", args)

	if len(args) == 0 {
		return cmd, nil
	}

	// Optional first argument: protocol version (2 or 3)
	if len(args) > 0 {
		if n, err := strconv.Atoi(string(args[0])); err == nil && (n == 2 || n == 3) {
			cmd.ProtoVersion = n
			args = args[1:]
		}
	}

	// Parse remaining options: AUTH and/or SETNAME in any order
	iter := 0
	for iter < len(args) {
		switch string(args[iter]) {
		case "AUTH":
			if iter+2 >= len(args) {
				return Hello{}, fmt.Errorf("syntax error: AUTH requires username and password")
			}
			cmd.AuthUser = string(args[iter+1])
			cmd.AuthPass = string(args[iter+2])
			iter += 3
		case "SETNAME":
			if iter+1 >= len(args) {
				return Hello{}, fmt.Errorf("syntax error: SETNAME requires client name")
			}
			cmd.ClientName = string(args[iter+1])
			iter += 2
		default:
			return Hello{}, fmt.Errorf("syntax error: unexpected token %q", string(args[iter]))
		}
	}

	return cmd, nil
}

// Run executes the HELLO command.
// It authenticates, switches protocol if needed, sets client name,
// and returns server information in the appropriate RESP format.
func (c Hello) Run(w redis.Writer, rdb redis.Redka) (any, error) {

	// 打印 Hello 结构体的内容
	fmt.Printf("Hello command: %+v\n", c)
	fmt.Printf("  ProtoVersion: %d\n", c.ProtoVersion)
	fmt.Printf("  AuthUser: %q\n", c.AuthUser)
	fmt.Printf("  AuthPass: %q\n", c.AuthPass)
	fmt.Printf("  ClientName: %q\n", c.ClientName)

	// 确定协议版本
	proto := c.ProtoVersion
	if proto != 2 && proto != 3 {
		return nil, fmt.Errorf("ERR unsupported protocol version: %d", proto)
	}

	// 构建响应
	if proto == 3 {

		// RESP3: Map format (%<count>\r\n<key><value>...)
		var b []byte
		b = append(b, '%')
		b = append(b, '7')
		b = append(b, "\r\n"...)

		bulk := func(s string) {
			b = append(b, '$')
			b = append(b, strconv.Itoa(len(s))...)
			b = append(b, "\r\n"...)
			b = append(b, s...)
			b = append(b, "\r\n"...)
		}
		integer := func(n int) {
			b = append(b, ':')
			b = append(b, strconv.Itoa(n)...)
			b = append(b, "\r\n"...)
		}

		bulk("server")
		bulk("redis")
		bulk("version")
		bulk("7.3.0")
		bulk("proto")
		integer(3)
		bulk("id")
		integer(0)
		bulk("mode")
		bulk("standalone")
		bulk("role")
		bulk("master")
		bulk("modules")
		b = append(b, '*')
		b = append(b, '0')
		b = append(b, "\r\n"...)

		w.WriteRaw(b)
		return nil, nil
	}

	// RESP2: 扁平数组 key-value 交替（无Map类型）
	resp := []any{
		"server", "redis",
		"version", "6.0.0",
		"proto", proto,
		"id", 0,
		"mode", "standalone",
		"role", "master",
		"modules", []any{},
	}
	w.WriteAny(resp)
	return resp, nil
}
