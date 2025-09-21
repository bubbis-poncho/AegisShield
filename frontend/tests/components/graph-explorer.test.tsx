import React from 'react'
import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { jest } from '@jest/globals'
import '@testing-library/jest-dom'

import { GraphExplorer } from '@/components/graph/GraphExplorer'
import { GraphProvider } from '@/contexts/GraphContext'
import { AuthProvider } from '@/contexts/AuthContext'

/**
 * Component tests for Graph Explorer interface
 * Tests visualization, interaction, filtering, and export functionality
 */

// Mock data for testing
const mockNodes = [
  {
    id: 'person_1',
    type: 'person',
    label: 'John Doe',
    properties: {
      name: 'John Doe',
      ssn: '123456789',
      risk_score: 0.7,
      created_at: '2024-01-15T10:00:00Z'
    },
    position: { x: 100, y: 100 }
  },
  {
    id: 'org_1',
    type: 'organization',
    label: 'ACME Corp',
    properties: {
      name: 'ACME Corp',
      tax_id: '987654321',
      industry: 'finance',
      risk_score: 0.5
    },
    position: { x: 300, y: 150 }
  },
  {
    id: 'account_1',
    type: 'account',
    label: 'ACC1234567',
    properties: {
      account_number: 'ACC1234567',
      bank: 'First National',
      balance: 50000,
      currency: 'USD'
    },
    position: { x: 200, y: 250 }
  },
  {
    id: 'transaction_1',
    type: 'transaction',
    label: 'TXN789',
    properties: {
      transaction_id: 'TXN789',
      amount: 10000,
      currency: 'USD',
      timestamp: '2024-01-15T14:30:00Z',
      status: 'completed'
    },
    position: { x: 400, y: 200 }
  }
]

const mockEdges = [
  {
    id: 'edge_1',
    source: 'person_1',
    target: 'account_1',
    type: 'owns',
    label: 'OWNS',
    properties: {
      since: '2020-01-01',
      ownership_percentage: 100
    }
  },
  {
    id: 'edge_2',
    source: 'org_1',
    target: 'account_1',
    type: 'controls',
    label: 'CONTROLS',
    properties: {
      control_type: 'beneficial_owner',
      percentage: 25
    }
  },
  {
    id: 'edge_3',
    source: 'account_1',
    target: 'transaction_1',
    type: 'sent',
    label: 'SENT',
    properties: {
      amount: 10000,
      timestamp: '2024-01-15T14:30:00Z'
    }
  }
]

const mockGraphData = {
  nodes: mockNodes,
  edges: mockEdges,
  statistics: {
    nodeCount: mockNodes.length,
    edgeCount: mockEdges.length,
    nodeTypes: ['person', 'organization', 'account', 'transaction'],
    edgeTypes: ['owns', 'controls', 'sent']
  }
}

// Mock implementations
const mockGraphContext = {
  graphData: mockGraphData,
  selectedNodes: [],
  selectedEdges: [],
  filters: {
    nodeTypes: [],
    edgeTypes: [],
    riskLevel: { min: 0, max: 1 },
    dateRange: { start: null, end: null }
  },
  searchQuery: '',
  layoutMode: 'force',
  isLoading: false,
  error: null,
  searchEntities: jest.fn(),
  selectNode: jest.fn(),
  selectEdge: jest.fn(),
  clearSelection: jest.fn(),
  applyFilters: jest.fn(),
  setLayoutMode: jest.fn(),
  expandNode: jest.fn(),
  collapseNode: jest.fn(),
  exportGraph: jest.fn(),
  resetGraph: jest.fn()
}

const mockAuthContext = {
  user: {
    id: 'user_1',
    email: 'test@example.com',
    role: 'investigator',
    permissions: ['read_cases', 'write_cases', 'export_data']
  },
  isAuthenticated: true,
  login: jest.fn(),
  logout: jest.fn(),
  refreshToken: jest.fn()
}

