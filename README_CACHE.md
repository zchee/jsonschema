# Schema Cache Usage (Draft)

## When to enable
- Repeated reflection of the *same types* in hot paths (e.g., service startup warming, codegen).
- You can tolerate modest memory overhead from cached schemas.

## How to enable
```go
r := &jsonschema.Reflector{
    EnableSchemaCache: true,
    MaxSchemaCacheEntries: 128, // optional cap to bound memory
}
s := r.ReflectFromType(reflect.TypeOf(MyType{}))
```

## Guidance
- Keep `EnableSchemaCache` off by default for general workloads.
- Set `MaxSchemaCacheEntries` to a small bound (e.g., 64–256) to avoid unbounded growth.
- Cache entries are cloned on load to prevent caller mutation from polluting the cache.
- Cache uses LRU-style promotion; oldest entries evicted when over the limit.

## Benchmarks (Apple M3 Max, gobench -count=20 -benchtime=5x)
- `ReflectRepeatedTypeCachedLimited`: 35.97µs → 17.24µs (-52%), 81.20KiB → 55.84KiB (-31%), 291 → 202 allocs/op (-31%).
- Effect is significant only when the same type is reflected repeatedly; wide structs with varied types see little to no gain.

## Caveats
- Memory still increases relative to cache-off due to stored clones; cap the cache if memory is tight.
- If hit rate is low, cache may hurt overall performance; measure on your workload.
