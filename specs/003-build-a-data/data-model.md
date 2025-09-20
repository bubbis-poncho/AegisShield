# Data Model: Data Intelligence Platform

**Date**: 2025-09-20  
**Feature**: Data Intelligence and Investigation Platform  
**Purpose**: Define entities, relationships, and data structures for the platform

## Core Entities

### Person Entity
**Purpose**: Represents individuals in the investigation graph  
**Storage**: Neo4j nodes + PostgreSQL metadata  

**Attributes**:
- `person_id` (UUID): Unique identifier
- `names[]` (Array): Full names, aliases, nicknames
- `birth_date` (Date): Date of birth (if known)
- `identification_numbers[]` (Array): SSN, passport, driver's license
- `addresses[]` (Array): Current and historical addresses
- `phone_numbers[]` (Array): Contact numbers
- `email_addresses[]` (Array): Email contacts
- `risk_score` (Float): Computed risk level (0.0-1.0)
- `created_at` (Timestamp): Entity creation time
- `updated_at` (Timestamp): Last modification time

**Validation Rules**:
- At least one name or identifier required
- Phone numbers must follow E.164 format
- Email addresses must be valid format
- Risk score must be between 0.0 and 1.0

### Organization Entity
**Purpose**: Represents companies, banks, institutions  
**Storage**: Neo4j nodes + PostgreSQL metadata

**Attributes**:
- `organization_id` (UUID): Unique identifier
- `legal_names[]` (Array): Official names and DBAs
- `registration_numbers[]` (Array): Tax ID, DUNS, LEI codes
- `addresses[]` (Array): Business addresses
- `industry_codes[]` (Array): NAICS/SIC classifications
- `incorporation_date` (Date): Date of establishment
- `jurisdiction` (String): Incorporation country/state
- `status` (Enum): Active, Dissolved, Suspended
- `risk_score` (Float): Computed risk level (0.0-1.0)
- `created_at` (Timestamp): Entity creation time
- `updated_at` (Timestamp): Last modification time

### Transaction Entity
**Purpose**: Financial movements and activities  
**Storage**: PostgreSQL tables + Neo4j relationships

**Attributes**:
- `transaction_id` (UUID): Unique identifier
- `source_account_id` (UUID): Originating account
- `destination_account_id` (UUID): Receiving account
- `amount` (Decimal): Transaction amount
- `currency` (String): ISO 4217 currency code
- `transaction_type` (Enum): Transfer, Deposit, Withdrawal, Payment
- `transaction_date` (Timestamp): When transaction occurred
- `processing_date` (Timestamp): When bank processed
- `reference_number` (String): Bank reference
- `description` (Text): Transaction description
- `fees` (Decimal): Associated fees
- `exchange_rate` (Decimal): If currency conversion
- `status` (Enum): Pending, Completed, Failed, Reversed
- `risk_flags[]` (Array): Automated risk indicators
- `created_at` (Timestamp): Record creation time

**Validation Rules**:
- Amount must be positive
- Currency must be valid ISO code
- Source and destination accounts cannot be the same
- Transaction date cannot be in the future

### Account Entity
**Purpose**: Financial accounts linking entities to transactions  
**Storage**: PostgreSQL tables + Neo4j relationships

**Attributes**:
- `account_id` (UUID): Unique identifier
- `account_number` (String): Bank account number (encrypted)
- `account_type` (Enum): Checking, Savings, Credit, Investment
- `bank_identifier` (String): Routing number, SWIFT code
- `account_holder_id` (UUID): Reference to Person/Organization
- `opening_date` (Date): Account opening date
- `closing_date` (Date): Account closure date (if closed)
- `status` (Enum): Active, Closed, Frozen, Restricted
- `balance` (Decimal): Current balance (if available)
- `currency` (String): Account currency
- `jurisdiction` (String): Bank's country/region
- `created_at` (Timestamp): Record creation time
- `updated_at` (Timestamp): Last modification time

### Alert Entity
**Purpose**: Risk notifications from automated analysis  
**Storage**: PostgreSQL tables

**Attributes**:
- `alert_id` (UUID): Unique identifier
- `alert_type` (Enum): ML, Sanctions, Velocity, Structuring, Unusual
- `severity` (Enum): Low, Medium, High, Critical
- `confidence_score` (Float): Algorithm confidence (0.0-1.0)
- `title` (String): Human-readable alert title
- `description` (Text): Detailed alert description
- `entities_involved[]` (Array): Related person/organization IDs
- `transactions_involved[]` (Array): Related transaction IDs
- `risk_indicators[]` (Array): Specific risk factors detected
- `detection_time` (Timestamp): When alert was generated
- `investigation_status` (Enum): New, InProgress, Resolved, False_Positive
- `assigned_analyst_id` (UUID): Analyst handling the alert
- `resolution_notes` (Text): Investigation findings
- `resolved_at` (Timestamp): Resolution timestamp
- `created_at` (Timestamp): Alert creation time

