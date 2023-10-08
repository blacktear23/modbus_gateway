# modbus_gateway

modbus_gateway is a Modbus TCP gateway for proxy Modbus requests to multiple configured backends.

# Usage

```
Usage of modbus_gateway:
  -c string
    	Config file name (default "config.yaml")
  -l string
    	Modbus TCP server listen address (default ":502")
  -t int
    	Timeout unit is ms
  -v	Show version
```

# Configuration File

```
---
unit_map:
  - unit_id: 1  # Unit ID for Gateway Server

    # Backend name for this Unit ID
    backend: Backend-1

    # Replace backend request Unit ID to this one, default is 1
    target_unit_id: 1

  - unit_id: 2
    backend: Backend-2
    target_unit_id: 1

  - unit_id: 3
    backend: Backend-3
    target_unit_id: 1

backends:
  - name: Backend-1     # Name for this backend

    # Protocol, options: `serial`, `tcp`, `tls`
    protocol: serial

    # Address for backend
    # if protocol is `tcp`, it is TCP address like `192.168.1.2:503`
    # if protocol is `serial`, it is tty file path
    address: /dev/ttyUSB0

    # Baud Rate, default 9600
    # baudrate: 9600

    # Data Bits, default 8
    # databits: 8

    # Stop Bits, default 1
    # stopbits: 1

    # Parity, options: N (None), E (Even), O (Odd); defualt N
    # parity: N

    # Timeout unit is ms default 0, 0 means no timeout
    timeout: 3000

    # Verify server's certificate, default false, only available when protocol is `tls`
    # tls_verify: false

  - name: Backend-2
    protocol: tcp
    address: 127.0.0.1:1502
    timeout: 3000

  - name: Backend-3
    protocol: tls
    address: 127.0.0.1:1503
    timeout: 3000
    tls_verify: true
```
