import { test, expect, Page } from '@playwright/test'
import { faker } from '@faker-js/faker'

/**
 * End-to-End tests for investigation workflow
 * Tests complete user journey from authentication to case resolution
 */

// Test configuration
const config = {
  baseURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3000',
  apiURL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
  testTimeout: 30000,
  investigatorEmail: 'test.investigator@aegisshield.com',
  investigatorPassword: 'test_password_123',
  adminEmail: 'admin@aegisshield.com',
  adminPassword: 'admin_password_123'
}

// Test data generators
class TestDataGenerator {
  static generateCase() {
    return {
      title: `Investigation ${faker.company.name()} - ${faker.date.recent().toISOString().split('T')[0]}`,
      description: faker.lorem.paragraph(3),
      priority: faker.helpers.arrayElement(['low', 'medium', 'high', 'critical']),
      type: faker.helpers.arrayElement(['aml', 'fraud', 'sanctions', 'kyc', 'compliance']),
      assignee: config.investigatorEmail
    }
  }

  static generateEntity() {
    const entityType = faker.helpers.arrayElement(['person', 'organization', 'account', 'transaction'])
    
    switch (entityType) {
      case 'person':
        return {
          type: 'person',
          name: faker.person.fullName(),
          ssn: faker.string.numeric(9),
          email: faker.internet.email(),
          phone: faker.phone.number(),
          address: {
            street: faker.location.streetAddress(),
            city: faker.location.city(),
            state: faker.location.state(),
            zipCode: faker.location.zipCode(),
            country: faker.location.countryCode()
          }
        }
      case 'organization':
        return {
          type: 'organization',
          name: faker.company.name(),
          taxId: faker.string.numeric(9),
          industry: faker.company.buzzPhrase(),
          address: {
            street: faker.location.streetAddress(),
            city: faker.location.city(),
            state: faker.location.state(),
            zipCode: faker.location.zipCode(),
            country: faker.location.countryCode()
          }
        }
      case 'account':
        return {
          type: 'account',
          accountNumber: faker.finance.accountNumber(12),
          bankName: faker.company.name() + ' Bank',
          accountType: faker.helpers.arrayElement(['checking', 'savings', 'investment', 'loan']),
          balance: parseFloat(faker.finance.amount(1000, 100000, 2)),
          currency: 'USD'
        }
      case 'transaction':
        return {
          type: 'transaction',
          transactionId: faker.string.uuid(),
          amount: parseFloat(faker.finance.amount(100, 50000, 2)),
          currency: 'USD',
          date: faker.date.recent({ days: 30 }).toISOString(),
          description: faker.finance.transactionDescription(),
          status: faker.helpers.arrayElement(['completed', 'pending', 'failed'])
        }
      default:
        throw new Error(`Unknown entity type: ${entityType}`)
    }
  }

  static generateDocument() {
    return {
      name: `${faker.lorem.words(3).replace(/\\s/g, '_')}.pdf`,
      type: faker.helpers.arrayElement(['identity', 'financial', 'legal', 'compliance']),
      description: faker.lorem.sentence(),
      content: faker.lorem.paragraphs(5)
    }
  }

  static generateAlert() {
    return {
      type: faker.helpers.arrayElement(['sanctions_match', 'unusual_transaction', 'kyc_failure', 'suspicious_pattern']),
      severity: faker.helpers.arrayElement(['low', 'medium', 'high', 'critical']),
      description: faker.lorem.sentence(),
      details: faker.lorem.paragraph()
    }
  }
}

// Page Object Models
class LoginPage {
  constructor(private page: Page) {}

  async navigate() {
    await this.page.goto('/login')
  }

  async login(email: string, password: string) {
    await this.page.fill('[data-testid="email-input"]', email)
    await this.page.fill('[data-testid="password-input"]', password)
    await this.page.click('[data-testid="login-button"]')
    
    // Wait for navigation to dashboard
    await this.page.waitForURL('/dashboard')
  }

