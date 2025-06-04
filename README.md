# GCP Workload Identity Federation CLI Tool

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)]()

🚀 **Enterprise-grade CLI tool for automating Google Cloud Workload Identity Federation (WIF) setup with GitHub Actions**

Eliminate the manual complexity of configuring WIF through the Google Cloud Console. This tool provides complete automation, multi-environment support, comprehensive testing, and enterprise-ready security configurations.

## ✨ Features

### **🎯 Complete WIF Automation**
- **Interactive Setup** - Guided configuration collection with validation
- **Service Account Management** - Automated creation and IAM role assignment
- **Workload Identity Federation** - Pool and provider configuration with security conditions
- **GitHub Actions Integration** - Workflow generation with multiple templates
- **Resource Conflict Detection** - Intelligent handling of existing resources

### **🌍 Multi-Environment Support**
- **Environment Management** - Production, staging, development, and testing environments
- **Multi-Region Deployment** - Geographic distribution across GCP regions
- **Environment-Specific Settings** - Custom security, roles, and configurations per environment
- **Smart Resource Naming** - Environment-aware resource naming with prefixes and suffixes

### **🔒 Enterprise Security**
- **Security Best Practices** - Environment-appropriate security configurations
- **Conditional Access** - Repository and branch-based access restrictions
- **Token Lifecycle Management** - Environment-specific token TTL settings
- **Approval Workflows** - Production deployment approval requirements

### **🧪 Comprehensive Testing**
- **Validation Framework** - Complete configuration and resource validation
- **Multi-Category Testing** - Configuration, GCP, GitHub, workflow, integration, security, performance, and resilience testing
- **Multiple Output Formats** - Summary, detailed, JSON, and JUnit formats
- **CI/CD Integration** - Automated testing for continuous integration

### **🔄 Lifecycle Management**
- **Cleanup and Rollback** - Safe resource removal and state restoration
- **Configuration Migration** - Automated version upgrades and configuration updates
- **Backup and Recovery** - Automatic configuration backups
- **State Tracking** - Complete operation history and audit trails

## 📦 Installation

### **Prerequisites**
- Go 1.21 or later
- Google Cloud CLI (`gcloud`) installed and configured
  - You will also need to authenticate for Application Default Credentials (ADC) if you haven't already. Run the following command and follow the browser prompts:
    ```bash
    gcloud auth application-default login
    ```
- Access to a Google Cloud Project with necessary permissions
- GitHub repository with Actions enabled

### **Install from Source**
```bash
# Clone the repository
git clone https://github.com/your-org/gcp-wif.git
cd gcp-wif

# Build the tool
go build -o gcp-wif .

# Install to PATH (optional)
sudo mv gcp-wif /usr/local/bin/
```

### **Quick Start**
```bash
# Interactive setup (recommended for first-time users)
gcp-wif setup

# Or configure directly
gcp-wif setup --project my-project --repo myorg/myrepo
```

## 🚀 Quick Start Guide

### **1. Initial Setup**
```bash
# Start interactive configuration
gcp-wif setup

# Follow the prompts to configure:
# - GCP Project ID
# - GitHub Repository (owner/name)
# - Service Account details
# - Workload Identity settings
# - Cloud Run configuration (optional)
```

### **2. Multi-Environment Configuration**
```bash
# Create a production environment
gcp-wif env create --name prod --type production --region us-central1

# Create staging environment
gcp-wif env create --name staging --type staging --region us-east1

# Create development environment
gcp-wif env create --name dev --type development --region us-west1

# Switch to production environment
gcp-wif env use prod
```

### **3. Generate Workflows**
```bash
# Generate production workflow
gcp-wif workflow generate --template production

# Generate staging workflow
gcp-wif workflow generate --template staging --output-file .github/workflows/staging.yml

# Generate development workflow
gcp-wif workflow generate --template development
```

### **4. Comprehensive Testing**
```bash
# Run all tests
gcp-wif test

# Run specific test categories
gcp-wif test --gcp-only --security-only

# Run tests with detailed output
gcp-wif test --verbose --show-details

# Run tests in CI/CD (JSON output)
gcp-wif test --output-format json --output-file test-results.json
```

## 📋 Command Reference

### **Core Commands**

#### `gcp-wif setup`
Complete end-to-end WIF setup with intelligent orchestration.

```bash
# Interactive setup
gcp-wif setup

# Direct configuration
gcp-wif setup --project my-project --repo owner/repo

# Using configuration file
gcp-wif setup --config wif-config.json

# Force update existing resources
gcp-wif setup --force-update
```

#### `gcp-wif env`
Multi-environment management system.

```bash
# List all environments
gcp-wif env list

# Create new environment
gcp-wif env create --name prod --type production --region us-central1

# Switch environments
gcp-wif env use staging

# Show current environment
gcp-wif env current --details

# Manage regions
gcp-wif env regions
```

