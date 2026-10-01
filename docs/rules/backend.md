# Backend Development Rules (Go)

## Functional Programming Principles

The Go backend adopts the **ideas** of functional programming — pure business logic, explicit inputs, side effects at the boundary, immutable domain values — but **not** a functional-programming library or a functional coding style that fights Go's own idioms.

This is a deliberate asymmetry with the frontend. The frontend uses Effect, because TypeScript's type system can carry typed errors, dependencies, and discriminated unions without friction. Go lacks sum types, higher-kinded types, generic methods, and `readonly`, and its ecosystem is built around `(value, error)` and explicit loops. Each principle below is kept only where it survives translation into idiomatic Go; principles that would fight Go's grain are explicitly marked **not adopted**.

Do not introduce an FP library (e.g. `fp-go`, `samber/lo`, `samber/mo`, or a custom `Option`/`Result`/`Either` package) to implement these principles. Use the standard library and the language's own `(value, error)` idiom.

These are guidelines, not an instruction to make every piece of code maximally functional. Linters enforce what they can; the rest is judgment.

### Relationship to Layered Architecture and DDD

Layering (handler / usecase / domain / infrastructure) and DDD tactical patterns (value object, entity, aggregate, domain event, repository) decide **where** a piece of logic lives and **what type** represents it. The principles in this document decide **how** that logic is written once placed: pure, with explicit inputs and outputs, without mutating domain values. The two are orthogonal and reinforce each other. In particular, a domain event is a fact that has already happened and must never be mutated — DDD and FP converge on the same answer.

---

## 1. Prefer Pure Functions for Business Logic

Business rules, calculations, and validation should be functions or methods that depend only on their inputs and return a result, without touching the database, the clock, the environment, or other external state.

Place these rules on the domain types that own them (value objects, entities, aggregates), not inline in the usecase or handler layer.

```go
// Good: depends only on the receiver's fields
func (i Invoice) TotalInclTax() Money {
	return i.Subtotal.Add(i.Tax)
}

// Avoid: the usecase reimplements a business rule that belongs to Invoice
func (s *InvoiceService) Confirm(ctx context.Context, id InvoiceID) error {
	inv, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	total := inv.Subtotal.Amount() + inv.Tax.Amount() // business rule inlined here
	// ...
}
```

---

## 2. Avoid Hidden Inputs

A function that reads `time.Now()`, `rand`, `os.Getenv`, or a package-level mutable variable has an input that does not appear in its signature. Two calls with identical arguments can then return different results, and the function cannot be tested without manipulating global state.

```go
// Avoid: hidden input, cannot be tested without controlling the system clock
func (s Subscription) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Good: the non-deterministic value is an explicit argument
func (s Subscription) IsExpired(now time.Time) bool {
	return now.After(s.ExpiresAt)
}
```

Read non-deterministic values once, at the boundary (usecase or handler), and pass them down. When the usecase itself needs the current time, inject a clock rather than calling `time.Now()` directly:

```go
type Clock interface {
	Now() time.Time
}

type SubscriptionService struct {
	repo  SubscriptionRepository
	clock Clock
}
```

Read configuration (environment variables, flags) in a single configuration package at startup and pass typed values into the rest of the application. If the project uses `golangci-lint`, `forbidigo` can mechanically ban `os.Getenv`, `os.LookupEnv`, and `time.Now` outside the packages allowed to use them.

---

## 3. Keep Side Effects at the Boundary

HTTP handling, database access, messaging, filesystem access, and external API calls belong in the handler and infrastructure layers. Domain logic should not perform I/O.

```text
HTTP Request
    │
    ▼
Handler (I/O: decode request, encode response)
    │
    ▼
Usecase (orchestration: load, call domain, persist)
    │
    ├── Domain (pure business rules, value objects, aggregates)
    │
    ▼
Repository / Infrastructure (I/O: DB, external APIs, filesystem)
```

A usecase typically follows the shape "load → decide (pure) → persist". Keep the "decide" step a pure call into the domain so it can be tested without any infrastructure.

```go
func (s *SubscriptionService) Renew(ctx context.Context, id SubscriptionID) error {
	sub, err := s.repo.FindByID(ctx, id) // I/O
	if err != nil {
		return err
	}
	renewed, err := sub.Renew(s.clock.Now()) // pure domain decision
	if err != nil {
		return err
	}
	return s.repo.Save(ctx, renewed) // I/O
}
```

