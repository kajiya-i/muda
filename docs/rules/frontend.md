# Frontend Development Rules (TypeScript + Effect)

## Functional Programming Principles

The frontend follows functional programming principles and uses **Effect** as its standard library for side effects, typed errors, dependency management, and data validation.

Effect is used where it adds guarantees that plain TypeScript cannot: typed failure, typed dependencies, controlled execution of side effects, and schema-based parsing. It is **not** used to wrap code that is already pure. The guiding split is:

| Kind of code | How to write it |
|---|---|
| Pure computation (domain rules, data transformation, reducers, derived values) | Plain TypeScript functions |
| Anything that performs I/O, can fail in an expected way, or needs a dependency | `Effect` |
| Untrusted data entering the application | `Schema` |
| Wiring dependencies together | `Layer`, at a single composition root |
| Running effects | Only at the application boundary, via a `ManagedRuntime` |

These rules are guidelines, not an instruction to make every piece of code maximally functional. Prefer the form that is clearest for a reader who knows TypeScript and the core of Effect.

### Effect Version

These rules target **Effect 4.x**. All code examples use the Effect 4 API.

* [Effect 4.0 release announcement](https://effect.website/blog/releases/effect/40)
* [Migration guide from Effect 3 to Effect 4](https://github.com/Effect-TS/effect/blob/main/MIGRATION.md)
* [Effect 4 documentation](https://effect.website/docs/v4/getting-started/)

Most existing articles, answers, and examples on the web still use the Effect 3 API. Do not copy Effect 3 code as-is. The differences that come up most often in this codebase are:

| Effect 3 | Effect 4 |
|---|---|
| `Context.Tag("Key")<Self, Shape>()` | `Context.Service<Self, Shape>()("Key")` |
| `Effect.async` | `Effect.callback` |
| `Effect.catchAll` | `Effect.catch` |
| `Schema.decodeUnknown` | `Schema.decodeUnknownEffect` |
| `ParseError` (tag `"ParseError"`) | `Schema.SchemaError` (tag `"SchemaError"`) |
| `Schema.pattern(regex)` | `.check(Schema.isPattern(regex))` |
| `Option.fromNullable` | `Option.fromUndefinedOr` / `Option.fromNullishOr` |
| `Data.struct` for structural equality | Not needed; `Equal.equals` compares plain objects and arrays structurally |
| `TestClock` from `effect` | `TestClock` from `effect/testing` |
| `@effect/platform` (`HttpClient`, `FetchHttpClient`) | `effect/http` |
| `STM`, `TRef` | `Effect.tx`, `TxRef` and other `Tx*` types |

When a review comment or a reference suggests an Effect 3 API, check it against the Effect 4 documentation before applying it.

---

## 1. Prefer Pure Functions

Prefer functions that depend only on their arguments, return their result without modifying external state, and perform no I/O.

```ts
// Good
const applyDiscount = (price: number, rate: number): number => price * (1 - rate)

// Avoid
let total = 0
const addToTotal = (value: number): void => {
  total += value
}
```

A pure function does not need Effect. Do not return `Effect.succeed(...)` from a function that cannot fail and has no dependencies; return the plain value.

```ts
// Avoid: wrapping pure computation in Effect adds noise and nothing else
const applyDiscount = (price: number, rate: number) =>
  Effect.succeed(price * (1 - rate))
```

---

## 2. Avoid Hidden Inputs

Purity is broken not only by hidden outputs (side effects) but also by hidden inputs: values not in the argument list that can change between calls, such as `Date.now()`, `Math.random()`, module-level mutable variables, and stale closures.

In **pure code**, pass the non-deterministic value as an argument:

```ts
// Avoid
const isExpired = (expiresAt: number): boolean => expiresAt < Date.now()

// Good
const isExpired = (expiresAt: number, now: number): boolean => expiresAt < now
```

In **effectful code**, read it from Effect's services instead of the global API, so tests can control it:

```ts
const checkSession = (session: Session) =>
  Effect.gen(function* () {
    const now = yield* Clock.currentTimeMillis
    return isExpired(session.expiresAt, now) // pure function does the deciding
  })
```

Use `Clock` for time and `Random` for randomness inside effects, and `TestClock` (from `effect/testing`) in tests. Do not call `Date.now()` or `Math.random()` inside an `Effect.gen` body.

The same applies to closures in React. A handler that reads a value captured from an earlier render can act on stale data:

```ts
// Avoid
const handleClick = () => setCount(count + 1)

// Good
const handleClick = () => setCount((current) => current + 1)
```

---

## 3. Do Not Mutate Input Data

Do not modify objects or arrays received as arguments. Return a new value instead.

```ts
// Good
const addUser = (users: ReadonlyArray<User>, user: User): ReadonlyArray<User> => [...users, user]

// Avoid
const addUser = (users: User[], user: User): User[] => {
  users.push(user)
  return users
}
```

Use `readonly` properties and `ReadonlyArray` for parameters and data structures that are not meant to be mutated. Schemas (Section 10) produce readonly types by default; keep them readonly.

---

## 4. Prefer Immutable State

Treat application state as immutable. Update state by creating a new value.

```ts
// Good
const updated = { ...user, name: "Alice" }

// Avoid
user.name = "Alice"
```

Immutability is especially important for React state, because React detects changes by reference.

---

## 5. Value Equality vs. Reference Equality

An immutable update always produces a new reference, even when the content is unchanged. React's `useEffect`, `useMemo`, `useCallback`, and `React.memo` compare by reference (`Object.is`), so creating a new reference on every render causes unnecessary work.

```ts
// Avoid: a new array on every render re-runs the effect every time
const filters = ids.filter((id) => id !== "")
useEffect(() => {
  load(filters)
}, [filters])

// Good: stable reference while the input is unchanged
const filters = useMemo(() => ids.filter((id) => id !== ""), [ids])
useEffect(() => {
  load(filters)
}, [filters])
```

When the question is "are these two values meaningfully equal?" rather than "should this hook re-run?", compare structurally with `Equal.equals(a, b)`. In Effect 4 it compares primitives, arrays, plain objects, maps, sets, and dates by content, and delegates to values that implement the `Equal` interface (such as `Data.TaggedClass`, `Data.TaggedEnum` values, and `Schema.Class` instances). Note that React itself still compares by reference; `Equal` does not change hook behavior.

---

## 6. Represent State Explicitly

Prefer discriminated unions over loosely related boolean flags and nullable fields when states are mutually exclusive. Use `_tag` as the discriminant throughout the codebase so the same values work with `Match`, `Effect.catchTag`, and Effect's data types.

```ts
type RequestState = Data.TaggedEnum<{
  Idle: {}
  Loading: {}
  Success: { readonly users: ReadonlyArray<User> }
  Failure: { readonly error: LoadUsersError }
}>

const RequestState = Data.taggedEnum<RequestState>()

const initial = RequestState.Idle()
```

Prefer this over:

```ts
type RequestState = {
  loading: boolean
  error: Error | null
  data: User[] | null
}
```

The type should make invalid states impossible to represent. A plain TypeScript union with a `_tag` field is also acceptable; `Data.TaggedEnum` adds constructors and structural equality.

---

## 7. Ensure Exhaustiveness

Make an unhandled case a compile-time error. Use `Match` with `Match.exhaustive`, or the tagged enum's `$match`:

```ts
const describe = (state: RequestState): string =>
  Match.value(state).pipe(
    Match.tag("Idle", () => "Idle"),
    Match.tag("Loading", () => "Loading"),
    Match.tag("Success", ({ users }) => `Loaded ${users.length} users`),
    Match.tag("Failure", ({ error }) => describeError(error)),
    Match.exhaustive,
  )
```

A plain `switch` on `_tag` with a `never` check in the `default` branch is equally acceptable in pure code. Either way, adding a variant without handling it must fail to compile. Do not use `Match.orElse` or a catch-all `default` to silence a missing case.

---

## 8. Use Functions for State Transitions

Represent state transitions governed by application rules as pure functions, independent of rendering and side effects.

```ts
type Action = Data.TaggedEnum<{
  Increment: {}
  Decrement: {}
}>

const reducer = (state: State, action: Action): State =>
  Match.value(action).pipe(
    Match.tag("Increment", () => ({ ...state, count: state.count + 1 })),
    Match.tag("Decrement", () => ({ ...state, count: state.count - 1 })),
    Match.exhaustive,
  )
```

Reducers stay pure. Never run an effect, call an API, or read the clock inside a reducer.

---

## 9. Model Side Effects as `Effect`

Any operation that performs I/O — HTTP, storage, timers, WebSocket, logging, analytics, browser APIs — is expressed as an `Effect<A, E, R>`:

* `A` is the success value.
* `E` is the union of **expected** errors the caller must handle.
* `R` is the set of services the operation requires.

Write sequential effectful code with `Effect.gen`, and keep the decisions inside it delegated to pure functions:

```ts
const loadActiveUsers = Effect.gen(function* () {
  const api = yield* UserApi
  const users = yield* api.list()
  return getActiveUsers(users) // pure
})
// Effect<ReadonlyArray<User>, NetworkError | DecodeError, UserApi>
```

Do not perform side effects outside of an `Effect` (no bare `fetch`, `localStorage`, or `console` calls in application code), and do not hide them inside functions that look pure. Wrap Promise-based or callback-based APIs once, at the integration point, with `Effect.tryPromise` or `Effect.callback`, mapping their failures into typed errors.

```ts
const fetchJson = (url: string) =>
  Effect.tryPromise({
    try: (signal) => fetch(url, { signal }).then((res) => res.json() as Promise<unknown>),
    catch: (cause) => new NetworkError({ cause }),
  })
```

Pass the provided `AbortSignal` through so that interrupting the effect cancels the request.

---

## 10. Parse, Don't Validate — With `Schema`

At every boundary where data enters the application (API responses, form input, URL parameters, storage, `postMessage`), decode it once with a `Schema` into a typed value. Do not pass `unknown`, `any`, or loosely typed objects deeper into the application, and do not use type assertions (`as User`) on external data.

```ts
const UserId = Schema.String.pipe(Schema.brand("UserId"))
type UserId = typeof UserId.Type

const Email = Schema.String.check(Schema.isPattern(/^[^@\s]+@[^@\s]+$/)).pipe(Schema.brand("Email"))

class User extends Schema.Class<User>("User")({
  id: UserId,
  name: Schema.NonEmptyString,
  email: Email,
  active: Schema.Boolean,
}) {}

const decodeUsers = Schema.decodeUnknownEffect(Schema.Array(User))

const listUsers = fetchJson("/api/users").pipe(
  Effect.flatMap(decodeUsers),
  Effect.catchTag("SchemaError", (error) => Effect.fail(new DecodeError({ error }))),
)
```

Use branded schemas (`Schema.brand`) for domain concepts with real invariants (IDs, email addresses, amounts) so a raw `string` cannot be passed where a validated value is expected. Do not brand every field merely for consistency.

Keep the schema as the single definition of a shape: derive the TypeScript type from it (`typeof User.Type`, `typeof X.Type`) instead of declaring the type separately.

---

## 11. Define Expected Errors as Tagged Errors

Model every expected failure as a tagged error class, and let it flow through the `E` channel.

```ts
class UserNotFound extends Data.TaggedError("UserNotFound")<{
  readonly id: UserId
}> {}

class NetworkError extends Data.TaggedError("NetworkError")<{
  readonly cause: unknown
}> {}
```

Fail with them by yielding them (tagged errors are yieldable) or with `Effect.fail`, and handle them by tag:

```ts
const findUser = (id: UserId) =>
  Effect.gen(function* () {
    const user = yield* lookup(id)
    if (user === undefined) {
      return yield* new UserNotFound({ id })
    }
    return user
  })

const program = findUser(id).pipe(
  Effect.catchTag("UserNotFound", () => Effect.succeed(guestUser)),
)
```

Rules:

* Do not `throw` inside effectful code. A thrown exception becomes a **defect**, which is invisible in the type.
* Distinguish expected errors (in `E`, handled by callers) from defects (bugs and unrecoverable conditions). Use `Effect.orDie` or `Effect.die` only for conditions the caller genuinely cannot handle; do not catch defects in business logic.
* Keep error types specific. Do not collapse everything into `Error` or `unknown` in the `E` channel.
* Prefer handling errors by tag (`Effect.catchTag`, `Effect.catchTags`) over catching everything (`Effect.catch`), so that a newly added error type is not silently swallowed.
* Map backend error codes from the API contract to tagged errors at the API client boundary, once (see Section 18).

In pure code that does not otherwise use Effect, a plain discriminated union return type is acceptable for multiple outcomes; do not pull in Effect just to return an error.

---

## 12. Manage Dependencies With Services and Layers

Express what an effect needs as a service in `R`, never as a module-level singleton imported directly.

```ts
class UserApi extends Context.Service<
  UserApi,
  {
    readonly list: () => Effect.Effect<ReadonlyArray<User>, NetworkError | DecodeError>
    readonly findById: (id: UserId) => Effect.Effect<User, UserNotFound | NetworkError | DecodeError>
  }
>()("UserApi") {}
```

Provide implementations as `Layer`s, and compose all layers in **one** composition root:

```ts
import { FetchHttpClient, HttpClient } from "effect/http"

const UserApiLive = Layer.effect(
  UserApi,
  Effect.gen(function* () {
    const http = yield* HttpClient.HttpClient
    return {
      list: () => /* ... */,
      findById: (id) => /* ... */,
    }
  }),
)

export const AppLayer = Layer.mergeAll(UserApiLive /* , ... */).pipe(
  Layer.provide(FetchHttpClient.layer),
)
```

Rules:

* A service interface's methods must not themselves leak requirements (`R` of each method should be `never`); dependencies are resolved when the layer is built.
* Do not call `Effect.provide` deep inside business logic. Provide layers at the composition root or in tests.
* In tests, provide a test layer (`Layer.succeed(UserApi, { ... })`) instead of mocking modules.

---

## 13. Run Effects Only at the Boundary

Create a single `ManagedRuntime` from the application layer and run effects only at the edges: event handlers, data-fetching hooks or query functions, route loaders, and application startup.

```ts
export const runtime = ManagedRuntime.make(AppLayer)

// e.g. inside a data-fetching function used by the UI layer
const queryFn = ({ signal }: { signal: AbortSignal }) =>
  runtime.runPromise(loadActiveUsers, { signal })
```

Rules:

* Never run effects during render. Components stay pure functions of props and state.
* Do not call `Effect.runPromise` / `Effect.runSync` scattered across the codebase; go through the shared runtime so every effect sees the same services.
* Forward cancellation (`AbortSignal`, component unmount) to the runtime so effects are interrupted rather than left running.
* Convert typed failures into UI state at the boundary (e.g. into a `Failure` variant of `RequestState`), handling every error tag exhaustively.
* Dispose the runtime when the application shuts down if it owns resources.

---

## 14. Prefer Data Transformation Over Mutation

Use `map`, `filter`, `find`, `some`, `every`, and `reduce` when they make transformation clearer:

```ts
const activeUserNames = users.filter((user) => user.active).map((user) => user.name)
```

In pure code, prefer native array methods. Effect's `Array`, `Record`, and `Option` modules are fine inside `pipe` chains where they read better, but do not convert plain code to them for its own sake. Do not use functional constructs merely to avoid a loop; prefer the form that most clearly communicates the operation.

---

## 15. Prefer Iteration Over Deep Recursion

JavaScript engines do not guarantee tail-call optimization. Use array methods or loops for collections, and reserve recursion for depth-bounded recursive structures.

```ts
// Avoid
const sumAll = (values: ReadonlyArray<number>): number =>
  values.length === 0 ? 0 : values[0]! + sumAll(values.slice(1))

// Good
const sumAll = (values: ReadonlyArray<number>): number =>
  values.reduce((total, value) => total + value, 0)
```

For effectful iteration, use `Effect.forEach` or `Effect.all` rather than recursively chaining effects.

---

## 16. Prefer Small, Composable Functions and Compose Them Explicitly

Give each function one clear responsibility, and compose them with Effect's `pipe` (or the `.pipe` method on Effect data types). Do not write a local `pipe`/`compose` helper and do not add another composition library.

```ts
const activeUserNames = pipe(users, getActiveUsers, getUserNames)

const program = loadActiveUsers.pipe(
  Effect.timeout("5 seconds"),
  Effect.retry({ schedule: Schedule.exponential("200 millis"), times: 3 }),
  Effect.withSpan("loadActiveUsers"),
)
```

Do not split code into trivial one-line functions without a meaningful abstraction, and do not build long point-free chains when a few named intermediate values are clearer. Inside `Effect.gen`, ordinary sequential statements are usually the most readable form.

---

## 17. Prefer Total Functions and Be Consistent About Absence

Prefer functions defined for every value of their declared input type, rather than functions that throw for "valid-looking" input.

```ts
// Avoid
const firstChar = (value: string): string => {
  if (value.length === 0) throw new Error("empty string")
  return value[0]!
}

// Good
const firstChar = (value: string): string | undefined => value[0]
```

For representing absence:

* In plain TypeScript code and at React/component boundaries, use `T | undefined`.
* Inside Effect pipelines where absence is composed with other operations, `Option<T>` is acceptable.
* Convert between them at the edge of the module (`Option.fromUndefinedOr`, `Option.getOrUndefined`). Do not expose both styles for the same concept, and do not use `null` and `undefined` interchangeably.

Enable `noUncheckedIndexedAccess` so array and record lookups are typed as possibly absent.

---

## 18. API Contract With the Backend

The backend is written in another language, so types cannot be shared directly. Treat the language-neutral contract (e.g. an OpenAPI document) as the source of truth.

* Define (or generate) a `Schema` for every request and response in the contract, and decode every response with it. The backend is an external system: do not trust its payloads without decoding.
* Keep transport schemas separate from domain types when they differ, and transform between them in the API client.
* Map the backend's machine-readable error codes to tagged errors in the API client, once. Unknown codes become a dedicated tagged error (e.g. `UnexpectedApiError`), not a silent fallback.
* A change in the contract must break compilation or decoding, never silently produce `undefined` at runtime.

---

## 19. Keep Side Effects at Application Boundaries

```text
UI / Event
    │
    ▼
Boundary (ManagedRuntime.runPromise, hooks, loaders)
    │
    ▼
Application Logic (Effect: orchestration, typed errors, services in R)
    │
    ├── Pure Functions (plain TypeScript)
    ├── State Transitions (reducers)
    └── Data Transformation
    │
    ▼
Services (provided by Layers)
    ├── HTTP client
    ├── Browser APIs / Storage
    └── External services
```

The core logic contains no direct side effects: it either computes purely or describes effects that the boundary runs.

---

## 20. Do Not Over-Apply Functional Programming

Effect is a tool, not a goal. Do not:

* Wrap pure computation in `Effect`, or make a function return `Effect` when it cannot fail and has no dependencies.
* Convert simple code to `Option`, `Result`, or Effect's collection modules where plain TypeScript is clearer.
* Introduce custom monadic abstractions on top of Effect.
* Use advanced features (fibers, `Stream`, transactional `Tx*` types, `Scope` manipulation, custom `Schedule`s) without a concrete requirement.
* Build deeply nested `pipe` chains or point-free code that a teammate cannot read without tracing every combinator.
* Push Effect types into React component props; components receive plain values and callbacks.

Prefer the simplest design that keeps state, errors, and side effects explicit.

---

## 21. Library Policy

Effect is the only functional-programming library in the frontend.

* Do not add `fp-ts`, `io-ts`, `purify-ts`, `neverthrow`, `Remeda`, `Ramda`, or similar libraries alongside it.
* Use `Schema` for validation instead of a second validation library, unless a dependency requires one at its own boundary (convert to `Schema`-decoded types immediately).
* Use the modules built into `effect` (e.g. `effect/http`) for HTTP and platform integration where they fit, rather than mixing several independent abstractions.
* Keep `effect` and every `@effect/*` package on the same version, and upgrade them together. Effect 4 releases all of them in lockstep under a single version number.

### Unstable Modules

Some Effect 4 modules (e.g. `effect/http`) are marked `@stability unstable` and may introduce breaking changes in minor releases. They may be used, but:

* Use them only inside service implementations (`Layer`s) at the integration points, never in domain logic or in service interfaces. A breaking change then stays contained in one place.
* Pin the Effect version exactly, and read the release notes before upgrading.

Any additional library must solve an identified problem that Effect and standard TypeScript cannot solve cleanly, and must be readable and maintainable by the team.

---

## Summary

* Pure computation is plain TypeScript; I/O, expected failure, and dependencies are `Effect`.
* These rules target Effect 4; do not copy Effect 3 code without checking it against the Effect 4 API.
* No hidden inputs: pass values explicitly in pure code, use `Clock` and `Random` in effects.
* Do not mutate inputs or state; use `readonly`.
* Compare by value with `Equal.equals` when meaning matters; remember React compares by reference.
* Use `_tag`-discriminated unions for state, and make unhandled cases a compile error with `Match.exhaustive`.
* Keep reducers pure.
* Decode all external data once with `Schema`; brand concepts with real invariants.
* Model expected errors as tagged errors in `E`; never `throw` inside effects; keep defects separate.
* Declare dependencies as services in `R` with `Context.Service`, provide them with `Layer`s at one composition root.
* Run effects only at the boundary through a single `ManagedRuntime`, never during render.
* Compose with Effect's `pipe`; avoid over-abstraction.
* Treat the backend API contract as the source of truth and map its error codes to tagged errors once.
* Effect is the only FP library; keep unstable Effect modules confined to service implementations.