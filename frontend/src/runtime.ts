import { Layer, ManagedRuntime } from "effect"
import { FetchHttpClient } from "effect/http"
import { HealthApiLive } from "./api/Health"

const AppLayer = Layer.mergeAll(HealthApiLive).pipe(Layer.provide(FetchHttpClient.layer))

export const runtime = ManagedRuntime.make(AppLayer)
