import React from 'react'
import ReactDOM from 'react-dom/client'
import { RouterProvider } from 'react-router-dom'
import { QueryClientProvider } from '@tanstack/react-query'
import { router } from '@/app/router'
import { queryClient } from '@/app/query-client'
import { AppToaster } from '@/shared/ui/toast'
import { AuthProvider } from '@/entities/auth/auth-context'
import './app/styles/global.css'

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <RouterProvider
          router={router}
          fallbackElement={
            <div className="flex min-h-screen items-center justify-center text-sm text-muted">
              Загрузка…
            </div>
          }
        />
      </AuthProvider>
      <AppToaster />
    </QueryClientProvider>
  </React.StrictMode>,
)
