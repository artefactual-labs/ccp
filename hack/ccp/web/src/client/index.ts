import { inject } from 'vue'
import type { InjectionKey, App } from 'vue'
import type { Transport } from '@connectrpc/connect'
import type { PromiseClient } from '@connectrpc/connect'
import { AdminService } from '../gen/archivematica/ccp/admin/v1beta1/service_connect'
import { createPromiseClient } from '@connectrpc/connect'
import { createConnectTransport } from '@connectrpc/connect-web'

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

function client(app: App) {
  const loc = window.location
  const baseUrl = `${loc.protocol}//${loc.hostname}:${loc.port}/api`
  const transport = createConnectTransport({ baseUrl })
  app.provide(transportKey, transport)

  const client = createPromiseClient(AdminService, transport)
  app.provide(adminClientKey, client)
}

export type { AdminServiceClient }

export { client, useTransport, useAdminServiceClient }
