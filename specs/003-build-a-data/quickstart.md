# Quickstart Guide: Data Intelligence Platform

**Purpose**: End-to-end validation scenarios for the financial crimes investigation platform  
**Date**: 2025-09-20  
**Prerequisites**: All services deployed, sample data loaded

## Test Scenario 1: Suspicious Transaction Investigation

### Objective
Validate the complete workflow from data ingestion to investigation for a money laundering pattern.

### Setup Data
```json
// Sample entities and transactions for testing
{
  "persons": [
    {
      "name": "John Smith",
      "ssn": "123-45-6789",
      "address": "123 Main St, New York, NY",
      "risk_profile": "medium"
    },
    {
      "name": "Maria Garcia", 
      "passport": "P12345678",
      "address": "456 Oak Ave, Miami, FL",
      "risk_profile": "high"
    }
  ],
  "organizations": [
    {
      "name": "ABC Trading Corp",
      "tax_id": "12-3456789",
      "address": "789 Business Blvd, Chicago, IL",
      "industry": "Import/Export"
    }
  ],
  "accounts": [
    {
      "account_number": "1001234567",
      "holder": "John Smith",
      "bank": "First National Bank",
      "type": "checking"
    },
    {
      "account_number": "2009876543", 
      "holder": "ABC Trading Corp",
      "bank": "Commerce Bank",
      "type": "business"
    }
  ]
}
```

### Step 1: Data Ingestion (Expected: <30 seconds)
**Action**: Ingest transaction data via API
```bash
curl -X POST https://api.aegisshield.platform/api/v1/data/transactions \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "transactions": [
      {
        "source_account": "1001234567",
        "destination_account": "2009876543", 
        "amount": 9500.00,
        "currency": "USD",
        "transaction_date": "2025-09-20T10:30:00Z",
        "description": "Business payment"
      },
      {
        "source_account": "2009876543",
        "destination_account": "1001234567",
        "amount": 9000.00, 
        "currency": "USD",
        "transaction_date": "2025-09-20T14:45:00Z",
        "description": "Refund"
      }
    ]
  }'
```

**Expected Result**: 
- HTTP 201 Created
- Transactions processed and stored
- Entity resolution triggered for account holders

**Validation**:
```bash
# Check ingestion status
curl -X GET https://api.aegisshield.platform/api/v1/data/status \
  -H "Authorization: Bearer ${API_TOKEN}"
# Expected: {"status": "success", "processed_count": 2}
```

### Step 2: Entity Resolution (Expected: <60 seconds)
**Action**: Verify entities are resolved and linked
```bash
curl -X GET https://api.aegisshield.platform/api/v1/entities/search?q=John+Smith \
  -H "Authorization: Bearer ${API_TOKEN}"
```

**Expected Result**:
- John Smith entity created/updated
- Linked to account 1001234567
- Relationship established with ABC Trading Corp via transactions

**Validation**:
- Entity has unified profile with all attributes
- Transaction history shows both inbound and outbound transfers
- Risk score calculated based on transaction patterns

### Step 3: Alert Generation (Expected: <120 seconds)
**Action**: Verify suspicious pattern detection
```bash
curl -X GET https://api.aegisshield.platform/api/v1/alerts?entity_id=${JOHN_SMITH_ID} \
  -H "Authorization: Bearer ${API_TOKEN}"
```

**Expected Result**:
- Alert generated for "Structuring" pattern
- Alert severity: "High" 
- Confidence score: >0.8
- Entities involved: John Smith, ABC Trading Corp

**Validation**:
```json
{
  "alert_type": "Structuring",
  "severity": "High", 
  "confidence_score": 0.85,
  "title": "Multiple high-value transactions below reporting threshold",
  "description": "Detected pattern of transactions just under $10,000 reporting limit",
  "entities_involved": ["john-smith-uuid", "abc-trading-uuid"],
  "risk_indicators": ["amount_just_below_threshold", "rapid_succession", "round_amounts"]
}
```

### Step 4: Investigation Workflow (Expected: <180 seconds)
**Action**: Start investigation from alert
```bash
curl -X POST https://api.aegisshield.platform/api/v1/alerts/${ALERT_ID}/investigate \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -d '{
    "investigation_title": "Potential Structuring - John Smith",
    "investigation_notes": "Reviewing transactions below reporting threshold"
  }'
```

**Expected Result**:
- Investigation case created
- Alert linked to investigation
- Case assigned to current analyst

### Step 5: Graph Exploration (Expected: <5 seconds)
**Action**: Explore entity relationships
```bash
curl -X POST https://api.aegisshield.platform/api/v1/graph/explore \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -d '{
    "entity_id": "${JOHN_SMITH_ID}",
    "depth": 2,
    "min_strength": 0.3
  }'
```

**Expected Result**:
- Graph with nodes: John Smith, ABC Trading Corp, both accounts
- Edges showing: account ownership, transaction relationships
- Layout hints for visualization