Enforce the direction of dependencies mechanically where possible (e.g. `depguard` rules that forbid the domain package from importing infrastructure, database, or HTTP packages).

---

## 4. Use Error Values for Expected Failure

Go already represents expected failure explicitly: as a second return value, not an exception. Do not introduce a `Result[T]` or `Either[E, A]` type to replicate what `(T, error)` already provides.

```go
var ErrInvalidInput = errors.New("invalid input")

func NewEmail(raw string) (Email, error) {
	if !strings.Contains(raw, "@") {
		return Email{}, fmt.Errorf("email %q: %w", raw, ErrInvalidInput)
	}
	return Email{value: raw}, nil
}
```

Make domain errors identifiable with sentinel errors or typed errors, wrap with `%w` to keep context, and inspect them with `errors.Is` / `errors.As` — never by comparing error strings.

Reserve `panic` for programmer errors and truly unrecoverable conditions, not for validation failures a caller is expected to handle.

---

## 5. Prefer Immutable Value Objects in the Domain Layer

Go has no `readonly`, and mutation is the normal idiom for builders, ORM entities, and request-binding structs. Do not fight that in infrastructure code. Scope this rule to **domain value objects and aggregates**: a method that changes a domain value should return a new value rather than mutating the receiver.

The standard library models this for `time.Time`: `t.Add(d)` returns a new `Time`. Follow the same shape.

```go
// Good: value receiver, returns a new value
func (s Subscription) Renew(now time.Time) (Subscription, error) {
	if s.Status == StatusCanceled {
		return Subscription{}, ErrCannotRenewCanceled
	}
	s.ExpiresAt = now.AddDate(0, 1, 0) // modifies the copy, not the caller's value
	return s, nil
}

// Avoid: pointer receiver mutates the caller's value in place
func (s *Subscription) Renew(now time.Time) {
	s.ExpiresAt = now.AddDate(0, 1, 0)
}
```

Keep value object fields unexported and expose them through constructors and accessors, so that an invalid or partially mutated value cannot be built from outside the package.

Be aware that a value receiver copies only the struct, not the data behind slices, maps, or pointers inside it. When a domain value holds a slice or map, copy it before modifying (`slices.Clone`, `maps.Clone`) and do not return internal slices or maps directly from accessors.

This rule does not apply to ORM-generated builders, query builders, or request DTOs populated by a framework's binding — those are infrastructure-layer mutation by design.

---

## 6. Ensure Exhaustiveness for Enums and Sealed Interfaces

Go's `switch` compiles even when a case is missing, so adding a new enum value can silently fall through every existing switch. Close this gap with linters rather than manual audits. Two constructs need two different linters.

**Enum-like constants** (a named type with `const` values): enable the `exhaustive` linter. A newly added constant that is not handled in some switch becomes a lint failure.

```go
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusCanceled  OrderStatus = "canceled"
)

// With `exhaustive` enabled, adding a new OrderStatus and forgetting
// a case here fails the lint step instead of shipping silently.
switch status {
case OrderStatusPending:
	// ...
case OrderStatusPaid:
	// ...
case OrderStatusCanceled:
	// ...
}
```

**Sealed interfaces** (Section 9): `exhaustive` does not check interface type switches. Use `gochecksumtype` (bundled in `golangci-lint` since v1.55.0). Mark the interface with a `//sumtype:decl` comment; every type switch on it must then cover all implementing types or have a `default` case.

```go
//sumtype:decl
type PaymentOutcome interface {
	isPaymentOutcome()
}

switch v := outcome.(type) {
case PaymentSucceeded:
	// ...
case PaymentFailed:
	// ...
}
```

Avoid a `default` case that silently swallows unknown values; it defeats both linters. If a `default` is needed, make it return an error or fail loudly.

---

## 7. Parse Into Validated Domain Types at the Boundary

Convert untrusted input (HTTP request bodies, query parameters, CSV rows, messages from a queue) into a validated domain type once, at the handler or usecase boundary. Do not pass raw `string` / `*string` / `map[string]any` deeper into the codebase and re-check it at each use site.

```go
// Handler: decode the transport DTO, then parse it into domain types once
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	email, err := domain.NewEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	// From here on, the rest of the call chain trusts `email`.
	// ...
}
```

Apply this to domain concepts with real invariants (format, length limits, allowed values, cross-field rules). Do not wrap every incidental string field in a dedicated type merely for consistency (see Section 10).

---

## 8. Not Adopted: Prefer `map` / `filter` / `reduce` Over Loops

