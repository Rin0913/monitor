# Health Monitor by Go

This is a health monitor like Nagios implemented by Golang.

You can simply use following commands as root:

Run:
```
make up
```

Run tests:
```
make test
```

Notice that you must have installed docker and docker compose on your server.

---
## API

`GET /health`: an API endpoint that allows you to test if you could connect the server properly.

`GET /devices`: list all the devices, and each device would be a service that you want to test.

`POST /devices`: add a testing target and return its deviceID. You should provide a json payload like following.

```json
{
  "address": "sandb0x.tw:80",
  "check_method": "tcp_check",
  "interval_sec": 10
}
```

Notice that `check_method` and `interval_sec` are optional with default value `tcp_check` and 10.

`GET /devices/{deviceID}`: get the health status of the device.

### Internal API

For workers. Authentication required. You can deploy other workers.

`POST /internal/worker/jobs/poll`: Get an active job.
`POST /internal/worker/jobs/report`: Report the result of the job.

The worker is not finished now. (It hasn't even started yet.)

---

## Features
1. You can use customized test tool by specifying in `checkers.yaml`.
2. API `GET /devices/{address}` was subtituded by `GET /devices/{deviceID}` because it allows to test one address by different tools.
3. You can implement a third-party worker by using the provided internal APIs. However, there's a internal worker.
