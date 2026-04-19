# ratelimit

English | [日本語](README.ja.md)

IP-based rate limiting using token bucket algorithm.

## Public API

|Function / Type|Description|
|---|---|
|`IPRateLimiter`|Interface with `AllowRequest(c) bool` and `Cleanup()`|
|`NewIPRateLimiter(ipLimitCfg)`|Create in-memory per-IP rate limiter|
|`Middleware(rl, ipCfg)`|Return Echo middleware that returns 429 when limit exceeded (skips ops endpoints)|

## Limitation: Not Suitable for Horizontal Scaling

This implementation stores rate limit state **in-memory** (`sync.Mutex` + `map`). Each process maintains its own independent counter, so in a multi-instance deployment the effective limit becomes `configured_limit × number_of_instances`.

For horizontal scaling, replace with a shared store such as Redis (`INCR` + `EXPIRE`) or a distributed rate limiter. The `IPRateLimiter` interface allows swapping the implementation without changing middleware or handler code.
