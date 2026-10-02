import { useQuery } from "@tanstack/react-query"
import { createFileRoute } from "@tanstack/react-router"
import { Effect } from "effect"
import { HealthApi } from "../api/Health"
import { runtime } from "../runtime"

export const Route = createFileRoute("/")({
  component: Home,
})

const checkHealth = Effect.gen(function* () {
  const api = yield* HealthApi
  yield* api.check
  return "healthy" as const
})

function Home() {
  const health = useQuery({
    queryKey: ["health"],
    queryFn: ({ signal }) => runtime.runPromise(checkHealth, { signal }),
    retry: false,
  })

  if (health.isPending) return <p>Checking backend...</p>
  if (health.isError) return <p>Backend is unavailable</p>
  return <p>Backend is healthy</p>
}
