import { inject, type InjectionKey } from 'vue'
import type { Client } from './client'

// The client the application was mounted with, which a test mounts with another.
export const clientKey: InjectionKey<Client> = Symbol('client')

export function useClient(): Client {
  const client = inject(clientKey)
  if (client === undefined) throw new Error('the application was mounted without a client')
  return client
}
