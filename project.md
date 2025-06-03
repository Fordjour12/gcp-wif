Step-by-Step Guide for Workload Identity Federation:
1. Prepare your Python Backend (Same as before)
Ensure your main.py, requirements.txt, and Dockerfile are in place within your GitHub repository.

2. Create a Google Cloud Service Account (if you don't have one)
This service account will be impersonated by your GitHub Actions workflow.

Go to IAM & Admin > Service Accounts in the Google Cloud Console.
Click + CREATE SERVICE ACCOUNT.
Give it a descriptive name (e.g., github-actions-wif-deployer).
Grant the necessary roles to this service account:
Artifact Registry Writer: To push Docker images to Artifact Registry.
Cloud Run Admin: To deploy and manage Cloud Run services.
Service Account User: (If your Cloud Run service needs to impersonate another service account at runtime, or if the deployer itself needs to impersonate others for complex operations, though typically not needed for basic Cloud Run deployment).
Click DONE.
Important: You do not create or download a key for this service account when using WIF.
3. Configure Workload Identity Pool and Provider
This is the core of WIF setup.

Go to IAM & Admin > Workload Identity Federation in the Google Cloud Console.
Click CREATE POOL.
Name: github-actions-pool (or a descriptive name)
Display name: GitHub Actions Pool
Description: Pool for GitHub Actions workflows to authenticate.
Click CONTINUE.
Click ADD A PROVIDER.
Select a provider: Choose OpenID Connect (OIDC).
Provider name: github-provider (or a descriptive name).
Issuer (URL): https://token.actions.githubusercontent.com (This is GitHub's OIDC issuer URL).
Allowed audiences: Enter https://github.com/YOUR_GITHUB_ORG_OR_USERNAME. This is a security measure to ensure only tokens from your GitHub organization or user account are trusted.
Click CONTINUE.
Configure attributes (Crucial for security): This step maps claims from the GitHub OIDC token to Google Cloud attributes. This is where you enforce granular conditions.
Default attribute mapping:
google.subject: assertion.sub (This maps the unique subject from the GitHub token, which is repo:YOUR_GITHUB_ORG_OR_USERNAME/YOUR_REPO_NAME:ref:main or similar, to the Google Cloud subject).
Custom attribute mappings: This is where you add more specific conditions. A common and highly recommended approach is to map repository and ref to attributes and then use them in conditions.
Attribute: attribute.actor
Value: assertion.actor (GitHub username of the action invoker)
Attribute: attribute.repository
Value: assertion.repository (e.g., YOUR_GITHUB_ORG_OR_USERNAME/YOUR_REPO_NAME)
Attribute: attribute.ref
Value: assertion.ref (e.g., refs/heads/main)
Attribute: attribute.pull_request
Value: assertion.pull_request (if you want to restrict based on PRs)
Click CONTINUE.
Review and Grant Access:
In the "Grant access to the service account" section, click GRANT ACCESS.
Service account: Select the service account you created in Step 2 (e.g., github-actions-wif-deployer@your-project-id.iam.gserviceaccount.com).
Condition: This is highly recommended for security. You can define a condition to restrict which GitHub repositories or branches can impersonate this service account.
Example condition for a specific repository and main branch:
attribute.repository == 'YOUR_GITHUB_ORG_OR_USERNAME/YOUR_REPO_NAME' && attribute.ref == 'refs/heads/main'
Replace YOUR_GITHUB_ORG_OR_USERNAME and YOUR_REPO_NAME with your actual GitHub details. You can also make it more general: attribute.repository == 'YOUR_GITHUB_ORG_OR_USERNAME/YOUR_REPO_NAME'
Click ADD CONDITION.
Click SAVE.
Copy the Provider Resource Name: After creating the provider, you'll see a summary page. Copy the Provider Resource Name. It will look something like: projects/PROJECT_NUMBER/locations/global/workloadIdentityPools/github-actions-pool/providers/github-provider You'll need this for your GitHub Actions workflow.
4. Update GitHub Secrets
You no longer need to store the service account JSON key in GitHub Secrets. Instead, you'll store your Google Cloud Project ID and the Workload Identity Provider resource name.

In your GitHub repository, go to Settings > Secrets and variables > Actions.
Delete the GCP_SA_KEY secret if you created it previously.
Add or confirm the following secrets:
GCP_PROJECT_ID: Your Google Cloud Project ID.
GCP_WIF_PROVIDER: Paste the Provider Resource Name you copied in the previous step.
5. Configure GitHub Actions Workflow
Now, modify your .github/workflows/deploy.yml file to use Workload Identity Federation.

YAML

name: Deploy Python Backend to Cloud Run (Workload Identity Federation)

on:
  push:
    branches:
      - main # Trigger on pushes to the main branch

env:
  PROJECT_ID: ${{ secrets.GCP_PROJECT_ID }}
  GCP_REGION: us-central1 # Your desired GCP region
  SERVICE_NAME: my-python-backend # Your Cloud Run service name
  ARTIFACT_REGISTRY_REPO: my-python-backends # Name of your Artifact Registry Docker repo
  # You'll use this service account for impersonation (the one configured in WIF)
  GCP_SERVICE_ACCOUNT: github-actions-wif-deployer@${{ secrets.GCP_PROJECT_ID }}.iam.gserviceaccount.com

jobs:
  build-and-deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: 'read' # Required to checkout code
      id-token: 'write' # Required to fetch the OIDC ID Token from GitHub

    steps:
      - name: Checkout code
        uses: actions/checkout@v4

      - name: Authenticate to Google Cloud
        id: 'auth'
        uses: 'google-github-actions/auth@v2'
        with:
          workload_identity_provider: ${{ secrets.GCP_WIF_PROVIDER }}
          service_account: ${{ env.GCP_SERVICE_ACCOUNT }}
          # Uncomment if you want to use a specific project ID for authentication, though PROJECT_ID env var is usually sufficient
          # project_id: ${{ env.PROJECT_ID }}

      - name: Set up Docker
        uses: docker/setup-buildx-action@v3

      - name: Configure Docker to use Artifact Registry
        run: |
          gcloud auth configure-docker ${GCP_REGION}-docker.pkg.dev

      - name: Build and push Docker image to Artifact Registry
        run: |
          docker build -t ${GCP_REGION}-docker.pkg.dev/${PROJECT_ID}/${ARTIFACT_REGISTRY_REPO}/${SERVICE_NAME}:${{ github.sha }} .
          docker push ${GCP_REGION}-docker.pkg.dev/${PROJECT_ID}/${ARTIFACT_REGISTRY_REPO}/${SERVICE_NAME}:${{ github.sha }}

      - name: Deploy to Cloud Run
        uses: google-github-actions/deploy-cloudrun@v2
        with:
          service: ${{ env.SERVICE_NAME }}
          region: ${{ env.GCP_REGION }}
          image: ${{ env.GCP_REGION }}-docker.pkg.dev/${{ env.PROJECT_ID }}/${{ env.ARTIFACT_REGISTRY_REPO }}/${{ env.SERVICE_NAME }}:${{ github.sha }}
          # Uncomment and modify if you need to set environment variables
          # env_vars: |
          #   NAME=GitHubActionsWIF
          allow_unauthenticated: true # Uncomment if your service should be publicly accessible
          # revision_suffix: ${{ github.sha }} # Optional: use commit SHA for unique revision names
          # traffic: LATEST=100 # Direct 100% traffic to the latest revision
Key Changes in the Workflow:

permissions block: This is essential for WIF.
contents: 'read': Allows the workflow to read your repository's code.
id-token: 'write': Grants the workflow permission to request the OIDC ID token from GitHub. This token is what Google Cloud's Workload Identity Federation verifies.
Authenticateto Google Cloud step:
The google-github-actions/auth@v2 action is used.
workload_identity_provider: This now uses the GCP_WIF_PROVIDER secret, which holds the full resource name of your Workload Identity Pool Provider.
service_account: Specifies the email of the Google Cloud Service Account that GitHub Actions will impersonate.
6. Commit and Push
Commit the deploy.yml file to your main branch and push it to GitHub:

Bash

git add .github/workflows/deploy.yml
git commit -m "Configure GitHub Actions with Workload Identity Federation for Cloud Run"
git push origin main
7. Monitor and Verify
Go to the Actions tab in your GitHub repository to monitor the workflow run.
Once completed, verify the deployment in the Google Cloud Console under Cloud Run.



'''#!/bin/bash

# --- User Configuration ---
# Your Google Cloud Project ID
YOUR_PROJECT_ID="your-gcp-project-id"

# Your GitHub Organization or Username (e.g., 'my-github-org' or 'my-github-username')
YOUR_GITHUB_ORG_OR_USERNAME="your-github-org-or-username"

# Your specific GitHub Repository Name (e.g., 'my-repo')
# If you want to allow all repositories under your organization, you can make the condition more general later.
YOUR_REPO_NAME="your-github-repo-name"

# The name of the service account you created in a previous step (e.g., github-actions-wif-deployer)
SERVICE_ACCOUNT_NAME="github-actions-wif-deployer"
# --- End User Configuration ---

# --- Derived Variables (do not modify) ---
POOL_ID="github-actions-pool"
PROVIDER_ID="github-provider"
SERVICE_ACCOUNT_EMAIL="${SERVICE_ACCOUNT_NAME}@${YOUR_PROJECT_ID}.iam.gserviceaccount.com"

echo "Starting Workload Identity Federation configuration for project: ${YOUR_PROJECT_ID}"
echo "GitHub Organization/User: ${YOUR_GITHUB_ORG_OR_USERNAME}"
echo "GitHub Repository: ${YOUR_REPO_NAME}"
echo "Service Account: ${SERVICE_ACCOUNT_EMAIL}"
echo ""

# Get the Project Number, which is needed for the IAM policy binding
PROJECT_NUMBER=$(gcloud projects describe "${YOUR_PROJECT_ID}" --format="value(projectNumber)")
if [ -z "${PROJECT_NUMBER}" ]; then
  echo "ERROR: Could not retrieve project number for ${YOUR_PROJECT_ID}. Please check your project ID and permissions."
  exit 1
fi
echo "Google Cloud Project Number: ${PROJECT_NUMBER}"
echo ""

# 1. Create Workload Identity Pool
echo "1. Creating Workload Identity Pool: ${POOL_ID}..."
gcloud iam workload-identity-pools create "${POOL_ID}" \
  --project="${YOUR_PROJECT_ID}" \
  --display-name="GitHub Actions Pool" \
  --description="Pool for GitHub Actions workflows to authenticate." || {
    echo "WARNING: Workload Identity Pool '${POOL_ID}' might already exist or creation failed."
  }
echo "Workload Identity Pool created/checked."
echo ""

# 2. Add a Provider (OpenID Connect)
echo "2. Adding Workload Identity Provider: ${PROVIDER_ID}..."
# The attribute mapping is crucial for granular security conditions later.
gcloud iam workload-identity-pools providers create-oidc "${PROVIDER_ID}" \
  --project="${YOUR_PROJECT_ID}" \
  --location="global" \
  --workload-identity-pool="${POOL_ID}" \
  --display-name="GitHub Provider" \
  --description="OIDC provider for GitHub Actions." \
  --issuer-uri="https://token.actions.githubusercontent.com" \
  --allowed-audiences="https://github.com/${YOUR_GITHUB_ORG_OR_USERNAME}" \
  --attribute-mapping="google.subject=assertion.sub,attribute.actor=assertion.actor,attribute.repository=assertion.repository,attribute.ref=assertion.ref,attribute.pull_request=assertion.pull_request" || {
    echo "WARNING: Workload Identity Provider '${PROVIDER_ID}' might already exist or creation failed."
  }
echo "Workload Identity Provider created/checked."
echo ""

# Construct the full provider resource path for IAM binding
PROVIDER_FULL_PATH="projects/${PROJECT_NUMBER}/locations/global/workloadIdentityPools/${POOL_ID}/providers/${PROVIDER_ID}"

# 3. Grant Access to Service Account with Condition
echo "3. Granting access to service account '${SERVICE_ACCOUNT_EMAIL}' with a condition..."

# Define the condition for the IAM policy binding
# This condition restricts impersonation to a specific GitHub repository and its 'main' branch.
# You can modify this condition based on your security needs.
# For example, to allow any branch in the repo: attribute.repository == '${YOUR_GITHUB_ORG_OR_USERNAME}/${YOUR_REPO_NAME}'
# Or to allow any repo in the org: attribute.repository.startsWith('${YOUR_GITHUB_ORG_OR_USERNAME}/')
CONDITION_EXPRESSION="attribute.repository == '${YOUR_GITHUB_ORG_OR_USERNAME}/${YOUR_REPO_NAME}' && attribute.ref == 'refs/heads/main'"
CONDITION_TITLE="GitHub Actions Main Branch Access for ${YOUR_REPO_NAME}"
CONDITION_DESCRIPTION="Allows GitHub Actions from ${YOUR_GITHUB_ORG_OR_USERNAME}/${YOUR_REPO_NAME} main branch to impersonate this service account."

echo "Applying IAM policy binding with condition: '${CONDITION_EXPRESSION}'"

gcloud iam service-accounts add-iam-policy-binding "${SERVICE_ACCOUNT_EMAIL}" \
  --project="${YOUR_PROJECT_ID}" \
  --role="roles/iam.workloadIdentityUser" \
  --member="principalSet://iam.googleapis.com/${PROVIDER_FULL_PATH}" \
  --condition="expression=${CONDITION_EXPRESSION},title=${CONDITION_TITLE},description=${CONDITION_DESCRIPTION}" || {
    echo "ERROR: Failed to add IAM policy binding. Check service account name, project ID, and permissions."
    exit 1
  }

echo "IAM policy binding added successfully."
echo ""

# 4. Copy the Provider Resource Name
echo "Configuration complete!"
echo "Please copy the following Workload Identity Provider Resource Name."
echo "You will need this in your GitHub Actions workflow configuration (e.g., in the 'id-token' section)."
echo "------------------------------------------------------------------------------------------------------"
echo "projects/${YOUR_PROJECT_ID}/locations/global/workloadIdentityPools/${POOL_ID}/providers/${PROVIDER_ID}"
echo "------------------------------------------------------------------------------------------------------"
echo ""
'''
