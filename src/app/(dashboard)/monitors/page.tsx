import { getOrgContext } from '../../../lib/auth'
import { prisma } from '../../../lib/prisma'
import Link from 'next/link'
import { MonitorCard } from '@/components/monitors/MonitorCard'

export default async function MonitorsPage() {
  const { orgId } = await getOrgContext()
  const monitors = await prisma.monitor.findMany({
    where: { organizationId: orgId },
    orderBy: { createdAt: 'desc' },
    include: {
      pingLogs: { orderBy: { checkedAt: 'desc' }, take: 1 }
    }
  })

  return (
    <div>
      <div className="flex justify-between mb-6">
        <h1 className="text-2xl font-bold">Monitors</h1>
        <Link
          href="/dashboard/monitors/new"
          className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
        >
          Add Monitor
        </Link>
      </div>
      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
        {monitors.map((monitor: { id: any }) => (
          <MonitorCard key={monitor.id} monitor={monitor} />
        ))}
      </div>
    </div>
  )
}