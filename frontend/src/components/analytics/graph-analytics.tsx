'use client';

import React, { useState, useEffect, useRef } from 'react';
import { BarChart, LineChart, PieChart, Network, TrendingUp, Users, Activity, AlertTriangle } from 'lucide-react';

interface GraphMetrics {
  totalNodes: number;
  totalEdges: number;
  connectedComponents: number;
  averageDegree: number;
  clusteringCoefficient: number;
  centralityDistribution: { [key: string]: number };
}

interface CommunityDetection {
  communityId: string;
  nodeCount: number;
  density: number;
  riskScore: number;
  suspiciousPatterns: string[];
}

interface PathAnalysis {
  pathId: string;
  sourceNode: string;
  targetNode: string;
  pathLength: number;
  totalAmount: number;
  riskScore: number;
  intermediateNodes: string[];
}

interface GraphAnalyticsProps {
  graphData: any;
  timeRange: {
    start: Date;
    end: Date;
  };
  onInsightSelect?: (insight: any) => void;
}

const GraphAnalytics: React.FC<GraphAnalyticsProps> = ({
  graphData,
  timeRange,
  onInsightSelect
}) => {
  const [metrics, setMetrics] = useState<GraphMetrics | null>(null);
  const [communities, setCommunities] = useState<CommunityDetection[]>([]);
  const [paths, setPaths] = useState<PathAnalysis[]>([]);
  const [loading, setLoading] = useState(true);
  const [activeTab, setActiveTab] = useState<'metrics' | 'communities' | 'paths' | 'patterns'>('metrics');
  const chartRef = useRef<HTMLDivElement>(null);

  // Mock data - in production this would come from your graph analytics service
  useEffect(() => {
    const calculateMockMetrics = () => {
      const mockMetrics: GraphMetrics = {
        totalNodes: 2847,
        totalEdges: 5692,
        connectedComponents: 23,
        averageDegree: 4.2,
        clusteringCoefficient: 0.34,
        centralityDistribution: {
          'Low (0.0-0.2)': 2145,
          'Medium (0.2-0.6)': 543,
          'High (0.6-0.8)': 127,
          'Very High (0.8-1.0)': 32
        }
      };

      const mockCommunities: CommunityDetection[] = [
        {
          communityId: 'C001',
          nodeCount: 156,
          density: 0.78,
          riskScore: 0.92,
          suspiciousPatterns: ['Circular transactions', 'Rapid fire transfers', 'Shell company network']
        },
        {
          communityId: 'C002',
          nodeCount: 89,
          density: 0.65,
          riskScore: 0.87,
          suspiciousPatterns: ['Layering pattern', 'Multiple jurisdictions', 'High-frequency trading']
        },
        {
          communityId: 'C003',
          nodeCount: 234,
          density: 0.45,
          riskScore: 0.73,
          suspiciousPatterns: ['Smurfing detected', 'Cross-border flows', 'Anonymous entities']
        },
        {
          communityId: 'C004',
          nodeCount: 67,
          density: 0.58,
          riskScore: 0.69,
          suspiciousPatterns: ['Wire transfer chains', 'Offshore accounts', 'Complex ownership']
        }
      ];

      const mockPaths: PathAnalysis[] = [
        {
          pathId: 'P001',
          sourceNode: 'ENT-4561',
          targetNode: 'ENT-8923',
          pathLength: 7,
          totalAmount: 2450000,
          riskScore: 0.95,
          intermediateNodes: ['ENT-1234', 'ENT-5678', 'ENT-9012', 'ENT-3456', 'ENT-7890']
        },
        {
          pathId: 'P002',
          sourceNode: 'ENT-2341',
          targetNode: 'ENT-6789',
          pathLength: 5,
          totalAmount: 890000,
          riskScore: 0.88,
          intermediateNodes: ['ENT-4444', 'ENT-5555', 'ENT-6666']
        },
        {
          pathId: 'P003',
          sourceNode: 'ENT-1111',
          targetNode: 'ENT-9999',
          pathLength: 9,
          totalAmount: 3200000,
          riskScore: 0.91,
          intermediateNodes: ['ENT-2222', 'ENT-3333', 'ENT-4444', 'ENT-5555', 'ENT-6666', 'ENT-7777', 'ENT-8888']
        }
      ];

      setMetrics(mockMetrics);
      setCommunities(mockCommunities);
      setPaths(mockPaths);
      setLoading(false);
    };

    calculateMockMetrics();
  }, [graphData, timeRange]);

  const renderMetricsTab = () => (
    <div className="space-y-6">
      {/* Key Metrics Grid */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="bg-white p-6 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <Network className="h-8 w-8 text-blue-500" />
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Total Entities</p>
              <p className="text-2xl font-semibold text-gray-900">{metrics?.totalNodes.toLocaleString()}</p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <Activity className="h-8 w-8 text-green-500" />
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Connections</p>
              <p className="text-2xl font-semibold text-gray-900">{metrics?.totalEdges.toLocaleString()}</p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <Users className="h-8 w-8 text-purple-500" />
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Components</p>
              <p className="text-2xl font-semibold text-gray-900">{metrics?.connectedComponents}</p>
            </div>
          </div>
        </div>

        <div className="bg-white p-6 rounded-lg border border-gray-200">
          <div className="flex items-center">
            <TrendingUp className="h-8 w-8 text-orange-500" />
            <div className="ml-4">
              <p className="text-sm font-medium text-gray-500">Avg Degree</p>
              <p className="text-2xl font-semibold text-gray-900">{metrics?.averageDegree}</p>
            </div>
          </div>
        </div>
      </div>

      {/* Centrality Distribution */}
      <div className="bg-white p-6 rounded-lg border border-gray-200">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Centrality Distribution</h3>
        <div className="space-y-2">
          {metrics?.centralityDistribution && Object.entries(metrics.centralityDistribution).map(([range, count]) => (
            <div key={range} className="flex items-center justify-between">
              <span className="text-sm text-gray-600">{range}</span>
              <div className="flex items-center space-x-2">
                <div className="w-32 bg-gray-200 rounded-full h-2">
                  <div
                    className="bg-blue-500 h-2 rounded-full"
                    style={{ width: `${(count / metrics.totalNodes) * 100}%` }}
                  />
                </div>
                <span className="text-sm font-medium text-gray-900">{count}</span>
              </div>
            </div>
          ))}
        </div>
      </div>

      {/* Network Health */}
      <div className="bg-white p-6 rounded-lg border border-gray-200">
        <h3 className="text-lg font-medium text-gray-900 mb-4">Network Health Indicators</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <div className="text-center">
            <div className="text-3xl font-bold text-green-500">0.34</div>
            <div className="text-sm text-gray-600">Clustering Coefficient</div>
            <div className="text-xs text-gray-500">Good connectivity</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-yellow-500">23</div>
            <div className="text-sm text-gray-600">Components</div>
            <div className="text-xs text-gray-500">Some fragmentation</div>
          </div>
          <div className="text-center">
            <div className="text-3xl font-bold text-red-500">32</div>
            <div className="text-sm text-gray-600">High Centrality Nodes</div>
            <div className="text-xs text-gray-500">Potential hubs</div>
          </div>
        </div>
      </div>
    </div>
  );

  const renderCommunitiesTab = () => (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900">Community Detection Results</h3>
        <span className="text-sm text-gray-500">{communities.length} communities found</span>
      </div>

      <div className="grid gap-4">
        {communities.map(community => (
          <div key={community.communityId} className="bg-white p-6 rounded-lg border border-gray-200">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-2 mb-2">
                  <h4 className="text-base font-medium text-gray-900">Community {community.communityId}</h4>
                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                    community.riskScore >= 0.8 ? 'bg-red-100 text-red-800' :
                    community.riskScore >= 0.6 ? 'bg-yellow-100 text-yellow-800' :
                    'bg-green-100 text-green-800'
                  }`}>
                    Risk: {(community.riskScore * 100).toFixed(0)}%
                  </span>
                </div>

                <div className="grid grid-cols-3 gap-4 mb-3">
                  <div>
                    <span className="text-sm text-gray-500">Nodes</span>
                    <div className="text-lg font-semibold text-gray-900">{community.nodeCount}</div>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Density</span>
                    <div className="text-lg font-semibold text-gray-900">{community.density.toFixed(2)}</div>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Risk Score</span>
                    <div className="text-lg font-semibold text-gray-900">{community.riskScore.toFixed(2)}</div>
                  </div>
                </div>

                <div>
                  <span className="text-sm font-medium text-gray-700">Suspicious Patterns:</span>
                  <div className="flex flex-wrap gap-1 mt-1">
                    {community.suspiciousPatterns.map((pattern, index) => (
                      <span
                        key={index}
                        className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-orange-100 text-orange-800"
                      >
                        {pattern}
                      </span>
                    ))}
                  </div>
                </div>
              </div>

              <button
                onClick={() => onInsightSelect && onInsightSelect({ type: 'community', data: community })}
                className="ml-4 px-4 py-2 text-sm font-medium text-blue-600 hover:text-blue-800"
              >
                Investigate
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  const renderPathsTab = () => (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900">Suspicious Transaction Paths</h3>
        <span className="text-sm text-gray-500">{paths.length} high-risk paths detected</span>
      </div>

      <div className="space-y-4">
        {paths.map(path => (
          <div key={path.pathId} className="bg-white p-6 rounded-lg border border-gray-200">
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center space-x-2 mb-2">
                  <h4 className="text-base font-medium text-gray-900">Path {path.pathId}</h4>
                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                    path.riskScore >= 0.9 ? 'bg-red-100 text-red-800' :
                    path.riskScore >= 0.7 ? 'bg-yellow-100 text-yellow-800' :
                    'bg-green-100 text-green-800'
                  }`}>
                    Risk: {(path.riskScore * 100).toFixed(0)}%
                  </span>
                </div>

                <div className="grid grid-cols-4 gap-4 mb-3">
                  <div>
                    <span className="text-sm text-gray-500">Path Length</span>
                    <div className="text-lg font-semibold text-gray-900">{path.pathLength} hops</div>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Total Amount</span>
                    <div className="text-lg font-semibold text-gray-900">${path.totalAmount.toLocaleString()}</div>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Source</span>
                    <div className="text-sm font-medium text-blue-600">{path.sourceNode}</div>
                  </div>
                  <div>
                    <span className="text-sm text-gray-500">Target</span>
                    <div className="text-sm font-medium text-blue-600">{path.targetNode}</div>
                  </div>
                </div>

                <div>
                  <span className="text-sm font-medium text-gray-700">Path Flow:</span>
                  <div className="flex items-center space-x-2 mt-1 text-xs">
                    <span className="px-2 py-1 bg-blue-100 text-blue-800 rounded">{path.sourceNode}</span>
                    {path.intermediateNodes.slice(0, 3).map((node, index) => (
                      <React.Fragment key={node}>
                        <span>→</span>
                        <span className="px-2 py-1 bg-gray-100 text-gray-700 rounded">{node}</span>
                      </React.Fragment>
                    ))}
                    {path.intermediateNodes.length > 3 && (
                      <>
                        <span>→ ... →</span>
                      </>
                    )}
                    <span>→</span>
                    <span className="px-2 py-1 bg-red-100 text-red-800 rounded">{path.targetNode}</span>
                  </div>
                </div>
              </div>

              <button
                onClick={() => onInsightSelect && onInsightSelect({ type: 'path', data: path })}
                className="ml-4 px-4 py-2 text-sm font-medium text-blue-600 hover:text-blue-800"
              >
                Trace Path
              </button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  const renderPatternsTab = () => (
    <div className="space-y-6">
      <h3 className="text-lg font-medium text-gray-900">Pattern Detection Summary</h3>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {[
          { name: 'Circular Transactions', count: 15, risk: 'high', description: 'Money flows in closed loops' },
          { name: 'Rapid Fire Transfers', count: 28, risk: 'medium', description: 'High frequency transaction patterns' },
          { name: 'Shell Company Networks', count: 8, risk: 'high', description: 'Complex ownership structures' },
          { name: 'Layering Patterns', count: 12, risk: 'high', description: 'Multiple transaction layers detected' },
          { name: 'Cross-Border Flows', count: 34, risk: 'medium', description: 'International transfer patterns' },
          { name: 'Smurfing Activity', count: 7, risk: 'high', description: 'Transaction amount splitting detected' }
        ].map(pattern => (
          <div key={pattern.name} className="bg-white p-4 rounded-lg border border-gray-200">
            <div className="flex items-center justify-between mb-2">
              <h4 className="text-sm font-medium text-gray-900">{pattern.name}</h4>
              <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                pattern.risk === 'high' ? 'bg-red-100 text-red-800' :
                pattern.risk === 'medium' ? 'bg-yellow-100 text-yellow-800' :
                'bg-green-100 text-green-800'
              }`}>
                {pattern.risk}
              </span>
            </div>
            <div className="text-2xl font-bold text-gray-900 mb-1">{pattern.count}</div>
            <p className="text-xs text-gray-500">{pattern.description}</p>
          </div>
        ))}
      </div>

      <div className="bg-white p-6 rounded-lg border border-gray-200">
        <h4 className="text-base font-medium text-gray-900 mb-4">Risk Distribution Over Time</h4>
        <div className="h-64 flex items-center justify-center bg-gray-50 rounded">
          <p className="text-gray-500">Time series chart would be rendered here</p>
        </div>
      </div>
    </div>
  );

  if (loading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-gray-900">Graph Analytics</h2>
        <div className="flex items-center space-x-2">
          <span className="text-sm text-gray-500">
            {timeRange.start.toLocaleDateString()} - {timeRange.end.toLocaleDateString()}
          </span>
        </div>
      </div>

      {/* Tab Navigation */}
      <div className="border-b border-gray-200">
        <nav className="-mb-px flex space-x-8">
          {[
            { key: 'metrics', label: 'Network Metrics', icon: BarChart },
            { key: 'communities', label: 'Communities', icon: Users },
            { key: 'paths', label: 'Suspicious Paths', icon: Network },
            { key: 'patterns', label: 'Pattern Analysis', icon: AlertTriangle }
          ].map(tab => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key as any)}
              className={`flex items-center space-x-2 py-2 px-1 border-b-2 font-medium text-sm ${
                activeTab === tab.key
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
              }`}
            >
              <tab.icon className="h-4 w-4" />
              <span>{tab.label}</span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div>
        {activeTab === 'metrics' && renderMetricsTab()}
        {activeTab === 'communities' && renderCommunitiesTab()}
        {activeTab === 'paths' && renderPathsTab()}
        {activeTab === 'patterns' && renderPatternsTab()}
      </div>
    </div>
  );
};

export default GraphAnalytics;