'use client'

import { useState, useEffect } from 'react'
import { useAuth } from '@/contexts/auth-context'

interface Alert {
  id: string
  title: string
  description: string
  severity: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  status: 'OPEN' | 'INVESTIGATING' | 'RESOLVED' | 'CLOSED'
  created_at: string
  risk_score: number
  entities: string[]
}

export function AlertsList() {
  const [alerts, setAlerts] = useState<Alert[]>([])
  const [loading, setLoading] = useState(true)
  const { user } = useAuth()

  useEffect(() => {
    // Simulate API call - replace with real GraphQL query
    const mockAlerts: Alert[] = [
      {
        id: 'alert_001',
        title: 'Suspicious High-Value Transfer',
        description: 'Large wire transfer to high-risk jurisdiction',
        severity: 'CRITICAL',
        status: 'OPEN',
        created_at: '2024-01-15T10:30:00Z',
        risk_score: 0.95,
        entities: ['person_12345', 'org_67890']
      },
      {
        id: 'alert_002',
        title: 'Unusual Transaction Pattern',
        description: 'Multiple small transactions below reporting threshold',
        severity: 'HIGH',
        status: 'INVESTIGATING',
        created_at: '2024-01-15T09:45:00Z',
        risk_score: 0.87,
        entities: ['person_54321']
      },
      {
        id: 'alert_003',
        title: 'Sanctions List Match',
        description: 'Transaction involving entity on sanctions list',
        severity: 'CRITICAL',
        status: 'OPEN',
        created_at: '2024-01-15T08:20:00Z',
        risk_score: 0.99,
        entities: ['org_99999']
      }
    ]

    setTimeout(() => {
      setAlerts(mockAlerts)
      setLoading(false)
    }, 1000)
  }, [])

  const getSeverityColor = (severity: Alert['severity']) => {
    switch (severity) {
      case 'CRITICAL': return 'bg-red-100 text-red-800 border-red-200'
      case 'HIGH': return 'bg-orange-100 text-orange-800 border-orange-200'
      case 'MEDIUM': return 'bg-yellow-100 text-yellow-800 border-yellow-200'
      case 'LOW': return 'bg-green-100 text-green-800 border-green-200'
      default: return 'bg-gray-100 text-gray-800 border-gray-200'
    }
  }

  const getStatusColor = (status: Alert['status']) => {
    switch (status) {
      case 'OPEN': return 'bg-red-50 text-red-700'
      case 'INVESTIGATING': return 'bg-yellow-50 text-yellow-700'
      case 'RESOLVED': return 'bg-green-50 text-green-700'
      case 'CLOSED': return 'bg-gray-50 text-gray-700'
      default: return 'bg-gray-50 text-gray-700'
    }
  }

  if (loading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <div key={i} className="animate-pulse">
            <div className="h-4 bg-gray-200 rounded w-3/4 mb-2"></div>
            <div className="h-3 bg-gray-200 rounded w-1/2"></div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {alerts.map((alert) => (
        <div
          key={alert.id}
          className="border rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
        >
          <div className="flex items-start justify-between mb-2">
            <h3 className="font-semibold text-gray-900 truncate flex-1">
              {alert.title}
            </h3>
            <div className="flex items-center space-x-2 ml-2">
              <span className={`px-2 py-1 text-xs font-medium rounded-full border ${getSeverityColor(alert.severity)}`}>
                {alert.severity}
              </span>
              <span className={`px-2 py-1 text-xs font-medium rounded ${getStatusColor(alert.status)}`}>
                {alert.status}
              </span>
            </div>
          </div>
          
          <p className="text-sm text-gray-600 mb-3">
            {alert.description}
          </p>
          
          <div className="flex items-center justify-between text-xs text-gray-500">
            <div className="flex items-center space-x-4">
              <span>Risk Score: {(alert.risk_score * 100).toFixed(0)}%</span>
              <span>Entities: {alert.entities.length}</span>
            </div>
            <span>
              {new Date(alert.created_at).toLocaleDateString()} {new Date(alert.created_at).toLocaleTimeString()}
            </span>
          </div>
          
          <div className="mt-3 flex space-x-2">
            <button className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700">
              Investigate
            </button>
            <button className="px-3 py-1 text-xs bg-gray-600 text-white rounded hover:bg-gray-700">
              View Details
            </button>
          </div>
        </div>
      ))}
      
      {alerts.length === 0 && (
        <div className="text-center py-8 text-gray-500">
          <p>No active alerts found</p>
        </div>
      )}
    </div>
  )
}