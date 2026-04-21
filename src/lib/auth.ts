import { auth, clerkClient } from '@clerk/nextjs/server'
import { prisma } from './prisma'

export async function getOrgContext() {
  const { userId, orgId } = await auth()
  if (!userId || !orgId) {
    throw new Error('Unauthorized: Missing user or organization')
  }

  // Try to find the org in our DB
  let org = await prisma.organization.findUnique({
    where: { clerkOrgId: orgId }
  })

  // If not found, create it now (useful when webhook hasn't fired)
  if (!org) {
    const client = await clerkClient()
    const clerkOrg = await client.organizations.getOrganization({ organizationId: orgId })
    
    org = await prisma.organization.create({
      data: {
        clerkOrgId: orgId,
        name: clerkOrg.name
      }
    })
  }

  return { userId, orgId: org.id, clerkOrgId: orgId }
}