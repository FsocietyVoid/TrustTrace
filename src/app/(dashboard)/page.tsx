import { getOrgContext } from '../../lib/auth'
import { prisma } from '../../lib/prisma'
import Link from 'next/link'

export default async function DashboardPage() {
  const { orgId } = await getOrgContext()
  const monitors = await prisma.monitor.findMany({
    where: { organizationId: orgId },
    take: 5,
    orderBy: { createdAt: 'desc' }
  })

  return (
    <div>
      <h1 className="text-2xl font-bold mb-6">Dashboard</h1>
      {monitors.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-gray-600 mb-4">No monitors yet.</p>
          <Link
            href="/dashboard/monitors/new"
            className="bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700"
          >
            Add Your First Monitor
          </Link>
        </div>
      ) : (
        <div>
          <p className="mb-4">You have {monitors.length} monitor(s).</p>
          <Link href="/dashboard/monitors" className="text-blue-600 hover:underline">
            View all monitors →
          </Link>
        </div>
      )}
    </div>
  )
}