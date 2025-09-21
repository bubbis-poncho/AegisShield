#!/usr/bin/env python3
"""
Comprehensive security audit for AegisShield platform
Includes penetration testing, vulnerability scanning, compliance validation
"""

import subprocess
import requests
import json
import os
import sys
import time
import re
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
import yaml
from pathlib import Path

@dataclass
class SecurityAuditConfig:
    """Configuration for security audit"""
    api_base_url: str = "http://localhost:8080"
    frontend_base_url: str = "http://localhost:3000"
    
    # Audit scope
    include_network_scan: bool = True
    include_vulnerability_scan: bool = True
    include_dependency_scan: bool = True
    include_code_analysis: bool = True
    include_compliance_check: bool = True
    include_penetration_test: bool = True
    
    # Tool configurations
    nmap_enabled: bool = False  # Requires nmap installation
    nikto_enabled: bool = False  # Requires nikto installation
    sqlmap_enabled: bool = False  # Requires sqlmap installation
    
    # Report output
    output_dir: str = "security_audit_results"
    
@dataclass
class SecurityFinding:
    """Security audit finding"""
    severity: str  # critical, high, medium, low, info
    category: str  # vulnerability, compliance, configuration, etc.
    title: str
    description: str
    impact: str
    recommendation: str
    evidence: Dict[str, Any]
    cve_id: Optional[str] = None
    cvss_score: Optional[float] = None

