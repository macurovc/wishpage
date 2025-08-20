# Family Wishlist Application

This is a full-stack web application designed to manage a family wishlist. It features a React frontend, a Node.js/Express backend, and a SQLite database. The entire application is containerized using Docker for easy deployment.

## Architecture

The application is composed of two main parts:

*   **Frontend:** A single-page application (SPA) built with React and Vite. It uses `zustand` for state management and `axios` for API communication.
*   **Backend:** A RESTful API built with Node.js, Express, and TypeScript. It uses a SQLite database for data persistence.

The application is designed to be built and run as a single container using Docker. The Dockerfile uses a multi-stage build to create a small, optimized production image.

## Project Structure

The project is organized into two main directories: `frontend` and `backend`.

### Frontend

```
frontend/
├── src/
│   ├── components/     # React components
│   ├── api.ts          # Functions for making API calls
│   ├── App.tsx         # Main application component
│   ├── main.tsx        # Entry point for the React application
│   └── store.ts        # Zustand store for state management
├── vite.config.ts    # Vite configuration
└── package.json      # Frontend dependencies and scripts
```

### Backend

```
backend/
├── src/
│   ├── middleware/     # Express middleware
│   ├── utils/          # Utility functions
│   ├── db.ts           # Database connection and setup
│   ├── index.ts        # Entry point for the backend server
│   └── routes.ts       # API routes
├── Dockerfile        # Docker configuration for the entire application
└── package.json      # Backend dependencies and scripts
```

## Building and Running in Production (Docker)

To build and run the application in a production environment, you will need to have Docker installed.

1.  **Build the Docker image:**

    ```bash
    docker build -t family-wishlist .
    ```

2.  **Run the Docker container:**

    ```bash
    docker run -p 3001:3001 family-wishlist
    ```

