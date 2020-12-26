import requests
import json
import proxy

def main():
    response = proxy.rpc_call("/greeter/say/hello", {"name": "John"})
    print response

if __name__ == "__main__":
    main()
