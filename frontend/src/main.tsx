import { createRouter, RouterProvider } from "@tanstack/react-router"
import { StrictMode } from "react"
import { createRoot } from "react-dom/client"
import { routeTree } from "./routeTree.gen"

const router = createRouter({ routeTree })

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router
  }
}

const container = document.getElementById("root")
if (container === null) {
  throw new Error("Root element #root not found")
}

createRoot(container).render(
  <StrictMode>
    <RouterProvider router={router} />
  </StrictMode>,
)
