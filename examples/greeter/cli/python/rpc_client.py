import requests
import json


def main():
    url = "http://localhost:4000"
    headers = {'content-type': 'application/json'}

    # Example echo method
    payload = {
        "method": "Say.Hello",
        "params": [{"name": "John"}],
        "jsonrpc": "2.0",
        "id": 0,
    }
    response = requests.post(
        url, data=json.dumps(payload), headers=headers).json()

    print response["result"]

if __name__ == "__main__":
    main()