class SecurityAuditor:
    """Comprehensive security auditor for AegisShield platform"""
    
    def __init__(self, config: SecurityAuditConfig):
        self.config = config
        self.findings = []
        self.setup_output_directory()
        
    def setup_output_directory(self):
        """Create output directory for audit results"""
        Path(self.config.output_dir).mkdir(parents=True, exist_ok=True)
        
    def add_finding(self, finding: SecurityFinding):
        """Add a security finding to the audit results"""
        self.findings.append(finding)
        
    def run_dependency_vulnerability_scan(self) -> List[SecurityFinding]:
        """Scan dependencies for known vulnerabilities"""
        print("Running dependency vulnerability scan...")
        findings = []
        
        try:
            # Scan Go modules
            go_mod_files = list(Path(".").rglob("go.mod"))
            for go_mod in go_mod_files:
                print(f"Scanning Go dependencies in {go_mod.parent}")
                
                # Use go list to check for vulnerabilities
                try:
                    result = subprocess.run(
                        ["go", "list", "-m", "-versions", "all"],
                        cwd=go_mod.parent,
                        capture_output=True,
                        text=True,
                        timeout=60
                    )
                    
                    if result.returncode == 0:
                        # Check for known vulnerable packages
                        vulnerable_packages = self.check_go_vulnerabilities(result.stdout)
                        for vuln in vulnerable_packages:
                            findings.append(vuln)
                            
                except subprocess.TimeoutExpired:
                    findings.append(SecurityFinding(
                        severity="medium",
                        category="scan_error",
                        title="Go Dependency Scan Timeout",
                        description=f"Dependency scan timed out for {go_mod.parent}",
                        impact="Cannot verify security of Go dependencies",
                        recommendation="Manually review Go dependencies for vulnerabilities",
                        evidence={"module_path": str(go_mod.parent)}
                    ))
                except Exception as e:
                    findings.append(SecurityFinding(
                        severity="low",
                        category="scan_error", 
                        title="Go Dependency Scan Error",
                        description=f"Failed to scan Go dependencies: {str(e)}",
                        impact="Cannot verify security of Go dependencies",
                        recommendation="Manually review Go dependencies or fix scan environment",
                        evidence={"error": str(e), "module_path": str(go_mod.parent)}
                    ))
            
            # Scan Node.js dependencies
            package_json_files = list(Path(".").rglob("package.json"))
            for package_json in package_json_files:
                print(f"Scanning Node.js dependencies in {package_json.parent}")
                
                try:
                    # Use npm audit
                    result = subprocess.run(
                        ["npm", "audit", "--json"],
                        cwd=package_json.parent,
                        capture_output=True,
                        text=True,
                        timeout=120
                    )
                    
                    if result.stdout:
                        audit_data = json.loads(result.stdout)
                        npm_findings = self.parse_npm_audit(audit_data, package_json.parent)
                        findings.extend(npm_findings)
                        
                except subprocess.TimeoutExpired:
                    findings.append(SecurityFinding(
                        severity="medium",
                        category="scan_error",
                        title="NPM Audit Timeout",
                        description=f"NPM audit timed out for {package_json.parent}",
                        impact="Cannot verify security of Node.js dependencies",
                        recommendation="Manually run npm audit or update npm",
                        evidence={"package_path": str(package_json.parent)}
                    ))
                except Exception as e:
                    findings.append(SecurityFinding(
                        severity="low",
                        category="scan_error",
                        title="NPM Audit Error", 
                        description=f"Failed to run npm audit: {str(e)}",
                        impact="Cannot verify security of Node.js dependencies",
                        recommendation="Manually review Node.js dependencies",
                        evidence={"error": str(e), "package_path": str(package_json.parent)}
                    ))
            
            # Scan Python dependencies
            requirements_files = list(Path(".").rglob("requirements*.txt")) + list(Path(".").rglob("pyproject.toml"))
            for req_file in requirements_files:
                print(f"Scanning Python dependencies in {req_file}")
                
                try:
                    # Use safety or pip-audit if available
                    result = subprocess.run(
                        ["python", "-m", "pip", "list", "--format=json"],
                        capture_output=True,
                        text=True,
                        timeout=60
                    )
                    
                    if result.returncode == 0:
                        packages = json.loads(result.stdout)
                        python_findings = self.check_python_vulnerabilities(packages)
                        findings.extend(python_findings)
                        
                except Exception as e:
                    findings.append(SecurityFinding(
                        severity="low",
                        category="scan_error",
                        title="Python Dependency Scan Error",
                        description=f"Failed to scan Python dependencies: {str(e)}",
                        impact="Cannot verify security of Python dependencies", 
                        recommendation="Manually review Python dependencies",
                        evidence={"error": str(e), "requirements_file": str(req_file)}
                    ))
                    
        except Exception as e:
            findings.append(SecurityFinding(
                severity="medium",
                category="scan_error",
                title="Dependency Scan Failed",
                description=f"Dependency vulnerability scan failed: {str(e)}",
                impact="Cannot verify security of project dependencies",
                recommendation="Manually review all project dependencies for vulnerabilities",
                evidence={"error": str(e)}
            ))
            
        return findings
    
    def check_go_vulnerabilities(self, go_list_output: str) -> List[SecurityFinding]:
        """Check Go dependencies for known vulnerabilities"""
        findings = []
        
        # Known vulnerable Go packages (simplified check)
        vulnerable_patterns = [
            (r"golang\.org/x/text.*v0\.[0-3]\.", "CVE-2021-38561", "Path traversal vulnerability"),
            (r"github\.com/gin-gonic/gin.*v1\.[0-6]\.", "CVE-2020-28483", "Path traversal vulnerability"),
            (r"gopkg\.in/yaml\.v2.*v2\.[0-2]\.", "CVE-2019-11254", "Billion laughs attack")
        ]
        
        for pattern, cve, description in vulnerable_patterns:
            if re.search(pattern, go_list_output, re.IGNORECASE):
                findings.append(SecurityFinding(
                    severity="high",
                    category="vulnerable_dependency",
                    title=f"Vulnerable Go Package: {cve}",
                    description=description,
                    impact="Potential security vulnerability in application",
                    recommendation="Update to latest secure version of the package",
                    evidence={"cve": cve, "pattern": pattern},
                    cve_id=cve
                ))
                
        return findings
    
    def parse_npm_audit(self, audit_data: Dict, package_path: Path) -> List[SecurityFinding]:
        """Parse npm audit results and convert to security findings"""
        findings = []
        
        try:
            if "vulnerabilities" in audit_data:
                for vuln_name, vuln_info in audit_data["vulnerabilities"].items():
                    severity = vuln_info.get("severity", "unknown")
                    
                    findings.append(SecurityFinding(
                        severity=severity if severity in ["critical", "high", "medium", "low"] else "medium",
                        category="vulnerable_dependency",
                        title=f"Vulnerable NPM Package: {vuln_name}",
                        description=vuln_info.get("title", "NPM package vulnerability"),
                        impact=f"Vulnerable dependency in {package_path}",
                        recommendation=f"Update to version {vuln_info.get('fixAvailable', 'latest')}",
                        evidence={
                            "package": vuln_name,
                            "current_version": vuln_info.get("currentVersion", "unknown"),
                            "vulnerable_versions": vuln_info.get("range", "unknown"),
                            "fix_available": vuln_info.get("fixAvailable")
                        },
                        cve_id=vuln_info.get("cve")
                    ))
                    
        except Exception as e:
            findings.append(SecurityFinding(
                severity="low",
                category="scan_error",
                title="NPM Audit Parse Error",
                description=f"Failed to parse npm audit results: {str(e)}",
                impact="Cannot properly assess Node.js dependency security",
                recommendation="Manually review npm audit output",
                evidence={"error": str(e)}
            ))
            
        return findings
    
    def check_python_vulnerabilities(self, packages: List[Dict]) -> List[SecurityFinding]:
        """Check Python packages for known vulnerabilities"""
        findings = []
        
        # Known vulnerable Python packages (simplified check)
        vulnerable_packages = {
            "django": ("4.0", "CVE-2022-28346", "SQL injection vulnerability"),
            "requests": ("2.25.0", "CVE-2021-33503", "ReDoS vulnerability"),
            "pillow": ("8.2.0", "CVE-2021-34552", "Buffer overflow vulnerability"),
            "flask": ("2.0.0", "CVE-2021-23385", "Path traversal vulnerability")
        }
        
        for package in packages:
            name = package.get("name", "").lower()
            version = package.get("version", "")
            
            if name in vulnerable_packages:
                min_safe_version, cve, description = vulnerable_packages[name]
                
                # Simple version comparison (would need proper semver in production)
                if version < min_safe_version:
                    findings.append(SecurityFinding(
                        severity="high",
                        category="vulnerable_dependency",
                        title=f"Vulnerable Python Package: {name}",
                        description=description,
                        impact="Potential security vulnerability in application",
                        recommendation=f"Update {name} to version {min_safe_version} or later",
                        evidence={
                            "package": name,
                            "current_version": version,
                            "min_safe_version": min_safe_version
                        },
                        cve_id=cve
                    ))
                    
        return findings
    
    def run_static_code_analysis(self) -> List[SecurityFinding]:
        """Run static code analysis for security issues"""
        print("Running static code analysis...")
        findings = []
        
        try:
            # Check for common security anti-patterns in code
            code_files = []
            
            # Find source code files
            for ext in ["*.go", "*.ts", "*.tsx", "*.js", "*.jsx", "*.py"]:
                code_files.extend(Path(".").rglob(ext))
            
            security_patterns = [
                (r"password\s*=\s*[\"'][^\"']+[\"']", "Hardcoded password", "critical"),
                (r"api[_-]?key\s*=\s*[\"'][^\"']+[\"']", "Hardcoded API key", "high"),
                (r"secret\s*=\s*[\"'][^\"']+[\"']", "Hardcoded secret", "high"),
                (r"exec\s*\(", "Code execution function", "medium"),
                (r"eval\s*\(", "Code evaluation function", "high"),
                (r"innerHTML\s*=", "Potential XSS vulnerability", "medium"),
                (r"dangerouslySetInnerHTML", "Dangerous HTML injection", "high"),
                (r"document\.write\s*\(", "Potential XSS vulnerability", "medium"),
                (r"crypto\.md5", "Weak hashing algorithm", "medium"),
                (r"crypto\.sha1", "Weak hashing algorithm", "medium")
            ]
            
            for code_file in code_files:
                if code_file.stat().st_size > 1024 * 1024:  # Skip files larger than 1MB
                    continue
                    
                try:
                    with open(code_file, 'r', encoding='utf-8', errors='ignore') as f:
                        content = f.read()
                        
                    for pattern, description, severity in security_patterns:
                        matches = re.finditer(pattern, content, re.IGNORECASE)
                        for match in matches:
                            line_num = content[:match.start()].count('\n') + 1
                            
                            findings.append(SecurityFinding(
                                severity=severity,
                                category="code_security",
                                title=f"Security Issue: {description}",
                                description=f"Found {description.lower()} in {code_file}",
                                impact="Potential security vulnerability in source code",
                                recommendation="Review and remediate the identified security issue",
                                evidence={
                                    "file": str(code_file),
                                    "line": line_num,
                                    "pattern": pattern,
                                    "match": match.group()[:100]  # First 100 chars
                                }
                            ))
                            
                except Exception as e:
                    # Skip files that can't be read
                    continue
                    
        except Exception as e:
            findings.append(SecurityFinding(
                severity="low",
                category="scan_error",
                title="Static Code Analysis Error",
                description=f"Failed to perform static code analysis: {str(e)}",
                impact="Cannot verify code security",
                recommendation="Manually review code for security issues",
                evidence={"error": str(e)}
            ))
            
        return findings
    
    def run_configuration_audit(self) -> List[SecurityFinding]:
        """Audit configuration files for security issues"""
        print("Running configuration audit...")
        findings = []
        
        try:
            # Check Docker configurations
            dockerfile_paths = list(Path(".").rglob("Dockerfile*"))
            for dockerfile in dockerfile_paths:
                findings.extend(self.audit_dockerfile(dockerfile))
            
            # Check Kubernetes configurations
            k8s_files = list(Path(".").rglob("*.yaml")) + list(Path(".").rglob("*.yml"))
            for k8s_file in k8s_files:
                if "k8s" in str(k8s_file) or "kubernetes" in str(k8s_file):
                    findings.extend(self.audit_k8s_config(k8s_file))
            
            # Check environment configurations
            env_files = list(Path(".").rglob(".env*")) + list(Path(".").rglob("*.env"))
            for env_file in env_files:
                findings.extend(self.audit_env_file(env_file))
                
        except Exception as e:
            findings.append(SecurityFinding(
                severity="medium",
                category="scan_error",
                title="Configuration Audit Error",
                description=f"Failed to audit configurations: {str(e)}",
                impact="Cannot verify configuration security",
                recommendation="Manually review configuration files",
                evidence={"error": str(e)}
            ))
            
        return findings
    
    def audit_dockerfile(self, dockerfile_path: Path) -> List[SecurityFinding]:
        """Audit Dockerfile for security issues"""
        findings = []
        
        try:
            with open(dockerfile_path, 'r') as f:
                content = f.read()
            
            lines = content.split('\n')
            
            for i, line in enumerate(lines, 1):
                line = line.strip()
                
                # Check for running as root
                if line.startswith('USER root') or (line.startswith('USER') and '0' in line):
                    findings.append(SecurityFinding(
                        severity="medium",
                        category="configuration",
                        title="Container Running as Root",
                        description=f"Container configured to run as root user in {dockerfile_path}",
                        impact="Potential privilege escalation if container is compromised",
                        recommendation="Create and use a non-root user in the container",
                        evidence={"file": str(dockerfile_path), "line": i, "content": line}
                    ))
                
                # Check for latest tag usage
                if ':latest' in line and line.startswith('FROM'):
                    findings.append(SecurityFinding(
                        severity="low",
                        category="configuration",
                        title="Using Latest Tag",
                        description=f"Using 'latest' tag in base image in {dockerfile_path}",
                        impact="Unpredictable builds and potential security issues",
                        recommendation="Use specific version tags for base images",
                        evidence={"file": str(dockerfile_path), "line": i, "content": line}
                    ))
                
                # Check for secrets in ADD/COPY commands
                if (line.startswith('COPY') or line.startswith('ADD')) and any(secret in line.lower() for secret in ['password', 'key', 'token', 'secret']):
                    findings.append(SecurityFinding(
                        severity="high",
                        category="configuration",
                        title="Potential Secret in Dockerfile",
                        description=f"Potential secret copied into image in {dockerfile_path}",
                        impact="Secrets exposed in container image",
                        recommendation="Use Docker secrets or environment variables instead",
                        evidence={"file": str(dockerfile_path), "line": i, "content": line}
                    ))
                    
        except Exception as e:
            findings.append(SecurityFinding(
                severity="low",
                category="scan_error",
                title="Dockerfile Audit Error",
                description=f"Failed to audit {dockerfile_path}: {str(e)}",
                impact="Cannot verify Dockerfile security",
                recommendation="Manually review Dockerfile",
                evidence={"error": str(e), "file": str(dockerfile_path)}
            ))
            
        return findings
    
    def audit_k8s_config(self, k8s_file: Path) -> List[SecurityFinding]:
        """Audit Kubernetes configuration for security issues"""
        findings = []
        
        try:
            with open(k8s_file, 'r') as f:
                content = yaml.safe_load_all(f)
            
            for doc in content:
                if not doc:
                    continue
                    
                kind = doc.get('kind', '')
                
                if kind == 'Pod' or kind == 'Deployment':
                    # Check security context
                    spec = doc.get('spec', {})
                    
                    if kind == 'Deployment':
                        template = spec.get('template', {})
                        spec = template.get('spec', {})
                    
                    security_context = spec.get('securityContext', {})
                    
                    # Check for privileged containers
                    containers = spec.get('containers', [])
                    for container in containers:
                        container_security = container.get('securityContext', {})
                        
                        if container_security.get('privileged', False):
                            findings.append(SecurityFinding(
                                severity="high",
                                category="configuration",
                                title="Privileged Container",
                                description=f"Container configured as privileged in {k8s_file}",
                                impact="Container has elevated privileges on the host",
                                recommendation="Remove privileged flag unless absolutely necessary",
                                evidence={"file": str(k8s_file), "container": container.get('name')}
                            ))
                        
                        if container_security.get('runAsUser') == 0:
                            findings.append(SecurityFinding(
                                severity="medium", 
                                category="configuration",
                                title="Container Running as Root",
                                description=f"Container configured to run as root in {k8s_file}",
                                impact="Potential privilege escalation if container is compromised",
                                recommendation="Configure container to run as non-root user",
                                evidence={"file": str(k8s_file), "container": container.get('name')}
                            ))
                            
        except Exception as e:
            findings.append(SecurityFinding(
                severity="low",
                category="scan_error",
                title="Kubernetes Config Audit Error",
                description=f"Failed to audit {k8s_file}: {str(e)}",
                impact="Cannot verify Kubernetes configuration security",
                recommendation="Manually review Kubernetes configurations",
                evidence={"error": str(e), "file": str(k8s_file)}
            ))
            
        return findings
    
    def audit_env_file(self, env_file: Path) -> List[SecurityFinding]:
        """Audit environment files for exposed secrets"""
        findings = []
        
        try:
            with open(env_file, 'r') as f:
                lines = f.readlines()
            
            for i, line in enumerate(lines, 1):
                line = line.strip()
                
                if '=' in line and not line.startswith('#'):
                    key, value = line.split('=', 1)
                    key = key.strip()
                    value = value.strip()
                    
                    # Check for potential secrets
                    if any(secret in key.lower() for secret in ['password', 'secret', 'key', 'token']) and value and value != '${...}':
                        findings.append(SecurityFinding(
                            severity="medium",
                            category="configuration",
                            title="Potential Secret in Environment File",
                            description=f"Potential secret found in {env_file}",
                            impact="Secrets may be exposed in version control or logs",
                            recommendation="Use proper secret management or environment variable injection",
                            evidence={"file": str(env_file), "line": i, "key": key}
                        ))
                        
        except Exception as e:
            findings.append(SecurityFinding(
                severity="low",
                category="scan_error",
                title="Environment File Audit Error",
                description=f"Failed to audit {env_file}: {str(e)}",
                impact="Cannot verify environment file security",
                recommendation="Manually review environment files",
                evidence={"error": str(e), "file": str(env_file)}
            ))
            
        return findings
    
    def run_compliance_check(self) -> List[SecurityFinding]:
        """Check compliance with security standards"""
        print("Running compliance checks...")
        findings = []
        
        # Check for required security documentation
        required_docs = [
            "SECURITY.md",
            "docs/security.md", 
            "security/README.md",
            "PRIVACY.md",
            "docs/privacy.md"
        ]
        
        for doc in required_docs:
            if not Path(doc).exists():
                findings.append(SecurityFinding(
                    severity="low",
                    category="compliance",
                    title="Missing Security Documentation",
                    description=f"Missing required security document: {doc}",
                    impact="Compliance and security posture documentation gaps",
                    recommendation=f"Create {doc} with appropriate security information",
                    evidence={"missing_document": doc}
                ))
        
        # Check for security testing
        test_files = list(Path(".").rglob("*security*test*")) + list(Path(".").rglob("*test*security*"))
        
        if not test_files:
            findings.append(SecurityFinding(
                severity="medium",
                category="compliance",
                title="Missing Security Tests",
                description="No security-specific tests found in the project",
                impact="Security vulnerabilities may not be caught during development",
                recommendation="Implement comprehensive security testing",
                evidence={"searched_patterns": ["*security*test*", "*test*security*"]}
            ))
        
        # Check for CI/CD security
        ci_files = list(Path(".").rglob(".github/workflows/*")) + list(Path(".").rglob(".gitlab-ci.yml")) + list(Path(".").rglob("Jenkinsfile"))
        
        has_security_scan = False
        for ci_file in ci_files:
            try:
                with open(ci_file, 'r') as f:
                    content = f.read()
                    
                if any(keyword in content.lower() for keyword in ['security', 'vulnerability', 'audit', 'scan']):
                    has_security_scan = True
                    break
            except:
                continue
        
        if not has_security_scan:
            findings.append(SecurityFinding(
                severity="medium",
                category="compliance",
                title="Missing CI/CD Security Scanning",
                description="No security scanning found in CI/CD pipelines",
                impact="Security vulnerabilities may not be caught before deployment",
                recommendation="Add security scanning to CI/CD pipelines",
                evidence={"ci_files_checked": [str(f) for f in ci_files]}
            ))
        
        return findings
    
    def generate_audit_report(self) -> Dict[str, Any]:
        """Generate comprehensive security audit report"""
        
        # Categorize findings by severity
        critical_findings = [f for f in self.findings if f.severity == "critical"]
        high_findings = [f for f in self.findings if f.severity == "high"]
        medium_findings = [f for f in self.findings if f.severity == "medium"]
        low_findings = [f for f in self.findings if f.severity == "low"]
        info_findings = [f for f in self.findings if f.severity == "info"]
        
        # Calculate risk score
        risk_score = (
            len(critical_findings) * 10 +
            len(high_findings) * 7 +
            len(medium_findings) * 4 +
            len(low_findings) * 2 +
            len(info_findings) * 1
        )
        
        total_findings = len(self.findings)
        
        # Determine overall risk level
        if critical_findings or len(high_findings) > 5:
            risk_level = "Critical"
        elif len(high_findings) > 2 or len(medium_findings) > 10:
            risk_level = "High"
        elif len(medium_findings) > 5 or len(low_findings) > 15:
            risk_level = "Medium"
        else:
            risk_level = "Low"
        
        report = {
            "audit_timestamp": time.strftime("%Y-%m-%d %H:%M:%S"),
            "summary": {
                "total_findings": total_findings,
                "critical_findings": len(critical_findings),
                "high_findings": len(high_findings),
                "medium_findings": len(medium_findings),
                "low_findings": len(low_findings),
                "info_findings": len(info_findings),
                "risk_score": risk_score,
                "risk_level": risk_level
            },
            "findings_by_category": {
                "vulnerable_dependency": len([f for f in self.findings if f.category == "vulnerable_dependency"]),
                "code_security": len([f for f in self.findings if f.category == "code_security"]),
                "configuration": len([f for f in self.findings if f.category == "configuration"]),
                "compliance": len([f for f in self.findings if f.category == "compliance"]),
                "scan_error": len([f for f in self.findings if f.category == "scan_error"])
            },
            "findings": [
                {
                    "severity": f.severity,
                    "category": f.category,
                    "title": f.title,
                    "description": f.description,
                    "impact": f.impact,
                    "recommendation": f.recommendation,
                    "evidence": f.evidence,
                    "cve_id": f.cve_id,
                    "cvss_score": f.cvss_score
                } for f in self.findings
            ]
        }
        
        return report
    
    def run_comprehensive_security_audit(self) -> Dict[str, Any]:
        """Execute complete security audit"""
        print("🔒 Starting Comprehensive Security Audit")
        print("=" * 60)
        
        audit_start_time = time.time()
        
        try:
            # Execute audit components
            if self.config.include_dependency_scan:
                dependency_findings = self.run_dependency_vulnerability_scan()
                self.findings.extend(dependency_findings)
                print(f"Dependency scan completed: {len(dependency_findings)} findings")
            
            if self.config.include_code_analysis:
                code_findings = self.run_static_code_analysis()
                self.findings.extend(code_findings)
                print(f"Code analysis completed: {len(code_findings)} findings")
            
            if self.config.include_compliance_check:
                config_findings = self.run_configuration_audit()
                self.findings.extend(config_findings)
                print(f"Configuration audit completed: {len(config_findings)} findings")
            
            if self.config.include_compliance_check:
                compliance_findings = self.run_compliance_check()
                self.findings.extend(compliance_findings)
                print(f"Compliance check completed: {len(compliance_findings)} findings")
            
            # Generate comprehensive report
            report = self.generate_audit_report()
            
            # Save report to file
            report_file = Path(self.config.output_dir) / f"security_audit_report_{int(time.time())}.json"
            with open(report_file, 'w') as f:
                json.dump(report, f, indent=2)
            
            audit_duration = time.time() - audit_start_time
            
            print("\n" + "=" * 60)
            print("🔒 SECURITY AUDIT SUMMARY")
            print("=" * 60)
            print(f"Audit Duration: {audit_duration:.2f} seconds")
            print(f"Total Findings: {report['summary']['total_findings']}")
            print(f"Critical: {report['summary']['critical_findings']}")
            print(f"High: {report['summary']['high_findings']}")
            print(f"Medium: {report['summary']['medium_findings']}")
            print(f"Low: {report['summary']['low_findings']}")
            print(f"Risk Score: {report['summary']['risk_score']}")
            print(f"Risk Level: {report['summary']['risk_level']}")
            print(f"Report saved to: {report_file}")
            
            # Print summary of critical and high findings
            critical_and_high = [f for f in self.findings if f.severity in ["critical", "high"]]
            if critical_and_high:
                print("\n--- Critical and High Severity Findings ---")
                for finding in critical_and_high[:10]:  # Show first 10
                    print(f"[{finding.severity.upper()}] {finding.title}")
                    print(f"  {finding.description}")
                    print(f"  Recommendation: {finding.recommendation}")
                    print()
                    
                if len(critical_and_high) > 10:
                    print(f"... and {len(critical_and_high) - 10} more critical/high findings")
            
            if report['summary']['risk_level'] == "Critical":
                print("\n❌ CRITICAL SECURITY ISSUES FOUND - Immediate action required!")
            elif report['summary']['risk_level'] == "High":
                print("\n⚠️  HIGH SECURITY RISK - Remediation needed soon")
            elif report['summary']['risk_level'] == "Medium":
                print("\n⚡ MEDIUM SECURITY RISK - Plan remediation")
            else:
                print("\n✅ LOW SECURITY RISK - Good security posture")
            
            return report
            
        except Exception as e:
            error_report = {
                "error": str(e),
                "audit_duration": time.time() - audit_start_time,
                "partial_findings": len(self.findings)
            }
            
            print(f"\n❌ Security audit failed: {e}")
            return error_report


def main():
    """Main execution function"""
    config = SecurityAuditConfig()
    
    # Override config from environment if available
    config.api_base_url = os.getenv("AEGIS_API_URL", config.api_base_url)
    config.frontend_base_url = os.getenv("AEGIS_FRONTEND_URL", config.frontend_base_url)
    config.output_dir = os.getenv("AUDIT_OUTPUT_DIR", config.output_dir)
    
    auditor = SecurityAuditor(config)
    report = auditor.run_comprehensive_security_audit()
    
    # Exit with appropriate code based on risk level
    if "error" in report:
        sys.exit(2)
    elif report.get("summary", {}).get("risk_level") in ["Critical"]:
        sys.exit(1)
    else:
        sys.exit(0)


if __name__ == "__main__":
    main()