  async expectLoginError() {
    await expect(this.page.locator('[data-testid="login-error"]')).toBeVisible()
  }
}

class DashboardPage {
  constructor(private page: Page) {}

  async expectDashboardElements() {
    await expect(this.page.locator('[data-testid="dashboard-header"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="cases-overview"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="alerts-summary"]')).toBeVisible()
  }

  async navigateToInvestigations() {
    await this.page.click('[data-testid="nav-investigations"]')
    await this.page.waitForURL('/investigations')
  }

  async navigateToNewCase() {
    await this.page.click('[data-testid="create-case-button"]')
    await this.page.waitForURL('/investigations/new')
  }
}

class InvestigationsPage {
  constructor(private page: Page) {}

  async expectInvestigationsList() {
    await expect(this.page.locator('[data-testid="investigations-list"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="search-investigations"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="filter-investigations"]')).toBeVisible()
  }

  async searchInvestigations(query: string) {
    await this.page.fill('[data-testid="search-investigations"]', query)
    await this.page.press('[data-testid="search-investigations"]', 'Enter')
    
    // Wait for search results
    await this.page.waitForTimeout(1000)
  }

  async filterByStatus(status: string) {
    await this.page.click('[data-testid="filter-investigations"]')
    await this.page.click(`[data-testid="filter-status-${status}"]`)
  }

  async openCase(caseId: string) {
    await this.page.click(`[data-testid="case-${caseId}"]`)
    await this.page.waitForURL(`/investigations/${caseId}`)
  }
}

class CaseCreationPage {
  constructor(private page: Page) {}

  async createCase(caseData: any) {
    await this.page.fill('[data-testid="case-title"]', caseData.title)
    await this.page.fill('[data-testid="case-description"]', caseData.description)
    
    await this.page.click('[data-testid="case-priority"]')
    await this.page.click(`[data-testid="priority-${caseData.priority}"]`)
    
    await this.page.click('[data-testid="case-type"]')
    await this.page.click(`[data-testid="type-${caseData.type}"]`)
    
    await this.page.fill('[data-testid="case-assignee"]', caseData.assignee)
    
    await this.page.click('[data-testid="create-case-submit"]')
    
    // Wait for case creation and redirect
    await this.page.waitForURL(/\\/investigations\\/\\d+/)
  }
}

class CaseDetailPage {
  constructor(private page: Page) {}

  async expectCaseDetails() {
    await expect(this.page.locator('[data-testid="case-header"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="case-timeline"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="case-entities"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="case-documents"]')).toBeVisible()
  }

  async addEntity(entityData: any) {
    await this.page.click('[data-testid="add-entity-button"]')
    
    // Fill entity form based on type
    await this.page.click('[data-testid="entity-type"]')
    await this.page.click(`[data-testid="entity-type-${entityData.type}"]`)
    
    if (entityData.type === 'person') {
      await this.page.fill('[data-testid="entity-name"]', entityData.name)
      await this.page.fill('[data-testid="entity-ssn"]', entityData.ssn)
      await this.page.fill('[data-testid="entity-email"]', entityData.email)
      await this.page.fill('[data-testid="entity-phone"]', entityData.phone)
    } else if (entityData.type === 'organization') {
      await this.page.fill('[data-testid="entity-name"]', entityData.name)
      await this.page.fill('[data-testid="entity-tax-id"]', entityData.taxId)
      await this.page.fill('[data-testid="entity-industry"]', entityData.industry)
    } else if (entityData.type === 'account') {
      await this.page.fill('[data-testid="entity-account-number"]', entityData.accountNumber)
      await this.page.fill('[data-testid="entity-bank-name"]', entityData.bankName)
      await this.page.fill('[data-testid="entity-balance"]', entityData.balance.toString())
    } else if (entityData.type === 'transaction') {
      await this.page.fill('[data-testid="entity-transaction-id"]', entityData.transactionId)
      await this.page.fill('[data-testid="entity-amount"]', entityData.amount.toString())
      await this.page.fill('[data-testid="entity-description"]', entityData.description)
    }
    
    await this.page.click('[data-testid="add-entity-submit"]')
    
    // Wait for entity to be added
    await this.page.waitForTimeout(1000)
  }

