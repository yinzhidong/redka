package rstring

import (
	"database/sql"
	"time"

	"github.com/nalgeon/redka/internal/core"
	"github.com/nalgeon/redka/internal/sqlx"
)

const (
	sqlGet = `
	select value
	from rstring
	  join rkey on key_id = rkey.id and (etime is null or etime > ?)
	where key = ?`

	sqlGetMany = `
	select key, value
	from rstring
	  join rkey on key_id = rkey.id and (etime is null or etime > ?)
	where key in (:keys)`

	sqlSet = `
	insert into rkey (key, type, version, etime, mtime)
	values (?, ?, ?, ?, ?)
	on conflict (key) do update set
	  version = version+1,
	  type = excluded.type,
	  etime = excluded.etime,
	  mtime = excluded.mtime;

	insert into rstring (key_id, value)
	values ((select id from rkey where key = ?), ?)
	on conflict (key_id) do update
	set value = excluded.value;`

	sqlUpdate = `
	insert into rkey (key, type, version, etime, mtime)
	values (?, ?, ?, null, ?)
	on conflict (key) do update set
	  version = version+1,
	  type = excluded.type,
	  -- not changing etime
	  mtime = excluded.mtime;

	insert into rstring (key_id, value)
	values ((select id from rkey where key = ?), ?)
	on conflict (key_id) do update
	set value = excluded.value`
)

// Tx is a string repository transaction.
type Tx struct {
	tx sqlx.Tx
}

// NewTx creates a string repository transaction
// from a generic database transaction.
func NewTx(tx sqlx.Tx) *Tx {
	return &Tx{tx}
}

// Get returns the value of the key.
// If the key does not exist or is not a string, returns ErrNotFound.
func (tx *Tx) Get(key string) (core.Value, error) {
	return get(tx.tx, key)
}

// GetMany returns a map of values for given keys.
// Ignores keys that do not exist or not strings,
// and does not return them in the map.
func (tx *Tx) GetMany(keys ...string) (map[string]core.Value, error) {
	// Get the values of the requested keys.
	now := time.Now().UnixMilli()
	query, keyArgs := sqlx.ExpandIn(sqlGetMany, ":keys", keys)
	args := append([]any{now}, keyArgs...)

	var rows *sql.Rows
	rows, err := tx.tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Fill the map with the values for existing keys.
	items := map[string]core.Value{}
	for rows.Next() {
		var key string
		var val []byte
		err = rows.Scan(&key, &val)
		if err != nil {
			return nil, err
		}
		items[key] = core.Value(val)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil
}

// Incr increments the integer key value by the specified amount.
// Returns the value after the increment.
// If the key does not exist, sets it to 0 before the increment.
// If the key value is not an integer, returns ErrValueType.
// If the key exists but is not a string, returns ErrKeyType.
func (tx *Tx) Incr(key string, delta int) (int, error) {
	// get the current value
	val, err := tx.Get(key)
	if err != nil && err != core.ErrNotFound {
		return 0, err
	}

	// check if the value is a valid integer
	valInt, err := val.Int()
	if err != nil {
		return 0, core.ErrValueType
	}

	// increment the value
	newVal := valInt + delta
	err = update(tx.tx, key, newVal)
	if err != nil {
		return 0, err
	}

	return newVal, nil
}

// IncrFloat increments the float key value by the specified amount.
// Returns the value after the increment.
// If the key does not exist, sets it to 0 before the increment.
// If the key value is not an float, returns ErrValueType.
// If the key exists but is not a string, returns ErrKeyType.
func (tx *Tx) IncrFloat(key string, delta float64) (float64, error) {
	// get the current value
	val, err := tx.Get(key)
	if err != nil && err != core.ErrNotFound {
		return 0, err
	}

	// check if the value is a valid float
	valFloat, err := val.Float()
	if err != nil {
		return 0, core.ErrValueType
	}

	// increment the value
	newVal := valFloat + delta
	err = update(tx.tx, key, newVal)
	if err != nil {
		return 0, err
	}

	return newVal, nil
}

// Set sets the key value that will not expire.
// Overwrites the value if the key already exists.
// If the key exists but is not a string, returns ErrKeyType.
func (tx *Tx) Set(key string, value any) error {
	return tx.SetExpires(key, value, 0)
}

// SetExpires sets the key value with an optional expiration time (if ttl > 0).
// Overwrites the value and ttl if the key already exists.
// If the key exists but is not a string, returns ErrKeyType.
func (tx *Tx) SetExpires(key string, value any, ttl time.Duration) error {
	if !core.IsValueType(value) {
		return core.ErrValueType
	}
	var at time.Time
	if ttl > 0 {
		at = time.Now().Add(ttl)
	}
	err := set(tx.tx, key, value, at)
	return err
}

// SetMany sets the values of multiple keys.
// Overwrites values for keys that already exist and
// creates new keys/values for keys that do not exist.
// Removes the TTL for existing keys.
// If any of the keys exists but is not a string, returns ErrKeyType.
func (tx *Tx) SetMany(items map[string]any) error {
	for _, val := range items {
		if !core.IsValueType(val) {
			return core.ErrValueType
		}
	}

	at := time.Time{} // no expiration
	for key, val := range items {
		err := set(tx.tx, key, val, at)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetWith sets the key value with additional options.
func (tx *Tx) SetWith(key string, value any) SetCmd {
	return SetCmd{tx: tx, key: key, val: value}
}

func get(tx sqlx.Tx, key string) (core.Value, error) {
	args := []any{
		time.Now().UnixMilli(), // now
		key,                    // key
	}
	var val []byte
	err := tx.QueryRow(sqlGet, args...).Scan(&val)
	if err == sql.ErrNoRows {
		return core.Value(nil), core.ErrNotFound
	}
	if err != nil {
		return core.Value(nil), err
	}
	return core.Value(val), nil
}

// set sets the key value and (optionally) its expiration time.
func set(tx sqlx.Tx, key string, value any, at time.Time) error {
	var etime *int64
	if !at.IsZero() {
		etime = new(int64)
		*etime = at.UnixMilli()
	}

	args := []any{
		// insert into rkey
		key,                    // key
		core.TypeString,        // type
		core.InitialVersion,    // version
		etime,                  // etime
		time.Now().UnixMilli(), // mtime
		// insert into rstring
		key,   // key
		value, // value
	}
	_, err := tx.Exec(sqlSet, args...)
	if err != nil {
		return sqlx.TypedError(err)
	}
	return nil
}

// update updates the value of the existing key without changing its
// expiration time. If the key does not exist, creates a new key with
// the specified value and no expiration time.
func update(tx sqlx.Tx, key string, value any) error {
	args := []any{
		// insert into rkey
		key,                    // key
		core.TypeString,        // type
		core.InitialVersion,    // version
		time.Now().UnixMilli(), // mtime
		// insert into rstring
		key,   // key
		value, // value
	}
	_, err := tx.Exec(sqlUpdate, args...)
	if err != nil {
		return sqlx.TypedError(err)
	}
	return nil
}