### Investigation Entity
**Purpose**: Case management for analyst activities  
**Storage**: PostgreSQL tables

**Attributes**:
- `investigation_id` (UUID): Unique identifier
- `case_number` (String): Human-readable case identifier
- `title` (String): Investigation title
- `description` (Text): Case description and objectives
- `priority` (Enum): Low, Medium, High, Urgent
- `status` (Enum): Open, InProgress, Closed, Archived
- `created_by` (UUID): Analyst who created the case
- `assigned_to` (UUID): Primary assigned analyst
- `alerts[]` (Array): Related alert IDs
- `entities_of_interest[]` (Array): Key entities being investigated
- `hypotheses[]` (Array): Investigation hypotheses and status
- `findings` (Text): Key investigation findings
- `evidence_links[]` (Array): Supporting evidence references
- `compliance_tags[]` (Array): Regulatory requirements addressed
- `created_at` (Timestamp): Case creation time
- `updated_at` (Timestamp): Last activity timestamp
- `closed_at` (Timestamp): Case closure time

### Relationship Entity
**Purpose**: Connections between entities with metadata  
**Storage**: Neo4j relationships + PostgreSQL metadata

**Attributes**:
- `relationship_id` (UUID): Unique identifier
- `source_entity_id` (UUID): Source entity
- `target_entity_id` (UUID): Target entity
- `relationship_type` (Enum): Controls, Owns, EmployedBy, Related, Transacts, Guarantees
- `strength` (Float): Relationship strength indicator (0.0-1.0)
- `confidence` (Float): Confidence in relationship (0.0-1.0)
- `start_date` (Date): When relationship began
- `end_date` (Date): When relationship ended (if applicable)
- `evidence_sources[]` (Array): Data sources supporting relationship
- `notes` (Text): Additional relationship context
- `created_at` (Timestamp): Relationship discovery time
- `updated_at` (Timestamp): Last update time

## Entity Relationships

### Core Relationship Types

1. **Person ←→ Organization**
   - `EMPLOYED_BY`: Person works for organization
   - `CONTROLS`: Person has control over organization
   - `OWNS`: Person owns shares/interest in organization

2. **Person ←→ Account**
   - `ACCOUNT_HOLDER`: Person owns the account
   - `AUTHORIZED_USER`: Person can access the account
   - `BENEFICIARY`: Person benefits from the account

3. **Organization ←→ Account**
   - `ACCOUNT_HOLDER`: Organization owns the account
   - `CONTROLS`: Organization controls the account

4. **Transaction ←→ Account**
   - `SOURCE_ACCOUNT`: Transaction originates from account
   - `DESTINATION_ACCOUNT`: Transaction goes to account

5. **Alert ←→ Entity**
   - `INVOLVES`: Alert relates to specific entities
   - `TRIGGERED_BY`: Alert triggered by entity activity

6. **Investigation ←→ Entity**
   - `INVESTIGATES`: Investigation focuses on entities
   - `EVIDENCE_FOR`: Entity provides evidence for investigation

### Temporal Relationships

All relationships include temporal aspects:
- `valid_from`: When relationship became active
- `valid_to`: When relationship ended (NULL if ongoing)
- `confidence_over_time[]`: Array of confidence scores with timestamps

## Data Storage Strategy

### PostgreSQL Schema
```sql
-- Structured data, metadata, user management
-- Optimized for ACID transactions and reporting
-- Partitioned by date for transaction tables
-- Indexes on frequently queried fields
```

### Neo4j Schema
```cypher
// Graph relationships and entity connections
// Optimized for traversal queries and pattern matching
// Indexes on entity IDs and relationship types
// Constraints for data integrity
```

### Apache Doris Schema
```sql
-- Analytical workloads and aggregations
-- Star schema with fact and dimension tables
-- Columnar storage for analytical queries
-- Materialized views for common aggregations
```

### Data Flow Patterns

1. **Ingestion Flow**: Raw Data → Kafka → Processing Services → PostgreSQL/Neo4j
2. **Analytics Flow**: PostgreSQL → Kafka → Apache Doris (via Iceberg)
3. **Investigation Flow**: Neo4j ← API Gateway ← Frontend
4. **Alerting Flow**: All Stores → Alert Engine → PostgreSQL → Notification

## State Transitions

### Alert Lifecycle
```
New → InProgress → [Resolved | False_Positive]
                ↓
           [Escalated → Investigation]
```

### Investigation Lifecycle
```
Open → InProgress → [Closed | Archived]
                 ↓
           [Reopened → InProgress]
```

### Entity Resolution States
```
Unresolved → Candidate_Matches → [Merged | Distinct]
                              ↓
                         [Manual_Review]
```

**Next Phase**: Create API contracts and gRPC service definitions