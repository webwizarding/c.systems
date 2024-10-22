# DTQSys
---


- **Database Credentials**: Ensure the `.env` file contains your actual PostgreSQL credentials.
- **JWT Secret Key**: Replace `"your_secure_jwt_secret_key"` in the `.env` file with a secure, randomly generated secret key.
- **Dependencies**: Run `go mod tidy` to download all the dependencies specified in `go.mod`.
- **Redis and PostgreSQL**: Ensure both Redis and PostgreSQL services are running on your machine.
- **Certificates**: Place your self-signed certificates in the `cert/` directory.
- **Environment Variables**: Make sure to load environment variables appropriately, especially in production environments.
- **CORS Configuration**: Adjust the allowed origins in the CORS settings as needed for your frontend application.

---

### **Running the Application**

1. **Start Redis**:

   ```bash
   redis-server
   ```

2. **Start PostgreSQL**:

   - Ensure the PostgreSQL service is running.
   - Create the `task_queue` database and the `tasks` and `users` tables as described.

     **SQL to Create the `tasks` Table**:

     ```sql
     CREATE TABLE tasks (
         id SERIAL PRIMARY KEY,
         task_id VARCHAR(255) UNIQUE,
         data TEXT,
         status VARCHAR(50),
         created TIMESTAMP,
         retries INT,
         priority INT
     );
     ```

     **SQL to Create the `users` Table**:

     ```sql
     CREATE TABLE users (
         id SERIAL PRIMARY KEY,
         username VARCHAR(50) UNIQUE NOT NULL,
         password_hash TEXT NOT NULL,
         created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
     );
     ```

3. **Run the Application**:

   ```bash
   go run main.go
   ```

4. **Register a User**:

   ```bash
   curl --insecure -X POST https://localhost:8443/register \
    -H "Content-Type: application/json" \
    -d '{"username": "testuser", "password": "testpassword"}'
   ```

5. **Log In to Obtain Tokens**:

   ```bash
   curl --insecure -X POST https://localhost:8443/login \
    -H "Content-Type: application/json" \
    -d '{"username": "testuser", "password": "testpassword"}'
   ```

6. **Submit a Task Using the Access Token**:

   ```bash
   curl --insecure -X POST https://localhost:8443/tasks \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer your_access_token" \
    -d '{"data": "Authenticated Task", "priority": 2}'
   ```

7. **Retrieve Tasks**:

   ```bash
   curl --insecure -X GET https://localhost:8443/tasks \
    -H "Authorization: Bearer your_access_token"
   ```

8. **Monitor Workers**:

   ```bash
   curl --insecure -H "Authorization: Bearer your_access_token" https://localhost:8443/workers
   ```

9. **Access Metrics**:

   - Prometheus Metrics Endpoint: `https://localhost:8443/metrics` (may need to adjust security settings)
   - Prometheus UI: `http://localhost:9090`
   - Grafana Dashboard: `http://localhost:3000`

---
