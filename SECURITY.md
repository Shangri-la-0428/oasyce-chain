# Security Policy

## Supported Versions

| Version | Supported |
|---------|-----------|
| 0.1.x   | Yes       |

## Reporting a Vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Please report vulnerabilities privately to: **wutc@oasyce.com**

### What to Include

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

| Step | Timeline |
|------|----------|
| Acknowledgment | 48 hours |
| Initial assessment | 7 days |
| Fix target | 30 days |

We follow coordinated disclosure — we will work with you on timing before any public announcement.

## Scope

### In Scope

- Consensus-breaking bugs (non-determinism, state divergence)
- Fund loss or theft (escrow bypass, unauthorized transfers)
- Cryptographic issues (weak ID generation, hash collisions)
- Integer overflow/underflow in economic calculations
- Unauthorized state mutations (access control bypass)

### Out of Scope

- Denial of service via transaction spam (handled by gas/fee mechanism)
- Issues in dependencies (report upstream)
- Social engineering
- Testnet-only issues