  async uploadDocument(documentData: any) {
    await this.page.click('[data-testid="upload-document-button"]')
    
    // Create a temporary file for upload simulation
    const fileContent = documentData.content
    await this.page.setInputFiles('[data-testid="document-upload"]', {
      name: documentData.name,
      mimeType: 'application/pdf',
      buffer: Buffer.from(fileContent)
    })
    
    await this.page.fill('[data-testid="document-description"]', documentData.description)
    
    await this.page.click('[data-testid="document-type"]')
    await this.page.click(`[data-testid="document-type-${documentData.type}"]`)
    
    await this.page.click('[data-testid="upload-document-submit"]')
    
    // Wait for upload completion
    await this.page.waitForTimeout(2000)
  }

  async openGraphExplorer() {
    await this.page.click('[data-testid="open-graph-explorer"]')
    await this.page.waitForURL(/.*graph-explorer.*/)
  }

  async addAlert(alertData: any) {
    await this.page.click('[data-testid="add-alert-button"]')
    
    await this.page.click('[data-testid="alert-type"]')
    await this.page.click(`[data-testid="alert-type-${alertData.type}"]`)
    
    await this.page.click('[data-testid="alert-severity"]')
    await this.page.click(`[data-testid="alert-severity-${alertData.severity}"]`)
    
    await this.page.fill('[data-testid="alert-description"]', alertData.description)
    await this.page.fill('[data-testid="alert-details"]', alertData.details)
    
    await this.page.click('[data-testid="add-alert-submit"]')
    
    // Wait for alert to be added
    await this.page.waitForTimeout(1000)
  }

  async updateCaseStatus(status: string) {
    await this.page.click('[data-testid="case-status-dropdown"]')
    await this.page.click(`[data-testid="status-${status}"]`)
    
    // Wait for status update
    await this.page.waitForTimeout(1000)
  }

  async addTimelineEntry(entry: string) {
    await this.page.fill('[data-testid="timeline-entry"]', entry)
    await this.page.click('[data-testid="add-timeline-entry"]')
    
    // Wait for entry to be added
    await this.page.waitForTimeout(1000)
  }
}

class GraphExplorerPage {
  constructor(private page: Page) {}

  async expectGraphElements() {
    await expect(this.page.locator('[data-testid="graph-canvas"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="graph-controls"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="graph-filters"]')).toBeVisible()
    await expect(this.page.locator('[data-testid="graph-legend"]')).toBeVisible()
  }

  async searchEntity(query: string) {
    await this.page.fill('[data-testid="graph-search"]', query)
    await this.page.press('[data-testid="graph-search"]', 'Enter')
    
    // Wait for search results
    await this.page.waitForTimeout(2000)
  }

  async applyFilter(filterType: string, value: string) {
    await this.page.click(`[data-testid="filter-${filterType}"]`)
    await this.page.click(`[data-testid="filter-${filterType}-${value}"]`)
    
    // Wait for filter to be applied
    await this.page.waitForTimeout(1000)
  }

  async expandNode(nodeId: string) {
    await this.page.click(`[data-testid="node-${nodeId}"]`)
    await this.page.click('[data-testid="expand-node"]')
    
    // Wait for expansion
    await this.page.waitForTimeout(2000)
  }

  async exportGraph(format: string) {
    await this.page.click('[data-testid="export-graph"]')
    await this.page.click(`[data-testid="export-${format}"]`)
    
    // Wait for export
    await this.page.waitForTimeout(1000)
  }