The frontend may use `map`/`filter`/`reduce` for data transformation. **Do not bring this to Go.** Idiomatic Go favors an explicit `for` loop: it is clearer, easier to debug and profile, and avoids closure allocations. The standard `slices` and `maps` packages cover sorting, searching, cloning, and similar utilities, and should be used where they fit; general-purpose `Map`/`Filter`/`Reduce` helpers should not be added.

```go
// Idiomatic Go — prefer this
active := make([]User, 0, len(users))
for _, u := range users {
	if u.Active {
		active = append(active, u)
	}
}
```

Do not add `samber/lo`, `fp-go`, or a hand-rolled generic helper package to chase a frontend convention.

---

## 9. Use Sealed Interfaces Sparingly for Genuinely Exclusive States

Go has no sum types. A sealed interface (an interface with an unexported marker method, implemented only by types in the same package) can simulate one, but it is heavier than a TypeScript discriminated union: more boilerplate, and a separate linter for exhaustiveness (Section 6).

Reserve this pattern for domain states that are genuinely mutually exclusive, carry different data per state, and where treating them as independent fields or flags would cause costly bugs.

```go
//sumtype:decl
type PaymentOutcome interface {
	isPaymentOutcome()
}

type PaymentSucceeded struct{ ConfirmedAt time.Time }
type PaymentFailed struct{ Reason string }

func (PaymentSucceeded) isPaymentOutcome() {}
func (PaymentFailed) isPaymentOutcome()    {}
```

For most enum-like values, a plain named type with constants and an `exhaustive`-checked `switch` is the right amount of ceremony.

---

## 10. Do Not Over-Apply Functional Programming

Functional programming is a design tool, not a goal. In Go specifically, do not:

* Introduce a generic `Option[T]` / `Result[T]` / `Either[L, R]` to replace `(T, bool)` or `(T, error)`.
* Chain higher-order functions or closures where a plain loop or a sequence of statements is clearer.
* Wrap every struct field in a single-purpose value-object type when the field has no real invariant.
* Reach for a sealed interface where a plain enum and an `exhaustive`-checked switch would do.
* Pursue functional purity at the expense of the project's layering, conventions, and readability for Go developers.

Prefer the simplest design consistent with the project's existing architecture.

---

## 11. Library Policy

Do not add a functional-programming library to the Go backend. Idiomatic Go favors the standard library, explicit loops, and `(value, error)` over generic functional abstractions.

Consider any new abstraction library only when all of the following hold:

1. A recurring problem cannot be expressed cleanly with the standard library and current project conventions.
2. The abstraction gives a clear, demonstrated improvement in safety or maintainability.
3. The team can read and maintain it without prior exposure.
4. The dependency does not introduce generics-heavy complexity disproportionate to the problem.

---

## 12. API Contract With the Frontend

The frontend and backend are written in different languages and cannot share types directly. Keep a single, language-neutral source of truth for the HTTP contract (e.g. an OpenAPI document), and derive or verify both sides from it rather than maintaining two hand-written definitions that can drift.

* Request and response shapes in the contract are the transport layer; parse them into domain types at the boundary (Section 7) and never expose domain types directly as JSON.
* Return expected failures in a consistent, machine-readable error format (e.g. a stable error `code` field, or RFC 9457 Problem Details), so the frontend can map each code to a typed error.
* Map domain errors to HTTP status codes and error codes in one place (the handler layer), using `errors.Is` / `errors.As`, not scattered across usecases.
* Treat the error code set as part of the contract: adding, renaming, or removing a code is a contract change.

---

## Summary

* Put pure business rules on domain types, not in usecases or handlers.
* Avoid hidden inputs; pass time and other non-deterministic values explicitly, or inject a clock.
* Keep I/O at the handler and infrastructure layers; keep the domain free of I/O, and enforce dependency direction with linters.
* Use `(value, error)` for expected failure, with identifiable errors via `errors.Is` / `errors.As`; do not build `Result`/`Either` on top of it.
* Make domain value objects return new values instead of mutating their receiver; leave builders and DTOs as idiomatic mutable code.
* Use `exhaustive` for enum-like constants and `gochecksumtype` for sealed interfaces.
* Parse untrusted input into validated domain types once, at the boundary, for concepts with real invariants.
* Prefer explicit `for` loops over generic `map`/`filter`/`reduce` helpers.
* Reserve sealed interfaces for genuinely exclusive, high-stakes domain states.
* Do not introduce an FP library.
* Keep a single source of truth for the API contract and a stable, machine-readable error format.
