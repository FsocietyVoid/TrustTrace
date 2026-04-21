import { NextRequest, NextResponse } from 'next/server'
import { getOrgContext } from '../../../lib/auth'
import { prisma } from '../../../lib/prisma'
import { isSafeUrl } from '@/lib/ssrf'

export async function GET() {
  const { orgId } = await getOrgContext()
  const monitors = await prisma.monitor.findMany({
    where: { organizationId: orgId }
  })
  return NextResponse.json(monitors)
}

export async function POST(req: NextRequest) {
  const { orgId } = await getOrgContext()
  const body = await req.json()
  const { name, url, interval } = body

  if (!isSafeUrl(url)) {
    return NextResponse.json({ error: 'URL is not allowed (SSRF protection)' }, { status: 400 })
  }

  const monitor = await prisma.monitor.create({
    data: {
      name,
      url,
      interval,
      organizationId: orgId,
      status: 'PENDING'
    }
  })

  return NextResponse.json(monitor, { status: 201 })
}