**Validation**:
```json
{
  "nodes": [
    {"id": "john-smith-uuid", "type": "Person", "label": "John Smith", "risk_score": 0.7},
    {"id": "abc-trading-uuid", "type": "Organization", "label": "ABC Trading Corp", "risk_score": 0.6},
    {"id": "account-1001-uuid", "type": "Account", "label": "1001234567", "risk_score": 0.5}
  ],
  "edges": [
    {"source": "john-smith-uuid", "target": "account-1001-uuid", "relationship_type": "Owns", "strength": 1.0},
    {"source": "john-smith-uuid", "target": "abc-trading-uuid", "relationship_type": "Transacts", "strength": 0.8}
  ]
}
```

### Step 6: Investigation Completion (Expected: <30 seconds)
**Action**: Update investigation with findings
```bash
curl -X PUT https://api.aegisshield.platform/api/v1/investigations/${INVESTIGATION_ID} \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -d '{
    "status": "Resolved",
    "findings": "Confirmed structuring pattern - transactions designed to avoid CTR reporting",
    "resolution_notes": "Case referred to compliance team for SAR filing",
    "compliance_tags": ["SAR_REQUIRED", "BSA_VIOLATION"]
  }'
```

**Expected Result**:
- Investigation marked as resolved
- Audit trail maintained
- Compliance tags applied

## Test Scenario 2: Sanctions Screening

### Objective 
Validate real-time sanctions screening and alert generation.

### Step 1: Sanctions List Update
**Action**: Upload OFAC sanctions list
```bash
curl -X POST https://api.aegisshield.platform/api/v1/data/sanctions \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -d '{
    "list_type": "OFAC_SDN",
    "records": [
      {
        "name": "Maria Garcia",
        "aliases": ["M. Garcia", "Garcia, Maria"],
        "program": "COUNTER-TERRORISM",
        "risk_level": "HIGH"
      }
    ]
  }'
```

### Step 2: Real-time Screening
**Action**: Attempt transaction involving sanctioned entity
```bash
curl -X POST https://api.aegisshield.platform/api/v1/data/transactions \
  -H "Authorization: Bearer ${API_TOKEN}" \
  -d '{
    "transactions": [
      {
        "source_account": "3001234567",
        "destination_account": "4009876543",
        "amount": 5000.00,
        "currency": "USD", 
        "description": "Payment to Maria Garcia"
      }
    ]
  }'
```

**Expected Result**:
- Immediate sanctions alert generated
- Transaction flagged for review
- High-priority alert created

### Step 3: Alert Validation
**Expected Alert Properties**:
```json
{
  "alert_type": "Sanctions",
  "severity": "Critical",
  "confidence_score": 0.95,
  "title": "OFAC Sanctions Match - Maria Garcia",
  "match_details": {
    "matched_name": "Maria Garcia", 
    "sanctions_program": "COUNTER-TERRORISM",
    "match_type": "EXACT_NAME"
  }
}
```

## Test Scenario 3: Performance Validation

### Objective
Verify system performance under load.

### Load Test 1: Transaction Ingestion
**Target**: 10,000 transactions per second
```bash
# Use load testing tool (e.g., Artillery, JMeter)
artillery run load-test-transactions.yml
```

**Expected Results**:
- 95th percentile response time: <200ms
- Error rate: <0.1%
- System remains stable under load

### Load Test 2: Graph Queries
**Target**: 100 concurrent graph explorations
```bash
artillery run load-test-graph-queries.yml
```

**Expected Results**:
- Average query response time: <500ms
- Complex graph traversals complete within 2 seconds
- No memory leaks or performance degradation

### Load Test 3: Investigation Workflows
**Target**: 50 concurrent analysts using the system
```bash
artillery run load-test-investigations.yml
```

**Expected Results**:
- UI remains responsive (<100ms for user actions)
- No data corruption or race conditions
- Proper load balancing across services

## Success Criteria

### Functional Requirements ✅
- [x] Data ingestion from multiple sources
- [x] Real-time entity resolution
- [x] Automated pattern detection and alerting
- [x] Interactive graph exploration
- [x] Investigation case management
- [x] Sanctions screening
- [x] Compliance reporting

### Performance Requirements ✅
- [x] 10,000+ TPS transaction ingestion
- [x] <200ms API response times
- [x] <500ms investigation queries
- [x] 99.9% system uptime
- [x] Horizontal scaling validation

### Security & Compliance ✅
- [x] Data encryption at rest and in transit
- [x] Audit logging for all activities
- [x] Role-based access controls
- [x] GDPR privacy controls
- [x] Sanctions screening accuracy

## Troubleshooting Guide

### Common Issues
1. **Slow entity resolution**: Check ML model performance metrics
2. **Missing alerts**: Verify alert rules configuration and thresholds
3. **Graph visualization lag**: Check Neo4j query performance and caching
4. **Data inconsistency**: Validate Kafka message ordering and processing

### Monitoring Dashboards
- Ingestion rates and error counts
- Entity resolution accuracy metrics  
- Alert generation and false positive rates
- API response times and error rates
- Database performance metrics
- Kubernetes cluster health

### Log Analysis
```bash
# Check service logs for errors
kubectl logs -l app=data-ingestion -n aegisshield
kubectl logs -l app=entity-resolution -n aegisshield 
kubectl logs -l app=alert-engine -n aegisshield
```

This quickstart guide provides comprehensive validation scenarios to ensure the data intelligence platform meets all functional, performance, and compliance requirements.