'use client'

import { useAuth } from '@/contexts/auth-context'
import { Dashboard } from '@/components/dashboard/dashboard'
import { LoginForm } from '@/components/auth/login-form'

export default function HomePage() {
  const { isAuthenticated, isLoading } = useAuth()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-primary"></div>
      </div>
    )
  }

  if (!isAuthenticated) {
    return <LoginForm />
  }

  return <Dashboard />
}