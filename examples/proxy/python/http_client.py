import requests
import json
import proxy

def main():
    response = proxy.http_call("/greeter", {"name": "John"})
    print response.text

if __name__ == "__main__":
    main()
