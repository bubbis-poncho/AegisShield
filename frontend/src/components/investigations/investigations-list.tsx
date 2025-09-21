'use client'

import { useState, useEffect } from 'react'

interface Investigation {
  id: string
  title: string
  description: string
  status: 'OPEN' | 'IN_PROGRESS' | 'CLOSED' | 'ARCHIVED'
  priority: 'LOW' | 'MEDIUM' | 'HIGH' | 'CRITICAL'
  assigned_to: string
  created_at: string
  updated_at: string
  entity_count: number
  alert_count: number
}

export function InvestigationsList() {
  const [investigations, setInvestigations] = useState<Investigation[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Simulate API call - replace with real GraphQL query
    const mockInvestigations: Investigation[] = [
      {
        id: 'inv_001',
        title: 'Money Laundering Network Investigation',
        description: 'Complex network of shell companies and suspicious transactions',
        status: 'IN_PROGRESS',
        priority: 'CRITICAL',
        assigned_to: 'senior.analyst@aegisshield.com',
        created_at: '2024-01-10T14:30:00Z',
        updated_at: '2024-01-15T16:45:00Z',
        entity_count: 23,
        alert_count: 8
      },
      {
        id: 'inv_002',
        title: 'Sanctions Violation Case',
        description: 'Potential sanctions evasion through intermediary entities',
        status: 'OPEN',
        priority: 'HIGH',
        assigned_to: 'compliance.officer@aegisshield.com',
        created_at: '2024-01-12T09:15:00Z',
        updated_at: '2024-01-14T11:20:00Z',
        entity_count: 7,
        alert_count: 3
      },
      {
        id: 'inv_003',
        title: 'Structured Transaction Analysis',
        description: 'Multiple transactions just below reporting thresholds',
        status: 'IN_PROGRESS',
        priority: 'MEDIUM',
        assigned_to: 'analyst@aegisshield.com',
        created_at: '2024-01-13T16:00:00Z',
        updated_at: '2024-01-15T10:30:00Z',
        entity_count: 12,
        alert_count: 15
      }
    ]

    setTimeout(() => {
      setInvestigations(mockInvestigations)
      setLoading(false)
    }, 800)
  }, [])

  const getStatusColor = (status: Investigation['status']) => {
    switch (status) {
      case 'OPEN': return 'bg-blue-50 text-blue-700 border-blue-200'
      case 'IN_PROGRESS': return 'bg-yellow-50 text-yellow-700 border-yellow-200'
      case 'CLOSED': return 'bg-green-50 text-green-700 border-green-200'
      case 'ARCHIVED': return 'bg-gray-50 text-gray-700 border-gray-200'
      default: return 'bg-gray-50 text-gray-700 border-gray-200'
    }
  }

  const getPriorityColor = (priority: Investigation['priority']) => {
    switch (priority) {
      case 'CRITICAL': return 'text-red-600'
      case 'HIGH': return 'text-orange-600'
      case 'MEDIUM': return 'text-yellow-600'
      case 'LOW': return 'text-green-600'
      default: return 'text-gray-600'
    }
  }

  if (loading) {
    return (
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="animate-pulse">
            <div className="h-5 bg-gray-200 rounded w-4/5 mb-2"></div>
            <div className="h-4 bg-gray-200 rounded w-3/5 mb-2"></div>
            <div className="h-3 bg-gray-200 rounded w-2/5"></div>
          </div>
        ))}
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {investigations.map((investigation) => (
        <div
          key={investigation.id}
          className="border rounded-lg p-4 hover:shadow-md transition-shadow cursor-pointer"
        >
          <div className="flex items-start justify-between mb-2">
            <h3 className="font-semibold text-gray-900 truncate flex-1">
              {investigation.title}
            </h3>
            <div className="flex items-center space-x-2 ml-2">
              <span className={`px-2 py-1 text-xs font-medium rounded border ${getStatusColor(investigation.status)}`}>
                {investigation.status.replace('_', ' ')}
              </span>
              <span className={`text-xs font-medium ${getPriorityColor(investigation.priority)}`}>
                {investigation.priority}
              </span>
            </div>
          </div>
          
          <p className="text-sm text-gray-600 mb-3 line-clamp-2">
            {investigation.description}
          </p>
          
          <div className="flex items-center justify-between text-xs text-gray-500 mb-3">
            <div className="flex items-center space-x-4">
              <span>Entities: {investigation.entity_count}</span>
              <span>Alerts: {investigation.alert_count}</span>
            </div>
            <span>
              Updated: {new Date(investigation.updated_at).toLocaleDateString()}
            </span>
          </div>
          
          <div className="flex items-center justify-between">
            <div className="text-xs text-gray-500">
              Assigned to: {investigation.assigned_to}
            </div>
            <div className="flex space-x-2">
              <button className="px-3 py-1 text-xs bg-blue-600 text-white rounded hover:bg-blue-700">
                Open
              </button>
              <button className="px-3 py-1 text-xs bg-gray-600 text-white rounded hover:bg-gray-700">
                Details
              </button>
            </div>
          </div>
        </div>
      ))}
      
      {investigations.length === 0 && (
        <div className="text-center py-8 text-gray-500">
          <p>No investigations found</p>
        </div>
      )}
    </div>
  )
}