// Mock Canvas API for graph rendering
Object.defineProperty(HTMLCanvasElement.prototype, 'getContext', {
  value: jest.fn(() => ({
    clearRect: jest.fn(),
    fillRect: jest.fn(),
    strokeRect: jest.fn(),
    arc: jest.fn(),
    fill: jest.fn(),
    stroke: jest.fn(),
    beginPath: jest.fn(),
    closePath: jest.fn(),
    moveTo: jest.fn(),
    lineTo: jest.fn(),
    fillText: jest.fn(),
    measureText: jest.fn(() => ({ width: 100 })),
    save: jest.fn(),
    restore: jest.fn(),
    translate: jest.fn(),
    scale: jest.fn(),
    rotate: jest.fn(),
    createLinearGradient: jest.fn(() => ({
      addColorStop: jest.fn()
    }))
  }))
})

// Mock requestAnimationFrame
global.requestAnimationFrame = jest.fn((cb) => setTimeout(cb, 16))
global.cancelAnimationFrame = jest.fn()

// Test wrapper component
const TestWrapper: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <AuthProvider value={mockAuthContext}>
      <GraphProvider value={mockGraphContext}>
        {children}
      </GraphProvider>
    </AuthProvider>
  )
}

describe('GraphExplorer Component', () => {
  beforeEach(() => {
    jest.clearAllMocks()
  })

  describe('Rendering and Initial State', () => {
    it('should render graph explorer interface', () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      expect(screen.getByTestId('graph-canvas')).toBeInTheDocument()
      expect(screen.getByTestId('graph-controls')).toBeInTheDocument()
      expect(screen.getByTestId('graph-filters')).toBeInTheDocument()
      expect(screen.getByTestId('graph-legend')).toBeInTheDocument()
    })

    it('should display graph statistics', () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      expect(screen.getByText(`${mockNodes.length} nodes`)).toBeInTheDocument()
      expect(screen.getByText(`${mockEdges.length} edges`)).toBeInTheDocument()
    })

    it('should show loading state when graph is loading', () => {
      const loadingContext = {
        ...mockGraphContext,
        isLoading: true,
        graphData: { nodes: [], edges: [], statistics: { nodeCount: 0, edgeCount: 0 } }
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={loadingContext}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      expect(screen.getByTestId('graph-loading')).toBeInTheDocument()
      expect(screen.getByText('Loading graph...')).toBeInTheDocument()
    })

    it('should display error state when graph fails to load', () => {
      const errorContext = {
        ...mockGraphContext,
        isLoading: false,
        error: 'Failed to load graph data',
        graphData: { nodes: [], edges: [], statistics: { nodeCount: 0, edgeCount: 0 } }
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={errorContext}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      expect(screen.getByTestId('graph-error')).toBeInTheDocument()
      expect(screen.getByText('Failed to load graph data')).toBeInTheDocument()
    })
  })

  describe('Search Functionality', () => {
    it('should handle entity search', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const searchInput = screen.getByTestId('graph-search')
      
      await user.type(searchInput, 'John Doe')
      await user.keyboard('{Enter}')

      await waitFor(() => {
        expect(mockGraphContext.searchEntities).toHaveBeenCalledWith('John Doe')
      })
    })

    it('should clear search when input is empty', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const searchInput = screen.getByTestId('graph-search')
      const clearButton = screen.getByTestId('clear-search')
      
      await user.type(searchInput, 'John Doe')
      await user.click(clearButton)

      await waitFor(() => {
        expect(searchInput).toHaveValue('')
        expect(mockGraphContext.searchEntities).toHaveBeenCalledWith('')
      })
    })

    it('should highlight search results in graph', async () => {
      const user = userEvent.setup()
      const contextWithSearchResults = {
        ...mockGraphContext,
        searchQuery: 'John',
        graphData: {
          ...mockGraphData,
          nodes: mockNodes.map(node => ({
            ...node,
            highlighted: node.label.includes('John')
          }))
        }
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={contextWithSearchResults}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      expect(screen.getByTestId('search-results')).toBeInTheDocument()
      expect(screen.getByText('1 result found')).toBeInTheDocument()
    })
  })

  describe('Filtering Functionality', () => {
    it('should apply node type filters', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const personFilter = screen.getByTestId('filter-node-person')
      
      await user.click(personFilter)

      await waitFor(() => {
        expect(mockGraphContext.applyFilters).toHaveBeenCalledWith({
          ...mockGraphContext.filters,
          nodeTypes: ['person']
        })
      })
    })

    it('should apply edge type filters', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const ownsFilter = screen.getByTestId('filter-edge-owns')
      
      await user.click(ownsFilter)

      await waitFor(() => {
        expect(mockGraphContext.applyFilters).toHaveBeenCalledWith({
          ...mockGraphContext.filters,
          edgeTypes: ['owns']
        })
      })
    })

    it('should apply risk level filters', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const riskSlider = screen.getByTestId('risk-level-slider')
      
      fireEvent.change(riskSlider, { target: { value: '0.5' } })

      await waitFor(() => {
        expect(mockGraphContext.applyFilters).toHaveBeenCalledWith({
          ...mockGraphContext.filters,
          riskLevel: { min: 0.5, max: 1 }
        })
      })
    })

    it('should apply date range filters', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const startDateInput = screen.getByTestId('date-range-start')
      const endDateInput = screen.getByTestId('date-range-end')
      
      await user.type(startDateInput, '2024-01-01')
      await user.type(endDateInput, '2024-01-31')

      await waitFor(() => {
        expect(mockGraphContext.applyFilters).toHaveBeenCalledWith({
          ...mockGraphContext.filters,
          dateRange: { start: '2024-01-01', end: '2024-01-31' }
        })
      })
    })

    it('should clear all filters', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const clearFiltersButton = screen.getByTestId('clear-filters')
      
      await user.click(clearFiltersButton)

      await waitFor(() => {
        expect(mockGraphContext.applyFilters).toHaveBeenCalledWith({
          nodeTypes: [],
          edgeTypes: [],
          riskLevel: { min: 0, max: 1 },
          dateRange: { start: null, end: null }
        })
      })
    })
  })

  describe('Graph Interaction', () => {
    it('should handle node selection', async () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const canvas = screen.getByTestId('graph-canvas')
      
      fireEvent.click(canvas, { clientX: 100, clientY: 100 })

      await waitFor(() => {
        expect(mockGraphContext.selectNode).toHaveBeenCalled()
      })
    })

    it('should handle edge selection', async () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const canvas = screen.getByTestId('graph-canvas')
      
      // Simulate clicking on an edge
      fireEvent.click(canvas, { clientX: 150, clientY: 175 })

      await waitFor(() => {
        expect(mockGraphContext.selectEdge).toHaveBeenCalled()
      })
    })

    it('should handle node expansion', async () => {
      const user = userEvent.setup()
      const contextWithSelection = {
        ...mockGraphContext,
        selectedNodes: ['person_1']
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={contextWithSelection}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      const expandButton = screen.getByTestId('expand-node')
      
      await user.click(expandButton)

      await waitFor(() => {
        expect(mockGraphContext.expandNode).toHaveBeenCalledWith('person_1')
      })
    })

    it('should handle node collapse', async () => {
      const user = userEvent.setup()
      const contextWithSelection = {
        ...mockGraphContext,
        selectedNodes: ['person_1']
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={contextWithSelection}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      const collapseButton = screen.getByTestId('collapse-node')
      
      await user.click(collapseButton)

      await waitFor(() => {
        expect(mockGraphContext.collapseNode).toHaveBeenCalledWith('person_1')
      })
    })
  })

  describe('Layout Controls', () => {
    it('should change layout mode', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const layoutSelect = screen.getByTestId('layout-selector')
      
      await user.selectOptions(layoutSelect, 'hierarchical')

      await waitFor(() => {
        expect(mockGraphContext.setLayoutMode).toHaveBeenCalledWith('hierarchical')
      })
    })

    it('should handle zoom controls', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const zoomInButton = screen.getByTestId('zoom-in')
      const zoomOutButton = screen.getByTestId('zoom-out')
      const resetZoomButton = screen.getByTestId('reset-zoom')
      
      await user.click(zoomInButton)
      await user.click(zoomOutButton)
      await user.click(resetZoomButton)

      // Verify zoom actions are handled
      expect(screen.getByTestId('zoom-level')).toBeInTheDocument()
    })

    it('should handle pan controls', () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const canvas = screen.getByTestId('graph-canvas')
      
      // Simulate mouse drag for panning
      fireEvent.mouseDown(canvas, { clientX: 100, clientY: 100 })
      fireEvent.mouseMove(canvas, { clientX: 150, clientY: 150 })
      fireEvent.mouseUp(canvas)

      // Verify canvas received pan events
      expect(canvas).toBeInTheDocument()
    })
  })

  describe('Property Panel', () => {
    it('should display node properties when node is selected', () => {
      const contextWithSelection = {
        ...mockGraphContext,
        selectedNodes: ['person_1']
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={contextWithSelection}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      expect(screen.getByTestId('property-panel')).toBeInTheDocument()
      expect(screen.getByText('John Doe')).toBeInTheDocument()
      expect(screen.getByText('123456789')).toBeInTheDocument()
      expect(screen.getByText('Risk Score: 0.7')).toBeInTheDocument()
    })

    it('should display edge properties when edge is selected', () => {
      const contextWithSelection = {
        ...mockGraphContext,
        selectedEdges: ['edge_1']
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={contextWithSelection}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      expect(screen.getByTestId('property-panel')).toBeInTheDocument()
      expect(screen.getByText('OWNS')).toBeInTheDocument()
      expect(screen.getByText('Ownership: 100%')).toBeInTheDocument()
    })

    it('should hide property panel when nothing is selected', () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      expect(screen.queryByTestId('property-panel')).not.toBeInTheDocument()
    })
  })

  describe('Export Functionality', () => {
    it('should export graph as PNG', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const exportButton = screen.getByTestId('export-menu')
      await user.click(exportButton)

      const pngOption = screen.getByTestId('export-png')
      await user.click(pngOption)

      await waitFor(() => {
        expect(mockGraphContext.exportGraph).toHaveBeenCalledWith('png')
      })
    })

    it('should export graph as SVG', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const exportButton = screen.getByTestId('export-menu')
      await user.click(exportButton)

      const svgOption = screen.getByTestId('export-svg')
      await user.click(svgOption)

      await waitFor(() => {
        expect(mockGraphContext.exportGraph).toHaveBeenCalledWith('svg')
      })
    })

    it('should export graph data as JSON', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const exportButton = screen.getByTestId('export-menu')
      await user.click(exportButton)

      const jsonOption = screen.getByTestId('export-json')
      await user.click(jsonOption)

      await waitFor(() => {
        expect(mockGraphContext.exportGraph).toHaveBeenCalledWith('json')
      })
    })

    it('should export filtered graph data', async () => {
      const user = userEvent.setup()
      const contextWithFilters = {
        ...mockGraphContext,
        filters: {
          ...mockGraphContext.filters,
          nodeTypes: ['person']
        }
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={contextWithFilters}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      const exportButton = screen.getByTestId('export-menu')
      await user.click(exportButton)

      const jsonOption = screen.getByTestId('export-json')
      await user.click(jsonOption)

      await waitFor(() => {
        expect(mockGraphContext.exportGraph).toHaveBeenCalledWith('json', {
          applyFilters: true
        })
      })
    })
  })

  describe('Legend and Help', () => {
    it('should display graph legend', () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      expect(screen.getByTestId('graph-legend')).toBeInTheDocument()
      expect(screen.getByText('Person')).toBeInTheDocument()
      expect(screen.getByText('Organization')).toBeInTheDocument()
      expect(screen.getByText('Account')).toBeInTheDocument()
      expect(screen.getByText('Transaction')).toBeInTheDocument()
    })

    it('should show help modal when help button is clicked', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const helpButton = screen.getByTestId('help-button')
      await user.click(helpButton)

      await waitFor(() => {
        expect(screen.getByTestId('help-modal')).toBeInTheDocument()
        expect(screen.getByText('Graph Explorer Help')).toBeInTheDocument()
      })
    })

    it('should close help modal when close button is clicked', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      const helpButton = screen.getByTestId('help-button')
      await user.click(helpButton)

      const closeButton = screen.getByTestId('close-help-modal')
      await user.click(closeButton)

      await waitFor(() => {
        expect(screen.queryByTestId('help-modal')).not.toBeInTheDocument()
      })
    })
  })

  describe('Performance and Accessibility', () => {
    it('should handle large graphs efficiently', () => {
      const largeGraphData = {
        nodes: Array.from({ length: 1000 }, (_, i) => ({
          id: `node_${i}`,
          type: 'person',
          label: `Person ${i}`,
          properties: { name: `Person ${i}` },
          position: { x: Math.random() * 800, y: Math.random() * 600 }
        })),
        edges: Array.from({ length: 2000 }, (_, i) => ({
          id: `edge_${i}`,
          source: `node_${Math.floor(i / 2)}`,
          target: `node_${Math.floor(i / 2) + 1}`,
          type: 'knows',
          label: 'KNOWS',
          properties: {}
        })),
        statistics: { nodeCount: 1000, edgeCount: 2000 }
      }

      const largeGraphContext = {
        ...mockGraphContext,
        graphData: largeGraphData
      }

      const renderStart = performance.now()
      
      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={largeGraphContext}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      const renderTime = performance.now() - renderStart
      
      // Verify render time is reasonable (< 1000ms)
      expect(renderTime).toBeLessThan(1000)
      expect(screen.getByTestId('graph-canvas')).toBeInTheDocument()
    })

    it('should be accessible with keyboard navigation', async () => {
      const user = userEvent.setup()

      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      // Test tab navigation
      await user.tab()
      expect(screen.getByTestId('graph-search')).toHaveFocus()

      await user.tab()
      expect(screen.getByTestId('layout-selector')).toHaveFocus()

      await user.tab()
      expect(screen.getByTestId('zoom-in')).toHaveFocus()
    })

    it('should have proper ARIA labels', () => {
      render(
        <TestWrapper>
          <GraphExplorer caseId="case_123" />
        </TestWrapper>
      )

      expect(screen.getByLabelText('Search entities in graph')).toBeInTheDocument()
      expect(screen.getByLabelText('Select graph layout')).toBeInTheDocument()
      expect(screen.getByLabelText('Zoom in')).toBeInTheDocument()
      expect(screen.getByLabelText('Zoom out')).toBeInTheDocument()
      expect(screen.getByLabelText('Graph visualization canvas')).toBeInTheDocument()
    })
  })

  describe('Error Handling', () => {
    it('should handle export errors gracefully', async () => {
      const user = userEvent.setup()
      const errorContext = {
        ...mockGraphContext,
        exportGraph: jest.fn().mockRejectedValue(new Error('Export failed'))
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={errorContext}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      const exportButton = screen.getByTestId('export-menu')
      await user.click(exportButton)

      const pngOption = screen.getByTestId('export-png')
      await user.click(pngOption)

      await waitFor(() => {
        expect(screen.getByTestId('export-error')).toBeInTheDocument()
        expect(screen.getByText('Failed to export graph')).toBeInTheDocument()
      })
    })

    it('should handle search errors gracefully', async () => {
      const user = userEvent.setup()
      const errorContext = {
        ...mockGraphContext,
        searchEntities: jest.fn().mockRejectedValue(new Error('Search failed'))
      }

      render(
        <AuthProvider value={mockAuthContext}>
          <GraphProvider value={errorContext}>
            <GraphExplorer caseId="case_123" />
          </GraphProvider>
        </AuthProvider>
      )

      const searchInput = screen.getByTestId('graph-search')
      
      await user.type(searchInput, 'John Doe')
      await user.keyboard('{Enter}')

      await waitFor(() => {
        expect(screen.getByTestId('search-error')).toBeInTheDocument()
        expect(screen.getByText('Search failed')).toBeInTheDocument()
      })
    })
  })
})