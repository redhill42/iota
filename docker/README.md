Pull docker image and run:

  ```shell
  $ docker pull icloudway/iota
  $ docker run --name iota-server -d -p 8080:8080 -p 1883:1883 -p 8086:8086 icloudway/iota
  ```

Add a user to iota server:

  ```shell
  $ docker exec iota-server /app/bin/iota useradd admin admin
  ```

Grab command line interface binary from the container:

  ```shell
  $ docker cp iota-server:/app/bin/iotacli .
  ```

Login to iota server:

  ```shell
  $ ./iotacli -H http://localhost:8080 login
  ```

Use command line interface to interact to iota server:

  ```shell
  $ ./iotacli version
  $ ./iotacli device:create foo
  $ ./iotacli device
  ```
