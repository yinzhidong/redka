package key

import "github.com/nalgeon/redka/internal/redis"

// Returns all key names that match a pattern.
// KEYS pattern
// https://redis.io/commands/keys
type Keys struct {
	redis.BaseCmd
	Pattern string
}

func ParseKeys(b redis.BaseCmd) (*Keys, error) {
	cmd := &Keys{BaseCmd: b}
	if len(cmd.Args()) != 1 {
		return cmd, redis.ErrInvalidArgNum
	}
	cmd.Pattern = string(cmd.Args()[0])
	return cmd, nil
}

func (cmd *Keys) Run(w redis.Writer, red redis.Redka) (any, error) {
	keys, err := red.Key().Keys(cmd.Pattern)
	if err != nil {
		w.WriteError(cmd.Error(err))
		return nil, err
	}
	w.WriteArray(len(keys))
	for _, key := range keys {
		w.WriteBulkString(key.Key)
	}
	return keys, nil
}
