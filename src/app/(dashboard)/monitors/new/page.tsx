'use client'

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useRouter } from 'next/navigation'
import { useState } from 'react'

const schema = z.object({
  name: z.string().min(1),
  url: z.string().url(),
  interval: z.number().min(10).max(3600)
})

type FormData = z.infer<typeof schema>

export default function NewMonitorPage() {
  const router = useRouter()
  const [error, setError] = useState('')
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting }
  } = useForm<FormData>({
    resolver: zodResolver(schema),
    defaultValues: { interval: 30 }
  })

  const onSubmit = async (data: FormData) => {
    const res = await fetch('/api/monitors', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
    if (res.ok) {
      router.push('/dashboard/monitors')
    } else {
      const json = await res.json()
      setError(json.error || 'Failed to create monitor')
    }
  }

  return (
    <div className="max-w-md mx-auto">
      <h1 className="text-2xl font-bold mb-6">Add Monitor</h1>
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <div>
          <label className="block text-sm font-medium mb-1">Name</label>
          <input
            {...register('name')}
            className="w-full border rounded px-3 py-2"
            placeholder="My API"
          />
          {errors.name && <p className="text-red-500 text-sm">{errors.name.message}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">URL</label>
          <input
            {...register('url')}
            className="w-full border rounded px-3 py-2"
            placeholder="https://example.com/health"
          />
          {errors.url && <p className="text-red-500 text-sm">{errors.url.message}</p>}
        </div>
        <div>
          <label className="block text-sm font-medium mb-1">Check Interval (seconds)</label>
          <input
            type="number"
            {...register('interval', { valueAsNumber: true })}
            className="w-full border rounded px-3 py-2"
          />
          {errors.interval && <p className="text-red-500 text-sm">{errors.interval.message}</p>}
        </div>
        {error && <p className="text-red-500">{error}</p>}
        <button
          type="submit"
          disabled={isSubmitting}
          className="w-full bg-blue-600 text-white py-2 rounded hover:bg-blue-700 disabled:opacity-50"
        >
          {isSubmitting ? 'Creating...' : 'Create Monitor'}
        </button>
      </form>
    </div>
  )
}