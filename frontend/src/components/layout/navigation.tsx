'use client'

interface User {
  id: string
  email: string
  roles: string[]
}

interface NavigationProps {
  user: User | null
  onLogout: () => void
}

export function Navigation({ user, onLogout }: NavigationProps) {
  return (
    <nav className="bg-white shadow-sm border-b">
      <div className="container mx-auto px-4">
        <div className="flex items-center justify-between h-16">
          {/* Logo and Brand */}
          <div className="flex items-center space-x-4">
            <div className="flex items-center">
              <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
                <span className="text-white font-bold text-sm">AS</span>
              </div>
              <span className="ml-2 text-xl font-bold text-gray-900">AegisShield</span>
            </div>
          </div>

          {/* Navigation Links */}
          <div className="hidden md:flex items-center space-x-8">
            <a href="/dashboard" className="text-gray-600 hover:text-gray-900 px-3 py-2 text-sm font-medium">
              Dashboard
            </a>
            <a href="/investigations" className="text-gray-600 hover:text-gray-900 px-3 py-2 text-sm font-medium">
              Investigations
            </a>
            <a href="/alerts" className="text-gray-600 hover:text-gray-900 px-3 py-2 text-sm font-medium">
              Alerts
            </a>
            <a href="/entities" className="text-gray-600 hover:text-gray-900 px-3 py-2 text-sm font-medium">
              Entities
            </a>
            <a href="/graph" className="text-gray-600 hover:text-gray-900 px-3 py-2 text-sm font-medium">
              Graph Explorer
            </a>
            <a href="/reports" className="text-gray-600 hover:text-gray-900 px-3 py-2 text-sm font-medium">
              Reports
            </a>
          </div>

          {/* User Menu */}
          <div className="flex items-center space-x-4">
            {/* Notifications */}
            <button className="relative p-2 text-gray-600 hover:text-gray-900">
              <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                      d="M15 17h5l-3.5-3.5L15 17zM16 19H8a1 1 0 01-1-1V6a1 1 0 011-1h8a1 1 0 011 1v12a1 1 0 01-1 1z" />
              </svg>
              <span className="absolute -top-1 -right-1 h-4 w-4 bg-red-500 text-white text-xs rounded-full flex items-center justify-center">
                3
              </span>
            </button>

            {/* User Info */}
            <div className="flex items-center space-x-3">
              <div className="text-right">
                <div className="text-sm font-medium text-gray-900">{user?.email}</div>
                <div className="text-xs text-gray-500">
                  {user?.roles.includes('admin') ? 'Administrator' : 
                   user?.roles.includes('investigator') ? 'Investigator' : 'Analyst'}
                </div>
              </div>
              
              {/* Avatar */}
              <div className="w-8 h-8 bg-gray-300 rounded-full flex items-center justify-center">
                <span className="text-gray-600 text-sm font-medium">
                  {user?.email.charAt(0).toUpperCase()}
                </span>
              </div>
              
              {/* Logout */}
              <button
                onClick={onLogout}
                className="text-gray-600 hover:text-gray-900 text-sm font-medium"
              >
                Logout
              </button>
            </div>
          </div>

          {/* Mobile menu button */}
          <div className="md:hidden">
            <button className="text-gray-600 hover:text-gray-900 p-2">
              <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} 
                      d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
          </div>
        </div>
      </div>
    </nav>
  )
}