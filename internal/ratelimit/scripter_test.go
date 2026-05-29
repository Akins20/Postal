package ratelimit

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
)

// deadScripter is a redis.Scripter whose every call fails, used to exercise the
// middleware's fail-open / fail-closed backend-error policy without a real
// Redis outage.
type deadScripter struct{}

var errDeadRedis = errors.New("redis unavailable")

func erroredCmd(ctx context.Context) *redis.Cmd {
	c := redis.NewCmd(ctx)
	c.SetErr(errDeadRedis)
	return c
}

func (deadScripter) Eval(ctx context.Context, _ string, _ []string, _ ...any) *redis.Cmd {
	return erroredCmd(ctx)
}

func (deadScripter) EvalSha(ctx context.Context, _ string, _ []string, _ ...any) *redis.Cmd {
	return erroredCmd(ctx)
}

func (deadScripter) EvalRO(ctx context.Context, _ string, _ []string, _ ...any) *redis.Cmd {
	return erroredCmd(ctx)
}

func (deadScripter) EvalShaRO(ctx context.Context, _ string, _ []string, _ ...any) *redis.Cmd {
	return erroredCmd(ctx)
}

func (deadScripter) ScriptExists(ctx context.Context, _ ...string) *redis.BoolSliceCmd {
	c := redis.NewBoolSliceCmd(ctx)
	c.SetErr(errDeadRedis)
	return c
}

func (deadScripter) ScriptLoad(ctx context.Context, _ string) *redis.StringCmd {
	c := redis.NewStringCmd(ctx)
	c.SetErr(errDeadRedis)
	return c
}
