# Architecture Note: Global Reactive Store (`pkg/store`)

`pkg/store` provides generic, thread-safe global state management for GoKS applications (especially client-side WASM islands, but also server-side singleton state).

## Key Guarantees & Semantics

1. **Typed State**:
   `Store[T]` wraps an arbitrary type `T`. State reads via `Get()` return a value copy of `T` guarded by `sync.RWMutex.RLock`.

2. **Re-entrancy Safety**:
   When `Set(newState)` or `Update(fn)` executes:
   - The store acquires an exclusive lock `mu.Lock()`.
   - The state is modified.
   - The list of active subscriber callbacks is snapshot into a local slice.
   - The store unlocks `mu.Unlock()` **before** invoking the listener callbacks.
   - Consequently, subscribers can safely call `Get()`, `Set()`, `Update()`, or `Subscribe()` / `unsubscribe()` without causing recursive mutex deadlocks.

3. **Synchronous Notification**:
   Listeners are invoked synchronously in registration order on the goroutine triggering the mutation. There is no implicit asynchronous batching or event-loop deferral inside `pkg/store` itself; subscribers that trigger UI re-renders schedule virtual DOM reconciliation via the component fiber's `AppRerender` callback.

4. **Thread Safety & Race Freedom**:
   All public methods (`Get`, `Set`, `Update`, `Subscribe`, and returned unsubscribe functions) are thread-safe and verified with Go's race detector (`go test -race`).
