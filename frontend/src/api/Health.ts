import { Context, Data, Effect, Layer } from "effect"
import { HttpClient } from "effect/http"

export class BackendUnavailable extends Data.TaggedError("BackendUnavailable")<{
  readonly cause: unknown
}> {}

export class HealthApi extends Context.Service<
  HealthApi,
  {
    readonly check: Effect.Effect<void, BackendUnavailable>
  }
>()("HealthApi") {}

export const HealthApiLive = Layer.effect(
  HealthApi,
  Effect.gen(function* () {
    const http = (yield* HttpClient.HttpClient).pipe(HttpClient.filterStatusOk)
    return {
      check: http.get("/healthz").pipe(
        Effect.asVoid,
        Effect.mapError((cause) => new BackendUnavailable({ cause })),
      ),
    }
  }),
)
