import { inject, type InjectionKey } from 'vue'
import type { Transport } from '@connectrpc/connect'
import type { PromiseClient } from '@connectrpc/connect'
import { AdminService } from '../gen/archivematica/ccp/admin/v1beta1/admin_connect'

type AdminServiceClient = PromiseClient<typeof AdminService>

const transportKey: InjectionKey<Transport> & symbol = Symbol()
const adminClientKey: InjectionKey<AdminServiceClient> & symbol = Symbol()

function useInject<T>(key: InjectionKey<T>): T {
  const injected = inject(key)
  if (!injected) {
    throw new Error(`No provider found for ${key.description || key.toString()}`)
  }
  return injected
}

function useTransport(): Transport {
  return useInject<Transport>(transportKey)
}

function useAdminServiceClient(): AdminServiceClient {
  return useInject<AdminServiceClient>(adminClientKey)
}

export type { AdminServiceClient }

export { transportKey, adminClientKey, useTransport, useAdminServiceClient }
