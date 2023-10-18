from locust import HttpUser, task
import csv
import datetime

class WebsiteUser(HttpUser):
    host = "http://localhost:8080"

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.writer = csv.writer(open('log.csv', 'w', newline=''))
        self.writer.writerow(["Timestamp", "Status Code"])

    @task
    def hit_endpoint(self):
        response = self.client.get("/")  # Replace with your actual endpoint.
        self.writer.writerow([datetime.datetime.now(), response.status_code])