import requests

# Define the API endpoint
url = "http://localhost:5050/users"

# Define the user data to send in the request
user_data = {
    "email": "testuser@example.com",
    "password": "securepassword123",
    "id": "12345"  # Assuming your API requires an ID
}

# Send the POST request
response = requests.post(url, json=user_data)

# Print the response
if response.status_code == 201:  # Assuming 201 Created is returned on success
    print("User created successfully:", response.json())
else:
    print("Failed to create user:", response.status_code, response.text)