from locust.locust import HttpUser, task
from locust.exception import CatchResponseError

class WebsiteUser(HttpUser):
    host = "http://localhost:8080"

    @task
    def hit_endpoint(self):
        self.client.get("/")  # Replace with your actual endpoint.
       