  async returnToCase() {
    await this.page.click('[data-testid="return-to-case"]')
    await this.page.waitForURL(/\\/investigations\\/\\d+/)
  }
}

// Test Suite
test.describe('Investigation Workflow E2E Tests', () => {
  let loginPage: LoginPage
  let dashboardPage: DashboardPage
  let investigationsPage: InvestigationsPage
  let caseCreationPage: CaseCreationPage
  let caseDetailPage: CaseDetailPage
  let graphExplorerPage: GraphExplorerPage

  test.beforeEach(async ({ page }) => {
    // Initialize page objects
    loginPage = new LoginPage(page)
    dashboardPage = new DashboardPage(page)
    investigationsPage = new InvestigationsPage(page)
    caseCreationPage = new CaseCreationPage(page)
    caseDetailPage = new CaseDetailPage(page)
    graphExplorerPage = new GraphExplorerPage(page)

    // Set longer timeout for E2E tests
    test.setTimeout(config.testTimeout)
  })

  test('should authenticate investigator and access dashboard', async () => {
    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.expectDashboardElements()
  })

  test('should reject invalid login credentials', async () => {
    await loginPage.navigate()
    await loginPage.login('invalid@email.com', 'wrong_password')
    await loginPage.expectLoginError()
  })

  test('should create new investigation case', async () => {
    const caseData = TestDataGenerator.generateCase()

    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToNewCase()
    await caseCreationPage.createCase(caseData)
    await caseDetailPage.expectCaseDetails()
  })

  test('should search and filter investigations', async () => {
    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToInvestigations()
    await investigationsPage.expectInvestigationsList()
    
    // Test search functionality
    await investigationsPage.searchInvestigations('AML')
    
    // Test filter functionality
    await investigationsPage.filterByStatus('active')
  })

  test('should complete full investigation workflow', async () => {
    const caseData = TestDataGenerator.generateCase()
    const entityData = TestDataGenerator.generateEntity()
    const documentData = TestDataGenerator.generateDocument()
    const alertData = TestDataGenerator.generateAlert()

    // Step 1: Create case
    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToNewCase()
    await caseCreationPage.createCase(caseData)
    await caseDetailPage.expectCaseDetails()

    // Step 2: Add entities
    await caseDetailPage.addEntity(entityData)

    // Step 3: Upload documents
    await caseDetailPage.uploadDocument(documentData)

    // Step 4: Add alerts
    await caseDetailPage.addAlert(alertData)

    // Step 5: Add timeline entry
    await caseDetailPage.addTimelineEntry('Investigation started with initial entity analysis')

    // Step 6: Explore graph
    await caseDetailPage.openGraphExplorer()
    await graphExplorerPage.expectGraphElements()
    await graphExplorerPage.searchEntity(entityData.name || entityData.accountNumber || entityData.transactionId)
    await graphExplorerPage.applyFilter('entityType', entityData.type)
    await graphExplorerPage.returnToCase()

    // Step 7: Update case status
    await caseDetailPage.updateCaseStatus('under_review')

    // Step 8: Complete investigation
    await caseDetailPage.addTimelineEntry('Investigation completed - no violations found')
    await caseDetailPage.updateCaseStatus('closed')
  })

  test('should handle graph exploration workflow', async () => {
    const caseData = TestDataGenerator.generateCase()
    const personData = TestDataGenerator.generateEntity()
    const orgData = TestDataGenerator.generateEntity()

    // Create case with multiple entities
    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToNewCase()
    await caseCreationPage.createCase(caseData)
    
    await caseDetailPage.addEntity(personData)
    await caseDetailPage.addEntity(orgData)

    // Explore graph relationships
    await caseDetailPage.openGraphExplorer()
    await graphExplorerPage.expectGraphElements()
    
    // Test various graph operations
    if (personData.name) {
      await graphExplorerPage.searchEntity(personData.name)
    }
    
    await graphExplorerPage.applyFilter('entityType', 'person')
    await graphExplorerPage.applyFilter('riskLevel', 'medium')
    
    // Test export functionality
    await graphExplorerPage.exportGraph('png')
    await graphExplorerPage.exportGraph('json')
    
    await graphExplorerPage.returnToCase()
  })

  test('should handle document management workflow', async () => {
    const caseData = TestDataGenerator.generateCase()
    const documents = [
      TestDataGenerator.generateDocument(),
      TestDataGenerator.generateDocument(),
      TestDataGenerator.generateDocument()
    ]

    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToNewCase()
    await caseCreationPage.createCase(caseData)

    // Upload multiple documents
    for (const documentData of documents) {
      await caseDetailPage.uploadDocument(documentData)
    }

    // Verify documents are listed
    for (const documentData of documents) {
      await expect(this.page.locator(`[data-testid="document-${documentData.name}"]`)).toBeVisible()
    }
  })

  test('should handle alert management workflow', async () => {
    const caseData = TestDataGenerator.generateCase()
    const alerts = [
      TestDataGenerator.generateAlert(),
      TestDataGenerator.generateAlert(),
      TestDataGenerator.generateAlert()
    ]

    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToNewCase()
    await caseCreationPage.createCase(caseData)

    // Add multiple alerts
    for (const alertData of alerts) {
      await caseDetailPage.addAlert(alertData)
    }

    // Verify alerts are displayed with correct severity
    for (const alertData of alerts) {
      await expect(this.page.locator(`[data-testid="alert-${alertData.type}"]`)).toBeVisible()
      await expect(this.page.locator(`[data-testid="alert-severity-${alertData.severity}"]`)).toBeVisible()
    }
  })

  test('should maintain case timeline throughout workflow', async () => {
    const caseData = TestDataGenerator.generateCase()
    const entityData = TestDataGenerator.generateEntity()

    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.navigateToNewCase()
    await caseCreationPage.createCase(caseData)

    // Timeline should show case creation
    await expect(this.page.locator('[data-testid="timeline-case-created"]')).toBeVisible()

    // Add entity and verify timeline entry
    await caseDetailPage.addEntity(entityData)
    await expect(this.page.locator('[data-testid="timeline-entity-added"]')).toBeVisible()

    // Add custom timeline entries
    const timelineEntries = [
      'Initial review completed',
      'Additional evidence requested',
      'Stakeholder consultation scheduled',
      'Final analysis completed'
    ]

    for (const entry of timelineEntries) {
      await caseDetailPage.addTimelineEntry(entry)
      await expect(this.page.locator(`text=${entry}`)).toBeVisible()
    }

    // Update status and verify timeline
    await caseDetailPage.updateCaseStatus('resolved')
    await expect(this.page.locator('[data-testid="timeline-status-updated"]')).toBeVisible()
  })
})

// Accessibility and Performance Tests
test.describe('Investigation Workflow Accessibility and Performance', () => {
  test('should meet accessibility standards', async ({ page }) => {
    const loginPage = new LoginPage(page)
    const dashboardPage = new DashboardPage(page)

    await loginPage.navigate()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    await dashboardPage.expectDashboardElements()

    // Check for accessibility violations
    await expect(page.locator('button')).toHaveAttribute('type')
    await expect(page.locator('input')).toHaveAttribute('aria-label')
    await expect(page.locator('[role="button"]')).toBeVisible()
  })

  test('should load pages within performance thresholds', async ({ page }) => {
    const loginPage = new LoginPage(page)

    // Measure page load times
    const startTime = Date.now()
    await loginPage.navigate()
    const loadTime = Date.now() - startTime

    expect(loadTime).toBeLessThan(3000) // 3 second threshold

    const loginStartTime = Date.now()
    await loginPage.login(config.investigatorEmail, config.investigatorPassword)
    const loginTime = Date.now() - loginStartTime

    expect(loginTime).toBeLessThan(5000) // 5 second threshold for authentication
  })
})