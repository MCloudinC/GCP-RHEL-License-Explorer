# GCP Instance Explorer

GCP Instance Explorer is a command-line tool that helps you list and manage your RHEL License for compute instances in Google Cloud Platform projects.

## Features

- Authenticate with Google Cloud Platform
- List all instances in a specified project with detailed information:
  - Instance name, zone, machine type, and status
  - IP addresses ---- This will likely be taken out as it causes to much noise
  - Disk type and size ---- totallly useless but was the first step will remove
  - License information
- Manage instances:
  - Start (turn on) instances
  - Stop (turn off) instances (its only gracefull if that is turned on, I don't garrentee that it won't just turn it off)
  - Update license information via metadata
  - Refresh instance list to see status changes

## Prerequisites

- Go 1.16 or later
- A Google Cloud Platform account
- One or more GCP projects with appropriate permissions
- [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) installed (recommended)

## Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/yourusername/gcp-instance-explorer.git
   cd gcp-instance-explorer
   ```

2. Build the application:

   ```bash
   go build -o gcp-instance-explorer ./cmd/main.go
   ```

## Authentication Setup

Before running the application, you must set up authentication with Google Cloud. Choose one of the following authentication methods:

### Option 1: Using gcloud (Recommended for Development)

1. Install the [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) if you haven't already.

2. Log in with your Google account:

   ```bash
   gcloud auth login
   ```

3. Set up application default credentials:

   ```bash
   gcloud auth application-default login
   ```
   
   This will open a browser window for authentication and store credentials in:  
   `~/.config/gcloud/application_default_credentials.json`

4. Verify your credentials are working:

   ```bash
   gcloud projects list
   ```

### Option 2: Using a Service Account (Recommended for Production/CI)

1. Create a service account in the Google Cloud Console:
   - Navigate to "IAM & Admin" > "Service Accounts"
   - Click "Create Service Account"
   - Assign appropriate roles (minimum required roles are "Compute Viewer" for listing, and "Compute Instance Admin" for management functions)

2. Create and download a key for the service account (JSON format)

3. Set the environment variable to point to your key file:

   ```bash
   export GOOGLE_APPLICATION_CREDENTIALS="/path/to/your-service-account-key.json"
   ```

## Required APIs

The following Google Cloud APIs must be enabled in your project:

1. Compute Engine API
2. Cloud Resource Manager API

You can enable these APIs via the Google Cloud Console or using gcloud:

```bash
gcloud services enable compute.googleapis.com
gcloud services enable cloudresourcemanager.googleapis.com
```

## Usage

Run the application:

```bash
./gcp-instance-explorer
```

The application will:

1. Authenticate with Google Cloud using your credentials
2. Prompt you to enter a GCP Project ID
3. List all instances in the specified project with detailed information
4. Present management options:
   - Turn ON an instance
   - Turn OFF an instance
   - Replace license URL (via metadata)
   - Refresh instance list
   - Export instance list to a YAML file
   - Exit

## Management Features

### Starting an Instance

1. Select option 1 from the management menu
2. Choose the instance you want to start
3. The application will send a start request to GCP
4. The instance list will refresh automatically to show the updated status

### Stopping an Instance

1. Select option 2 from the management menu
2. Choose the instance you want to stop
3. The application will send a stop request to GCP
4. The instance list will refresh automatically to show the updated status

### Replacing License URL

1. Select option 3 from the management menu
2. Choose the instance you want to modify
3. Enter the new license URL
4. The application will update the instance metadata with the new license information

### Refreshing Instance List

1. Select option 4 from the management menu
2. The application will fetch the latest instance data from GCP

### Exporting Instance List

1. Select option 5 from the management menu
2. The application will export key instance details (name, zone, machine type, status, and licenses) to a YAML file
3. The file will be named using the project ID (e.g., `my-project-id-instances.yml`)
4. You can find the file in the directory where you ran the application

### BYOS to PAYG Mass Mover

This feature helps convert instances from Bring Your Own Subscription (BYOS) licensing to Pay As You Go (PAYG) licensing.

1. First, use the "Export list to file" option to create a YAML file of instances
2. Edit the YAML file to include only the instances you want to convert
3. Select option 3 from the management menu
4. The tool will:
   - Look for a file matching `{projectID}-instances.yml` in the current directory
   - Verify the instances exist in the current project
   - Display the current OS and license information for each instance
   - Ask for confirmation before proceeding
5. After confirmation, the tool will:
   - Apply the PAYG license code to each instance
   - Verify the conversion by checking updated license information
   - Display a summary of results

## Example Output

```
Authenticating with GCP...
Authentication successful!

Enter Project ID: my-project-id
Using project: my-project-id
Fetching instances for project my-project-id...
Found 2 instances:
1. instance-1
   Zone: us-central1-a
   Type: n1-standard-1
   Status: RUNNING
   IP: 34.123.45.67
   Disk: PERSISTENT (20 GB)
   Licenses: debian-cloud:debian-10

2. instance-2
   Zone: us-east1-b
   Type: e2-medium
   Status: TERMINATED
   IP: 
   Disk: PERSISTENT (100 GB)
   Licenses: windows-cloud:windows-server-2019-dc

Management Options:
[1] Turn ON an instance
[2] Turn OFF an instance
[3] Replace license URL
[4] Refresh instance list
[5] Export instance list to a YAML file
[0] Exit

Enter choice: 
```

## Troubleshooting

### Authentication Issues

If you encounter authentication errors:

1. Verify your credentials are set up correctly:
   ```bash
   gcloud auth list
   ```

2. Check if application default credentials exist:
   ```bash
   cat ~/.config/gcloud/application_default_credentials.json
   ```

3. If using a service account, verify the environment variable is set:
   ```bash
   echo $GOOGLE_APPLICATION_CREDENTIALS
   ```

4. Ensure the account has sufficient permissions to list projects and VM instances

### Permission Issues

If you're unable to start or stop instances, check that your account has the necessary permissions:

1. For listing instances: "Compute Viewer" role
2. For starting/stopping: "Compute Instance Admin" role

### "No instances found" Message

This might happen if:
- The project genuinely has no instances
- Your account doesn't have permissions to view instances
- The instances are in a different region/zone than expected

### API Not Enabled Errors

If you see errors about APIs not being enabled, follow the "Required APIs" section to enable them.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License. See the LICENSE file for details.
