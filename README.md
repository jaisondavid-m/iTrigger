# iTrigger 🚀

iTrigger is a lightweight, Go-based continuous deployment (CD) and webhook automation server. It maps GitHub repository push events (and manual dashboard triggers) to local server directories, executing custom build/deployment scripts automatically.

---

## 🛠️ Getting Started

### 1. Clone the Repository
Clone the repository to your server or local development environment:
```bash
git clone https://github.com/jaisondavid-m/iTrigger.git
cd iTrigger
```

### 2. Configure Environment Variables
Copy the example environment file and configure your values:
```bash
cp .env.example .env
```
Inside `.env`, define the **GitHub Webhook Secret** used to secure communications between GitHub and your server:
```env
GITHUB_WEBHOOK_SECRET=your_secure_random_hex_secret
```

> [!TIP]
> You can generate a strong secure secret key using the following command:
> ```bash
> openssl rand -hex 20
> ```

---

## 🚀 Running iTrigger

Choose one of the deployment methods below to start the server:

### Method A: Local (Native Go)
Ensure you have Go installed, then execute:
```bash
go run cmd/server/main.go
```
The server will start on port `8080` (e.g., `http://localhost:8080`).

### Method B: Docker Container
1. **Build the Docker Image:**
   ```bash
   docker build -t itrigger .
   ```
2. **Run the container with your `.env` configuration:**
   ```bash
   docker run -d --name itrigger --env-file .env -p 8080:8080 itrigger
   ```

### Method C: Docker Compose (With Automatic HTTPS reverse proxy via Caddy)
Ideal for production staging.
1. Make sure your `.env` contains the webhook secret.
2. If deploying to a public domain, you can configure your domain name in Caddyfile/compose.
3. Start the containers in detached mode:
   ```bash
   docker compose up --build -d
   ```
4. Stop the services:
   ```bash
   docker compose down
   ```

## 🔐 Authentication & Access Control

iTrigger includes built-in user authentication and role-based access control (RBAC) to secure your deployments.

### Default Administrator Account
Upon the first database initialization, iTrigger automatically creates a default administrator account:
* **Username:** `itrigger`
* **Password:** `itrigger`

> [!IMPORTANT]
> For security, log in immediately and update the password under the **Settings** tab.

### Roles and Permissions
Administrators can configure granular access controls for other users in the **Users** sidebar tab:
1. **System Roles:**
   - **Administrator:** Full system access (includes viewing webhook payload logs, managing user accounts, browsing the server filesystem, and performing backups).
   - **User:** Restricted access. Can be restricted from creating new projects and can only view/manage projects explicitly assigned to them.
2. **Project-Level Permissions:**
   - **No Access:** The project is completely hidden from the user's dashboard.
   - **Read-Only:** The user can see the project card and click **View Details** to inspect the configuration, but cannot edit settings, trigger deployments, or delete the project.
   - **Write / Trigger:** The user can edit the project configuration and click **Deploy Now** to trigger deployments, but cannot delete the project.
   - **Write / Trigger / Delete:** The user has full control over the project, including editing, deploying, and deleting it. *(Project creators automatically receive this level for their created projects).*

---

## ⚙️ How to Add a Project in the Dashboard UI

1. Open your web browser and navigate to the iTrigger dashboard (e.g., `http://localhost:8080`).
2. Log in using your credentials.
3. Click **Add Project** in the top right (visible to admins or users allowed to create projects).
4. Provide the project configuration:
   - **Project Display Name:** A unique label for the dashboard UI (e.g., `Core Web Application`).
   - **GitHub Repository:** The owner and repository name (e.g., `owner/repo-name` or `repo-name`).
   - **Target Branch:** The branch name to trigger deployments for (e.g., `main`).
   - **Server Directory Path:** The absolute directory path on the server where deployment actions execute. You can click **Browse...** to interactively navigate the filesystem on the server to select the directory.
   - **Deployment Execution Mode:**
     - *Auto-detect `.itrigger`:* Resolves and executes a `.itrigger`, `.itrigger.sh`, or `itrigger.sh` script in the root of the pulled repository on the server.
     - *Custom Script:* Select this to type or paste shell commands directly into the textarea (e.g., `git pull origin main && docker compose up -d --build`).
   - **Enable Auto-Deploy:** Toggle on to allow incoming GitHub webhooks to run this configuration automatically.
4. Click **Save Project**.

---

## 🔗 Connecting Projects to GitHub Webhooks

To automate deployments when code is pushed:

1. Navigate to your repository page on GitHub.
2. Go to **Settings** -> **Webhooks** -> **Add webhook**.
3. Fill out the webhook form:
   - **Payload URL:** `https://your-domain.com/api/webhooks/github` (Note: GitHub webhooks require HTTPS. For local testing, use a tunneling service like `ngrok` or `localtunnel` to expose port `8080`).
   - **Content type:** `application/json` (Required)
   - **Secret:** Enter the exact value of your `GITHUB_WEBHOOK_SECRET` defined in the `.env` file.
   - **Which events:** Select **Just the push event**.
4. Click **Add webhook**.

---

## ⚡ Triggering Deployments

iTrigger supports two execution trigger modes:

### 1. Automatic Deployments (via GitHub Webhook)
Whenever code is pushed to the target repository and branch on GitHub:
* GitHub sends a push event payload signed with your webhook secret.
* iTrigger verifies the signature to ensure authenticity, matches the repository and branch name against active projects, and executes the script in the background.

### 2. Manual Deployments (via UI)
* Go to the **Projects** tab.
* Find the target project card and click **Deploy Now**.
* iTrigger will launch the execution runner instantly and pop open the real-time **Console Log Viewer** showing output logs.

---

## 📊 Monitoring & Troubleshooting

* **Dashboard Analytics:** Review configured projects, total executions, success rates, and failures on the top metrics panel.
* **Console Logs:** Go to the **Deployment Logs** tab and click **View Log** on any deployment entry to view the complete script stdout/stderr terminal log outputs.
* **Error Diagnosis:** If a deployment fails, click **Why Failed?** to see detailed logs and diagnostic metadata.
* **Webhook Payloads:** Inspect raw incoming requests in the **Webhook Payloads** tab to ensure GitHub is delivering payloads to the server correctly.