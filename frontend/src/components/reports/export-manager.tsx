'use client';

import React, { useState, useEffect } from 'react';
import { Download, FileText, Calendar, Filter, Search } from 'lucide-react';

interface ExportOptions {
  format: 'pdf' | 'excel' | 'csv' | 'json';
  template: string;
  includeCharts: boolean;
  includeMetadata: boolean;
  dateRange: {
    start: Date;
    end: Date;
  };
}

interface ReportTemplate {
  id: string;
  name: string;
  description: string;
  type: 'investigation' | 'compliance' | 'audit' | 'analytics';
  fields: string[];
  supportedFormats: string[];
}

interface ExportManagerProps {
  investigationId?: string;
  alertIds?: string[];
  entityIds?: string[];
  onExportComplete?: (exportId: string) => void;
}

const ExportManager: React.FC<ExportManagerProps> = ({
  investigationId,
  alertIds = [],
  entityIds = [],
  onExportComplete
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [exportOptions, setExportOptions] = useState<ExportOptions>({
    format: 'pdf',
    template: 'standard-report',
    includeCharts: true,
    includeMetadata: true,
    dateRange: {
      start: new Date(Date.now() - 30 * 24 * 60 * 60 * 1000), // 30 days ago
      end: new Date()
    }
  });
  
  const [templates, setTemplates] = useState<ReportTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [exports, setExports] = useState<any[]>([]);

  // Mock templates - in production this would come from your API
  useEffect(() => {
    const mockTemplates: ReportTemplate[] = [
      {
        id: 'standard-report',
        name: 'Standard Investigation Report',
        description: 'Comprehensive investigation summary with all key findings',
        type: 'investigation',
        fields: ['summary', 'timeline', 'entities', 'transactions', 'risk-assessment', 'recommendations'],
        supportedFormats: ['pdf', 'excel', 'json']
      },
      {
        id: 'executive-summary',
        name: 'Executive Summary',
        description: 'High-level overview for management and stakeholders',
        type: 'investigation',
        fields: ['summary', 'key-findings', 'risk-assessment', 'recommendations'],
        supportedFormats: ['pdf', 'excel']
      },
      {
        id: 'compliance-sar',
        name: 'Suspicious Activity Report (SAR)',
        description: 'Regulatory SAR filing template',
        type: 'compliance',
        fields: ['entity-details', 'suspicious-activity', 'transaction-details', 'investigation-summary'],
        supportedFormats: ['pdf', 'json']
      },
      {
        id: 'transaction-analysis',
        name: 'Transaction Analysis Report',
        description: 'Detailed transaction pattern and flow analysis',
        type: 'analytics',
        fields: ['transaction-summary', 'patterns', 'flows', 'risk-indicators', 'visualizations'],
        supportedFormats: ['pdf', 'excel', 'csv']
      },
      {
        id: 'entity-profile',
        name: 'Entity Profile Report',
        description: 'Comprehensive entity information and risk profile',
        type: 'investigation',
        fields: ['entity-info', 'risk-profile', 'transaction-history', 'relationships', 'flags'],
        supportedFormats: ['pdf', 'excel', 'json']
      },
      {
        id: 'audit-trail',
        name: 'Audit Trail Report',
        description: 'Complete audit trail of investigation activities',
        type: 'audit',
        fields: ['activities', 'users', 'timestamps', 'changes', 'approvals'],
        supportedFormats: ['excel', 'csv', 'json']
      }
    ];

    setTemplates(mockTemplates);
  }, []);

  // Mock export history
  useEffect(() => {
    const mockExports = [
      {
        id: 'exp001',
        name: 'Investigation INV001 - Standard Report',
        format: 'pdf',
        createdAt: new Date(Date.now() - 2 * 60 * 60 * 1000),
        status: 'completed',
        size: '2.4 MB',
        downloadUrl: '/exports/exp001.pdf'
      },
      {
        id: 'exp002',
        name: 'Entity Profile - ENT123',
        format: 'excel',
        createdAt: new Date(Date.now() - 6 * 60 * 60 * 1000),
        status: 'completed',
        size: '1.8 MB',
        downloadUrl: '/exports/exp002.xlsx'
      },
      {
        id: 'exp003',
        name: 'SAR Filing - Entity ENT456',
        format: 'pdf',
        createdAt: new Date(Date.now() - 24 * 60 * 60 * 1000),
        status: 'processing',
        size: null,
        downloadUrl: null
      }
    ];

    setExports(mockExports);
  }, []);

  const handleExport = async () => {
    setLoading(true);
    
    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 2000));
      
      const newExport = {
        id: `exp${Date.now()}`,
        name: `${templates.find(t => t.id === exportOptions.template)?.name} - ${new Date().toISOString()}`,
        format: exportOptions.format,
        createdAt: new Date(),
        status: 'processing',
        size: null,
        downloadUrl: null
      };
      
      setExports(prev => [newExport, ...prev]);
      
      // Simulate processing completion
      setTimeout(() => {
        setExports(prev => 
          prev.map(exp => 
            exp.id === newExport.id 
              ? { ...exp, status: 'completed', size: '1.2 MB', downloadUrl: `/exports/${exp.id}.${exp.format}` }
              : exp
          )
        );
        
        if (onExportComplete) {
          onExportComplete(newExport.id);
        }
      }, 3000);
      
      setIsOpen(false);
    } catch (error) {
      console.error('Export failed:', error);
    } finally {
      setLoading(false);
    }
  };

  const selectedTemplate = templates.find(t => t.id === exportOptions.template);

  return (
    <div className="space-y-4">
      {/* Export Button */}
      <button
        onClick={() => setIsOpen(true)}
        className="inline-flex items-center px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
      >
        <Download className="h-4 w-4 mr-2" />
        Export Report
      </button>

      {/* Export History */}
      <div className="bg-white shadow rounded-lg">
        <div className="px-4 py-5 sm:p-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">Export History</h3>
          
          <div className="space-y-3">
            {exports.map(exportItem => (
              <div key={exportItem.id} className="flex items-center justify-between p-3 border rounded-lg">
                <div className="flex items-center space-x-3">
                  <FileText className="h-5 w-5 text-gray-400" />
                  <div>
                    <p className="text-sm font-medium text-gray-900">{exportItem.name}</p>
                    <p className="text-xs text-gray-500">
                      {exportItem.format.toUpperCase()} • {exportItem.createdAt.toLocaleString()}
                      {exportItem.size && ` • ${exportItem.size}`}
                    </p>
                  </div>
                </div>
                
                <div className="flex items-center space-x-2">
                  <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${
                    exportItem.status === 'completed' ? 'bg-green-100 text-green-800' :
                    exportItem.status === 'processing' ? 'bg-yellow-100 text-yellow-800' :
                    'bg-red-100 text-red-800'
                  }`}>
                    {exportItem.status}
                  </span>
                  
                  {exportItem.status === 'completed' && exportItem.downloadUrl && (
                    <button
                      onClick={() => window.open(exportItem.downloadUrl)}
                      className="text-blue-600 hover:text-blue-800 text-sm font-medium"
                    >
                      Download
                    </button>
                  )}
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Export Modal */}
      {isOpen && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex items-end justify-center min-h-screen pt-4 px-4 pb-20 text-center sm:block sm:p-0">
            <div className="fixed inset-0 bg-gray-500 bg-opacity-75" onClick={() => setIsOpen(false)} />
            
            <div className="inline-block align-bottom bg-white rounded-lg text-left overflow-hidden shadow-xl transform transition-all sm:my-8 sm:align-middle sm:max-w-lg sm:w-full">
              <div className="bg-white px-4 pt-5 pb-4 sm:p-6 sm:pb-4">
                <div className="sm:flex sm:items-start">
                  <div className="mt-3 text-center sm:mt-0 sm:text-left w-full">
                    <h3 className="text-lg leading-6 font-medium text-gray-900 mb-4">
                      Export Report
                    </h3>
                    
                    <div className="space-y-4">
                      {/* Template Selection */}
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Report Template
                        </label>
                        <select
                          value={exportOptions.template}
                          onChange={(e) => setExportOptions(prev => ({ ...prev, template: e.target.value }))}
                          className="w-full border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                        >
                          {templates.map(template => (
                            <option key={template.id} value={template.id}>
                              {template.name}
                            </option>
                          ))}
                        </select>
                        {selectedTemplate && (
                          <p className="text-xs text-gray-500 mt-1">{selectedTemplate.description}</p>
                        )}
                      </div>

                      {/* Format Selection */}
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Export Format
                        </label>
                        <div className="grid grid-cols-4 gap-2">
                          {selectedTemplate?.supportedFormats.map(format => (
                            <button
                              key={format}
                              onClick={() => setExportOptions(prev => ({ ...prev, format: format as any }))}
                              className={`px-3 py-2 text-xs font-medium rounded-md ${
                                exportOptions.format === format
                                  ? 'bg-blue-100 text-blue-800 border-blue-300'
                                  : 'bg-gray-100 text-gray-700 border-gray-300'
                              } border`}
                            >
                              {format.toUpperCase()}
                            </button>
                          ))}
                        </div>
                      </div>

                      {/* Date Range */}
                      <div>
                        <label className="block text-sm font-medium text-gray-700 mb-2">
                          Date Range
                        </label>
                        <div className="grid grid-cols-2 gap-2">
                          <input
                            type="date"
                            value={exportOptions.dateRange.start.toISOString().split('T')[0]}
                            onChange={(e) => setExportOptions(prev => ({
                              ...prev,
                              dateRange: { ...prev.dateRange, start: new Date(e.target.value) }
                            }))}
                            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                          <input
                            type="date"
                            value={exportOptions.dateRange.end.toISOString().split('T')[0]}
                            onChange={(e) => setExportOptions(prev => ({
                              ...prev,
                              dateRange: { ...prev.dateRange, end: new Date(e.target.value) }
                            }))}
                            className="border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          />
                        </div>
                      </div>

                      {/* Options */}
                      <div className="space-y-2">
                        <label className="flex items-center">
                          <input
                            type="checkbox"
                            checked={exportOptions.includeCharts}
                            onChange={(e) => setExportOptions(prev => ({ ...prev, includeCharts: e.target.checked }))}
                            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          />
                          <span className="ml-2 text-sm text-gray-700">Include charts and visualizations</span>
                        </label>
                        
                        <label className="flex items-center">
                          <input
                            type="checkbox"
                            checked={exportOptions.includeMetadata}
                            onChange={(e) => setExportOptions(prev => ({ ...prev, includeMetadata: e.target.checked }))}
                            className="rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                          />
                          <span className="ml-2 text-sm text-gray-700">Include metadata and audit trail</span>
                        </label>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              
              <div className="bg-gray-50 px-4 py-3 sm:px-6 sm:flex sm:flex-row-reverse">
                <button
                  onClick={handleExport}
                  disabled={loading}
                  className="w-full inline-flex justify-center rounded-md border border-transparent shadow-sm px-4 py-2 bg-blue-600 text-base font-medium text-white hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:ml-3 sm:w-auto sm:text-sm disabled:opacity-50"
                >
                  {loading ? (
                    <>
                      <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                      Exporting...
                    </>
                  ) : (
                    'Export'
                  )}
                </button>
                <button
                  onClick={() => setIsOpen(false)}
                  className="mt-3 w-full inline-flex justify-center rounded-md border border-gray-300 shadow-sm px-4 py-2 bg-white text-base font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 sm:mt-0 sm:ml-3 sm:w-auto sm:text-sm"
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default ExportManager;