#### `gcp-wif workflow`
GitHub Actions workflow generation and management.

```bash
# Generate workflow with default template
gcp-wif workflow generate

# Use specific template
gcp-wif workflow generate --template production

# Custom output location
gcp-wif workflow generate --output-file custom-deploy.yml

# Preview workflow without writing
gcp-wif workflow preview
```

#### `gcp-wif test`
Comprehensive validation and testing framework.

```bash
# Run all tests
gcp-wif test

# Run specific test suites
gcp-wif test --suites configuration,gcp,security

# Run tests with filtering
gcp-wif test --min-severity high --skip-categories performance

# Dry run mode
gcp-wif test --dry-run

# Parallel execution
gcp-wif test --parallel --timeout 10m
```

#### `gcp-wif cleanup`
Resource cleanup and management.

```bash
# Interactive cleanup
gcp-wif cleanup

# Force cleanup without confirmation
gcp-wif cleanup --force

# Cleanup specific resource types
gcp-wif cleanup --service-account --workload-identity

# Dry run cleanup
gcp-wif cleanup --dry-run
```

#### `gcp-wif rollback`
State rollback and recovery.

```bash
# List available rollback points
gcp-wif rollback list

# Rollback to specific state
gcp-wif rollback restore --state-id abc123

# Rollback to previous state
gcp-wif rollback restore --previous
```

### **Environment Types**

| Type | Security Level | Use Case | Token TTL | Approval Required |
|------|---------------|----------|-----------|-------------------|
| **production** | High | Live production workloads | 1 hour | Yes |
| **staging** | Medium | Pre-production testing | 2 hours | Optional |
| **development** | Standard | Feature development | 4 hours | No |
| **testing** | Standard | Automated testing | 4 hours | No |

### **Workflow Templates**

| Template | Environment | Features | Security |
|----------|-------------|----------|----------|
| **production** | Production | Health checks, monitoring, rollback | Strict branch restrictions, signed commits |
| **staging** | Staging | Integration tests, approval gates | Moderate restrictions |
| **development** | Development | Fast builds, feature flags | Flexible access |

## 🔧 Configuration

### **Configuration File Structure**
```json
{
  "version": "1.0",
  "project": {
    "id": "my-gcp-project",
    "region": "us-central1"
  },
  "repository": {
    "owner": "myorg",
    "name": "myrepo"
  },
  "service_account": {
    "name": "github-myorg-myrepo",
    "roles": [
      "roles/run.admin",
      "roles/storage.admin",
      "roles/artifactregistry.admin"
    ]
  },
  "workload_identity": {
    "pool_id": "gh-myorg-myrepo-pool",
    "provider_id": "gh-myorg-myrepo-provider"
  },
  "environments": {
    "production": {
      "type": "production",
      "region": "us-central1",
      "security": {
        "require_approval": true,
        "require_signed_commits": true,
        "restrict_branches": ["main"]
      }
    },
    "staging": {
      "type": "staging",
      "region": "us-east1",
      "security": {
        "require_approval": false,
        "require_signed_commits": true,
        "restrict_branches": ["main", "staging"]
      }
    }
  },
  "current_environment": "production"
}
```

### **Environment Variables**
```bash
# Google Cloud Configuration
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/service-account.json"
export GOOGLE_CLOUD_PROJECT="my-project"

# CLI Configuration
export WIF_CONFIG_FILE="/path/to/wif-config.json"
export WIF_LOG_LEVEL="info"
export WIF_LOG_FILE="/var/log/gcp-wif.log"
```

## 🧪 Testing

### **Test Categories**

| Category | Description | Test Count | Coverage |
|----------|-------------|------------|----------|
| **Configuration** | Schema validation, required fields | 15+ | Configuration structure, defaults, validation rules |
| **GCP** | Resource access, permissions, connectivity | 20+ | Authentication, IAM, resource availability |
| **GitHub** | Repository validation, workflow integration | 10+ | Repository format, access, workflow generation |
| **Workflow** | GitHub Actions workflow validation | 15+ | YAML syntax, security, best practices |
| **Integration** | End-to-end orchestration testing | 12+ | Complete workflow validation |
| **Security** | Security configuration and compliance | 18+ | Access controls, permissions, conditions |
| **Performance** | Efficiency and speed validation | 8+ | Load times, resource usage |
| **Resilience** | Error handling and recovery | 10+ | Failure scenarios, recovery procedures |

### **Running Tests**
```bash
# Complete test suite
gcp-wif test

# Security and compliance focus
gcp-wif test --security-only --min-severity high

# Performance testing
gcp-wif test --performance-only --parallel

# CI/CD integration
gcp-wif test --output-format junit --output-file test-results.xml
```

## 🔒 Security

