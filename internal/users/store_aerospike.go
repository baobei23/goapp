package users

import (
	"context"
	"fmt"
	"time"

	as "github.com/aerospike/aerospike-client-go/v8"
)

// asstore is an Aerospike-backed refresh-token store. Refresh tokens are a
// natural fit for Aerospike: key-value with a native record TTL, so expired
// tokens drop themselves with no cleanup job.
//
// It implements only the refresh-token subset of the users store interface
// (SaveRefreshToken/CheckRefreshToken/RevokeRefreshToken). Wire it into the
// service where you want refresh tokens offloaded from Postgres; user CRUD
// stays on the Postgres store.
type asstore struct {
	client    *as.Client
	namespace string
	set       string

	// The Go client takes no context, so ctx cannot carry cancellation into a
	// call. TotalTimeout is the only bound available -- without it a hung node
	// stalls the auth handler indefinitely.
	readPolicy  *as.BasePolicy
	writePolicy *as.WritePolicy
}

const refreshTokenBin = "user_id"

func (ps *asstore) key(jti string) (*as.Key, error) {
	key, err := as.NewKey(ps.namespace, ps.set, jti)
	if err != nil {
		return nil, fmt.Errorf("failed building aerospike key: %w", err)
	}
	return key, nil
}

func (ps *asstore) SaveRefreshToken(ctx context.Context, jti, userID string, expiresAt time.Time) error {
	key, err := ps.key(jti)
	if err != nil {
		return err
	}

	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return fmt.Errorf("refresh token already expired")
	}

	policy := *ps.writePolicy
	policy.Expiration = uint32(ttl.Seconds())
	if err := ps.client.Put(&policy, key, as.BinMap{refreshTokenBin: userID}); err != nil {
		return fmt.Errorf("failed saving refresh token: %w", err)
	}
	return nil
}

func (ps *asstore) CheckRefreshToken(ctx context.Context, jti string) (bool, error) {
	key, err := ps.key(jti)
	if err != nil {
		return false, err
	}

	exists, err := ps.client.Exists(ps.readPolicy, key)
	if err != nil {
		return false, fmt.Errorf("failed checking refresh token: %w", err)
	}
	return exists, nil
}

func (ps *asstore) RevokeRefreshToken(ctx context.Context, jti string) error {
	key, err := ps.key(jti)
	if err != nil {
		return err
	}

	if _, err := ps.client.Delete(ps.writePolicy, key); err != nil {
		return fmt.Errorf("failed revoking refresh token: %w", err)
	}
	return nil
}

func NewAerospikeStore(client *as.Client, namespace, set string) *asstore {
	readPolicy := as.NewPolicy()
	readPolicy.TotalTimeout = QueryTimeoutDuration

	writePolicy := as.NewWritePolicy(0, 0)
	writePolicy.TotalTimeout = QueryTimeoutDuration

	return &asstore{
		client:      client,
		namespace:   namespace,
		set:         set,
		readPolicy:  readPolicy,
		writePolicy: writePolicy,
	}
}
