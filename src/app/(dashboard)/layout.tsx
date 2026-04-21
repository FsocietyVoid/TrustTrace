import { ClerkProvider, OrganizationSwitcher, UserButton } from '@clerk/nextjs'
import Link from 'next/link'

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <ClerkProvider>
      <div className="min-h-screen bg-gray-50">
        <nav className="bg-white shadow-sm p-4 flex justify-between">
          <div className="flex items-center space-x-4">
            <Link href="/dashboard" className="font-bold text-xl">PulseBoard</Link>
            <Link href="/dashboard/monitors">Monitors</Link>
            <Link href="/dashboard/incidents">Incidents</Link>
          </div>
          <div className="flex items-center space-x-4">
            <OrganizationSwitcher />
            <UserButton />
          </div>
        </nav>
        <main className="p-6">{children}</main>
      </div>
    </ClerkProvider>
  )
}