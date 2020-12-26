import requests
import json


def main():
    url = "http://localhost:8081/rpc"
    headers = {'content-type': 'application/json'}

    # Example echo method
    payload = {
	"service": "go.micro.srv.greeter",
        "method": "Say.Hello",
        "request": {"name": "John"},
    }
    response = requests.post(
        url, data=json.dumps(payload), headers=headers).json()

    print response

if __name__ == "__main__":
    main()