The application will be accessible at [http://localhost:3001](http://localhost:3001).

### Building for a specific architecture

This repository is configured to build a Docker image that can be run on different CPU architectures (e.g., ARM and x86/amd64). To build the image for a specific architecture and load it into your local Docker images, you can use the `docker buildx` command with the `--platform` and `--load` flags.

For example, to build the image for an x86/amd64 architecture, run the following command:

```bash
docker buildx build --platform linux/amd64 --load -t family-wishlist:latest-amd64 .
```

To build for an ARM architecture, you can use:

```bash
docker buildx build --platform linux/arm64 --load -t family-wishlist:latest-arm64 .
```

This will create a Docker image that is compatible with the specified architecture and load it into your local image repository.

### Setting Up Docker Buildx

Docker Buildx is a CLI plugin that extends the Docker command with the full support of the features provided by the Moby BuildKit builder toolkit. It provides users with a familiar user experience to `docker build` with many new features like creating scoped builder instances and building against multiple nodes concurrently.

#### Enabling Docker Buildx

**macOS and Windows**

Docker Buildx is included by default with Docker Desktop for Windows and macOS. No additional installation is required.

**Linux**

For Linux distributions, you will need to install the `docker-buildx-plugin` package.

For Debian-based distributions (e.g., Ubuntu):
```bash
sudo apt-get update
sudo apt-get install docker-buildx-plugin
```

For Red Hat-based distributions (e.g., CentOS):
```bash
sudo yum install docker-buildx-plugin
```

After installation, you may need to restart the Docker service:
```bash
sudo systemctl restart docker
```

## Environment Variables

The application uses environment variables for configuration.

| Variable              | Description                                                                 | Default (Dev)      | Required in Prod |
| --------------------- | --------------------------------------------------------------------------- | ------------------ | ---------------- |
| `PORT`                | The port for the backend server.                                            | `3001`             | No               |
| `DATABASE_FILENAME`   | The file path for the SQLite database.                                      | `./wishlist.db`    | No               |
| `EDIT_PASSWORD`       | The password to enable edit mode for adding/deleting items and members.     | `admin`            | **Yes**          |
| `EMAIL_USER`          | The email address used to send notifications (e.g., your Gmail address).    | `(none)`           | No               |
| `EMAIL_PASS`          | The password or app-specific password for the `EMAIL_USER` account.         | `(none)`           | No               |
| `EMAIL_TO`            | The recipient email address for notifications.                              | `(none)`           | No               |
| `EMAIL_HOST`          | SMTP host for the email service (e.g., `smtp.gmail.com`).                   | `smtp.gmail.com`   | No               |
| `EMAIL_PORT`          | SMTP port for the email service (e.g., `587` for TLS, `465` for SSL).       | `587`              | No               |
| `EMAIL_FROM`          | Optional: The 'From' email address for notifications. Defaults to `EMAIL_USER`. | `(none)`           | No               |

### Email Notifications

The backend can send email notifications for database write transactions (e.g., adding/deleting items, reserving items, adding/deleting family members). To enable this feature, configure the following environment variables:

*   **`EMAIL_USER`**: Your email address that will be used to send the notifications. For Gmail, this is your Gmail address.
*   **`EMAIL_PASS`**: The password for your `EMAIL_USER` account. **For Gmail, it is highly recommended to use an App Password** instead of your regular account password for security reasons. You can generate an App Password in your Google Account security settings.

#### Generating a Gmail App Password

To generate an App Password for your Gmail account (required if you have 2-Step Verification enabled, which is highly recommended):

1.  Go to your Google Account: [myaccount.google.com](https://myaccount.google.com/).
2.  In the left navigation panel, click **Security**.
3.  Under "How you sign in to Google," click **2-Step Verification**. (If it's off, you'll need to turn it on first).
4.  Scroll down to "App passwords" and click on it.
5.  You may be asked to sign in to your Google Account again.
6.  Under "Select app" and "Select device," choose **Mail** and **Other (Custom name...)** respectively. For the custom name, you could enter something like "Family Wishlist App".
7.  Click **Generate**.
8.  A 16-character password will appear in a yellow bar. This is your App Password. **Copy this password immediately**, as you won't see it again.
9.  Use this generated password as the value for `EMAIL_PASS` in your `.env` file or environment variables.

*   **`EMAIL_TO`**: The email address where you want to receive the notifications. This can be the same as `EMAIL_USER` or a different address.
*   **`EMAIL_HOST`**: The SMTP host of your email provider. Defaults to `smtp.gmail.com`.
*   **`EMAIL_PORT`**: The SMTP port. Defaults to `587` (for TLS). Use `465` for SSL.
*   **`EMAIL_FROM`**: (Optional) The 'From' address that will appear in the notification emails. If not set, `EMAIL_USER` will be used.

**Note on other email services (ProtonMail, Tutanota):**
While the system is designed to be flexible with SMTP settings, ProtonMail and Tutanota often use custom clients or bridge applications for email sending, or have specific SMTP configurations that might require additional setup beyond simple SMTP credentials. For these services, you may need to consult their official documentation for SMTP details or consider using their provided bridge applications. Gmail with an App Password is generally the most straightforward to configure.

### Development

In development, these variables are loaded from a `.env` file in the `backend` directory. Create a file named `backend/.env` with the following content:

```
# backend/.env
PORT=3001
DATABASE_FILENAME=./wishlist.db
EDIT_PASSWORD=admin
```

### Production (Docker)

For production, you should pass these variables to the `docker run` command. It is **strongly recommended** to use a secure, randomly generated password for `EDIT_PASSWORD`.

```bash
docker run -d \
  -p 3001:3001 \
  -e PORT=3001 \
  -e DATABASE_FILENAME=/data/wishlist.db \
  -e EDIT_PASSWORD=your_super_secret_password \
  -v $(pwd)/data:/data \
  --name family-wishlist \
  family-wishlist
```
*   The `-e` flag sets an environment variable.
*   The `-v` flag mounts a local directory (`./data`) into the container at `/data`. This is crucial for persisting the SQLite database file outside the container. Without this, your data will be lost if the container is removed.

## Running in Development Mode

For development, you will need to run the frontend and backend servers separately.

### Backend

1.  **Navigate to the backend directory:**

    ```bash
    cd backend
    ```

2.  **Create a `.env` file** (see Environment Variables section above).

3.  **Install dependencies:**

    ```bash
    npm install
    ```

4.  **Run the development server:**

    ```bash
    npm run dev
    ```

The backend server will start on the port defined in your `.env` file (default: 3001).

### Frontend

1.  **Navigate to the frontend directory:**

    ```bash
    cd frontend
    ```

2.  **Install dependencies:**

    ```bash
    npm install
    ```

3.  **Run the development server:**

    ```bash
    npm run dev
    ```

The frontend development server will start on a different port (usually 5173) and will proxy API requests to the backend server.
