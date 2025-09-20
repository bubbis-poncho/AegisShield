# AegisShield Git Workflow

## Branching Strategy

### Main Branches
- `main` - Production-ready code, protected branch
- `develop` - Integration branch for features, CI/CD testing

### Feature Branches
- `feature/task-XXX-description` - Individual task implementation
- `hotfix/critical-fix-description` - Emergency production fixes
- `release/vX.X.X` - Release preparation branches

## Workflow Process

### 1. Feature Development
```bash
# Start from develop
git checkout develop
git pull origin develop

# Create feature branch for specific task
git checkout -b feature/task-001-git-setup

# Work on task, commit frequently
git add .
git commit -m "T001: Add comprehensive .gitignore for multi-language stack"

# Push and create PR
git push origin feature/task-001-git-setup
```

### 2. Pull Request Requirements
- [ ] Branch is up-to-date with `develop`
- [ ] All CI/CD checks pass (build, test, security scan)
- [ ] Code review approval from team lead
- [ ] Constitutional compliance verified
- [ ] Documentation updated if needed

### 3. Commit Message Format
```
TXXX: Brief description of change

Detailed explanation of what was implemented and why.
References constitutional principle if applicable.

Closes #issue-number
```

### 4. Release Process
```bash
# Create release branch from develop
git checkout develop
git checkout -b release/v1.0.0

# Finalize version, update docs
# Test release candidate
# Merge to main and tag
git checkout main
git merge release/v1.0.0
git tag -a v1.0.0 -m "Version 1.0.0: Initial release"
git push origin main --tags

# Merge back to develop
git checkout develop
git merge main
```

## Constitutional Compliance

Every commit must adhere to our constitutional principles:

1. **Data Integrity**: Include data validation tests
2. **Scalability**: Consider performance implications  
3. **Modular Code**: Maintain clean interfaces
4. **Comprehensive Testing**: Include automated tests
5. **Consistent UX**: Follow design system patterns

## Branch Protection Rules

### Main Branch
- Require pull request reviews (1 reviewer minimum)
- Require status checks to pass
- Require branches to be up to date
- Restrict pushes to administrators only
- Require signed commits

### Develop Branch  
- Require pull request reviews
- Require status checks to pass
- Allow force pushes by administrators

## Emergency Procedures

### Hotfix Process
```bash
# Critical production fix
git checkout main
git checkout -b hotfix/security-patch-cve-2024-001

# Fix, test, commit
git commit -m "HOTFIX: Patch critical security vulnerability CVE-2024-001"

# Merge to main and develop immediately
git checkout main
git merge hotfix/security-patch-cve-2024-001
git tag -a v1.0.1 -m "Hotfix v1.0.1: Security patch"

git checkout develop  
git merge hotfix/security-patch-cve-2024-001
```

## Local Development Setup

```bash
# Clone repository
git clone https://github.com/your-org/aegisshield.git
cd aegisshield

# Set up Git hooks for constitutional compliance
cp .githooks/* .git/hooks/
chmod +x .git/hooks/*

# Configure Git for project
git config core.autocrlf false
git config pull.rebase false
git config branch.autosetupmerge always
```