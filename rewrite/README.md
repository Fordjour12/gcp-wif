# WIF Setup - Simple Workload Identity Federation Tool

🚀 **One-command setup for GitHub Actions → Google Cloud Platform authentication**

Eliminate the complexity of manually configuring Workload Identity Federation. This focused CLI tool sets up secure, keyless authentication between GitHub Actions and GCP in seconds.

## ✨ Features

- **One Command**: Complete WIF setup with a single command
- **Smart Defaults**: Auto-generates pool IDs, provider IDs, and service account names
- **GitHub Compatible**: Uses correct audience patterns that match GitHub's expectations
- **Idempotent**: Safe to run multiple times - won't duplicate resources
- **Clear Output**: Shows exactly what was created and how to use it

## 🏗️ Prerequisites

- **gcloud CLI** installed and authenticated
- **Go 1.21+** (for building from source)
- A **Google Cloud Project** with IAM API enabled
- A **GitHub repository** you want to authenticate from

## 📦 Installation

### From Source
```bash
git clone https://github.com/yourusername/wif-setup.git
cd wif-setup
go build -o wif-setup .
```

### Quick Install
```bash
go install github.com/yourusername/wif-setup@latest
```

## 🚀 Quick Start

### Basic Usage
```bash
# Minimal command - auto-generates everything else
wif-setup setup --project my-gcp-project --repo owner/repository

# With custom service account
wif-setup setup \
  --project my-gcp-project \
  --repo owner/repository \
  --service-account my-sa@my-project.iam.gserviceaccount.com
```

### Example Output
```
🚀 Setting up Workload Identity Federation
   Project: my-gcp-project
   Repository: owner/repository
   Pool ID: owner-repository-pool
   Provider ID: github-provider
   Service Account: github-owner-repository@my-gcp-project.iam.gserviceaccount.com

🔧 Starting Workload Identity Federation setup...
   🏊 Checking workload identity pool: owner-repository-pool
   🆕 Creating workload identity pool: owner-repository-pool
   🔌 Checking workload identity provider: github-provider
   🆕 Creating workload identity provider: github-provider
   🔗 Binding service account: github-owner-repository@my-gcp-project.iam.gserviceaccount.com

🎉 Workload Identity Federation setup completed successfully!

🚀 Next Steps:
   Add this to your GitHub Actions workflow:

   - name: Authenticate to Google Cloud
     uses: google-github-actions/auth@v1
     with:
       workload_identity_provider: projects/my-gcp-project/locations/global/workloadIdentityPools/owner-repository-pool/providers/github-provider
       service_account: github-owner-repository@my-gcp-project.iam.gserviceaccount.com
```

## 🔧 Command Reference

### `wif-setup setup`

| Flag | Short | Description | Required |
|------|-------|-------------|----------|
| `--project` | `-p` | GCP Project ID | ✅ Yes |
| `--repo` | `-r` | GitHub repository (owner/repo format) | ✅ Yes |
| `--service-account` | `-s` | Service account email (auto-generated if not provided) | No |
| `--pool-id` | | Workload Identity Pool ID (auto-generated if not provided) | No |
| `--provider-id` | | Workload Identity Provider ID (default: github-provider) | No |

## 📋 What It Creates

1. **Workload Identity Pool**: Named `{owner}-{repo}-pool`
2. **OIDC Provider**: 
   - Issuer: `https://token.actions.githubusercontent.com`
   - Audience: `https://github.com/{owner}` + `sts.googleapis.com`
   - Maps essential GitHub claims (repository, actor, ref, etc.)
3. **IAM Binding**: Allows GitHub Actions from your repository to impersonate the service account

## 🔒 Security Features

- **Repository Restriction**: Only your specified repository can authenticate
- **Audience Validation**: Uses GitHub-specific audiences for enhanced security  
- **Attribute Conditions**: Validates repository ownership and actor
- **No Long-lived Keys**: Completely keyless authentication

## 📖 Example GitHub Workflow

After running the tool, add this to `.github/workflows/deploy.yml`:

```yaml
name: Deploy to GCP
on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      id-token: write
      contents: read
    
    steps:
    - uses: actions/checkout@v3
    
    - name: Authenticate to Google Cloud
      uses: google-github-actions/auth@v1
      with:
        workload_identity_provider: projects/my-project/locations/global/workloadIdentityPools/owner-repo-pool/providers/github-provider
        service_account: github-owner-repo@my-project.iam.gserviceaccount.com
    
    - name: Set up Cloud SDK
      uses: google-github-actions/setup-gcloud@v1
    
    - name: Deploy
      run: |
        gcloud run deploy my-service \
          --image gcr.io/my-project/my-app:latest \
          --region us-central1
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Test thoroughly
5. Submit a pull request

## 📝 License

MIT License - see [LICENSE](LICENSE) file for details.

---

**Simple. Focused. Just works.** 🎯 