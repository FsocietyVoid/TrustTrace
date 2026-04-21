import { Webhook } from 'svix'
import { headers } from 'next/headers'
import { WebhookEvent } from '@clerk/nextjs/server'
import { prisma } from '../../../../lib/prisma'

export async function POST(req: Request) {
  const WEBHOOK_SECRET = process.env.CLERK_WEBHOOK_SECRET
  if (!WEBHOOK_SECRET) {
    throw new Error('Missing CLERK_WEBHOOK_SECRET')
  }

  const headerPayload = await headers()
  const svix_id = headerPayload.get('svix-id')
  const svix_timestamp = headerPayload.get('svix-timestamp')
  const svix_signature = headerPayload.get('svix-signature')

  if (!svix_id || !svix_timestamp || !svix_signature) {
    return new Response('Missing svix headers', { status: 400 })
  }

  const payload = await req.json()
  const body = JSON.stringify(payload)

  const wh = new Webhook(WEBHOOK_SECRET)
  let evt: WebhookEvent
  try {
    evt = wh.verify(body, {
      'svix-id': svix_id,
      'svix-timestamp': svix_timestamp,
      'svix-signature': svix_signature,
    }) as WebhookEvent
  } catch (err) {
    console.error('Webhook verification failed:', err)
    return new Response('Invalid signature', { status: 400 })
  }

  const eventType = evt.type

  if (eventType === 'organization.created') {
    const { id, name } = evt.data
    await prisma.organization.upsert({
      where: { clerkOrgId: id },
      update: { name },
      create: { clerkOrgId: id, name }
    })
  }

  if (eventType === 'organizationMembership.created') {
    const { organization, public_user_data } = evt.data
    const org = await prisma.organization.findUnique({
      where: { clerkOrgId: organization.id }
    })
    if (org) {
      await prisma.member.upsert({
        where: {
          clerkUserId_organizationId: {
            clerkUserId: public_user_data.user_id,
            organizationId: org.id
          }
        },
        update: {},
        create: {
          clerkUserId: public_user_data.user_id,
          organizationId: org.id,
          role: 'VIEWER'
        }
      })
    }
  }

  return new Response('', { status: 200 })
}