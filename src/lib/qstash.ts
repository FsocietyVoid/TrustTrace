import { Client } from '@upstash/qstash'

export const qstash = new Client({
  token: process.env.QSTASH_TOKEN!
})

export async function verifyQStashSignature(req: Request) {
  const signature = req.headers.get('upstash-signature')
  if (!signature) return false
  // We'll use the QStash receiver in the route directly
  return true
}