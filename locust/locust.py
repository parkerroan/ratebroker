from locust import HttpUser, task, between
import uuid
import csv
import datetime
import os

# Global variables to handle the log file
log_file_path = 'log.csv'
file_is_empty = not os.path.exists(log_file_path) or os.path.getsize(log_file_path) == 0

# Open the file once and create a writer accessible to all user instances
with open(log_file_path, 'a', newline='') as logfile:
    writer = csv.writer(logfile)
    if file_is_empty:  # Only write the header if the file is empty
        writer.writerow(["Timestamp", "Status Code", "User ID", "Latency"])

class WebsiteUser(HttpUser):
    host = "http://localhost:8080"
    wait_time = between(.04, .05)  # ~20 rps

    def on_start(self):
        # self.user_id = str(uuid.uuid4())  # Replace with actual user data or parameterize as needed.
        self.user_id = "user-1" 

    @task
    def hit_endpoint(self):
        with self.client.get("/", catch_response=True, headers={"X-User-ID": self.user_id}) as response:
            request_latency = response.elapsed.total_seconds()
            current_time = datetime.datetime.now().utcnow().isoformat()

            # Using the global writer, write the request details to the CSV file.
            with open(log_file_path, 'a', newline='') as logfile:
                writer = csv.writer(logfile)
                writer.writerow([current_time, response.status_code, self.user_id, request_latency])

    # No need for on_stop to close the file, as it's being handled at a global level.

# Ensure you handle exceptions and errors as necessary for your use case.
