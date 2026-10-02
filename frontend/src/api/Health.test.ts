import { describe, expect, it } from "@effect/vitest"
import { Effect, Layer } from "effect"
import { HttpClient, HttpClientRequest, HttpClientResponse } from "effect/http"
import { HealthApi, HealthApiLive } from "./Health"

// A fake HttpClient that answers every request with the given status code.
// Relative URLs need a base outside the browser, so the test prepends one.
const fakeHttpClient = (status: number) =>
  Layer.succeed(
    HttpClient.HttpClient,
    HttpClient.make((request) =>
      Effect.succeed(HttpClientResponse.fromWeb(request, new Response(null, { status }))),
    ).pipe(HttpClient.mapRequest(HttpClientRequest.prependUrl("http://backend.test"))),
  )

const testLayer = (status: number) => HealthApiLive.pipe(Layer.provide(fakeHttpClient(status)))

const check = Effect.gen(function* () {
  const api = yield* HealthApi
  yield* api.check
})

describe("HealthApi", () => {
  it.effect("succeeds when the backend returns 200", () =>
    check.pipe(Effect.provide(testLayer(200))),
  )

  it.effect("fails with BackendUnavailable when the backend returns 503", () =>
    Effect.gen(function* () {
      const error = yield* check.pipe(Effect.flip, Effect.provide(testLayer(503)))
      expect(error._tag).toBe("BackendUnavailable")
    }),
  )
})