### **Security Features**
- **Conditional Access** - Repository and branch-based restrictions
- **Token Lifecycle Management** - Short-lived tokens with environment-specific TTLs
- **IAM Best Practices** - Principle of least privilege
- **Audit Logging** - Complete operation history
- **Secret Management** - Secure handling of sensitive data

### **Production Security Checklist**
- [ ] Enable signed commit requirements
- [ ] Configure branch restrictions (main/master only)
- [ ] Set up deployment approval requirements
- [ ] Use short token TTLs (1 hour for production)
- [ ] Enable monitoring and logging
- [ ] Regular security audits via `gcp-wif test --security-only`

### **Security Best Practices**
```bash
# Create production environment with enhanced security
gcp-wif env create \
  --name prod \
  --type production \
  --region us-central1

# Generate secure workflow
gcp-wif workflow generate \
  --template production \
  --output-file .github/workflows/prod-deploy.yml

# Validate security configuration
gcp-wif test --security-only --min-severity critical
```

## 🌍 Multi-Environment Deployment

### **Environment Strategy**
```bash
# 1. Set up environments
gcp-wif env create --name prod --type production --region us-central1
gcp-wif env create --name staging --type staging --region us-east1  
gcp-wif env create --name dev --type development --region us-west1

# 2. Configure environment-specific settings
gcp-wif env use prod
gcp-wif setup  # Configure production-specific settings

gcp-wif env use staging
gcp-wif setup  # Configure staging-specific settings

# 3. Generate environment-specific workflows
gcp-wif env use prod && gcp-wif workflow generate --template production
gcp-wif env use staging && gcp-wif workflow generate --template staging
gcp-wif env use dev && gcp-wif workflow generate --template development
```

### **Region Management**
```bash
# View available regions
gcp-wif env regions

# Configure multi-region deployment
gcp-wif env create --name prod-us --type production --region us-central1
gcp-wif env create --name prod-eu --type production --region europe-west1
gcp-wif env create --name prod-asia --type production --region asia-east1
```

## 📊 Monitoring and Observability

### **Built-in Monitoring**
- **Operation Logging** - Comprehensive logging of all operations
- **Performance Metrics** - Execution time tracking and optimization
- **Error Tracking** - Detailed error reporting and diagnosis
- **Audit Trails** - Complete history of configuration changes

### **Integration with GCP Monitoring**
```bash
# Enable monitoring for production environments
gcp-wif env create \
  --name prod \
  --type production \
  --region us-central1

# The production template automatically includes:
# - Cloud Logging integration
# - Cloud Monitoring metrics
# - Error reporting
# - Performance insights
```

## 🚨 Troubleshooting

### **Common Issues**

#### Authentication Problems
```bash
# Verify GCP authentication
gcp-wif test --gcp-only

# Check service account permissions
gcloud auth list
gcloud config list project
```

#### Configuration Issues
```bash
# Validate configuration
gcp-wif test --config-only

# Check configuration file
gcp-wif config validate --config wif-config.json
```

#### Workflow Generation Problems
```bash
# Test workflow generation
gcp-wif workflow preview

# Validate workflow syntax
gcp-wif test --workflow-only
```

### **Debug Mode**
```bash
# Enable verbose logging
gcp-wif --log-level debug --verbose setup

# Save logs to file
gcp-wif --log-file debug.log --log-level debug test
```

### **Getting Help**
```bash
# Command-specific help
gcp-wif setup --help
gcp-wif env --help
gcp-wif test --help

# Show version information
gcp-wif version
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### **Development Setup**
```bash
# Clone repository
git clone https://github.com/your-org/gcp-wif.git
cd gcp-wif

# Install dependencies
go mod tidy

# Run tests
go test ./...

# Build
go build -o gcp-wif .
```

### **Testing Your Changes**
```bash
# Run comprehensive tests
./gcp-wif test

# Test specific functionality
./gcp-wif test --gcp-only
./gcp-wif test --security-only
```

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🎯 Roadmap

### **Completed Features** ✅
- [x] Complete WIF automation
- [x] Multi-environment support
- [x] Comprehensive testing framework
- [x] Workflow generation with multiple templates
- [x] Cleanup and rollback functionality
- [x] Enterprise security features

### **Future Enhancements** 🚀
- [ ] Web dashboard for configuration management
- [ ] Terraform provider integration
- [ ] Advanced monitoring and alerting
- [ ] Multi-cloud support (AWS, Azure)
- [ ] API for programmatic access
- [ ] Plugin system for extensibility

## 📞 Support

- **Documentation**: [Full documentation](docs/)
- **Issues**: [GitHub Issues](https://github.com/your-org/gcp-wif/issues)
- **Discussions**: [GitHub Discussions](https://github.com/your-org/gcp-wif/discussions)
- **Security**: Report security issues to security@your-org.com

---

**🎉 Made with ❤️ for the DevOps community**

*Simplifying Google Cloud Workload Identity Federation, one deployment at a time.* 