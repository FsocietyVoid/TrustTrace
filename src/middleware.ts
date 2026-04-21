// src/middleware.ts
import { clerkMiddleware, createRouteMatcher } from '@clerk/nextjs/server'

const isPublicRoute = createRouteMatcher([
  '/status/:slug(.*)',
  '/api/webhooks/clerk(.*)',
  '/api/cron/ping(.*)',
  '/api/worker/ping-single(.*)'
])

export default clerkMiddleware(async (auth, req) => {
  if (!isPublicRoute(req)) {
    await auth.protect() // Ensure user is signed in

    // Check if user has an active organization
    if (!auth.orgId) {
      // Redirect to a custom page to create/join an organization
      const orgSelection = new URL('/organization-selection', req.url)
      return Response.redirect(orgSelection)
    }
  }
})

export const config = {
  matcher: ['/((?!.*\\..*|_next).*)', '/', '/(api|trpc)(.*)'],
}