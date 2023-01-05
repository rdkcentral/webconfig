# webconfig

This project is to implement a configuration management server. RDK devices download configurations from this server during bootup or notified when updates are available.

## Transport route
Webconfig supports 2 types of transport between cloud and devices
1. http / webpa
2. mqtt

## Install go
This project is written and tested with Go **1.17**.

## Build the binary
```shell
cd $HOME/go/src/github.com/rdkcentral/webconfig
make
```
**bin/webconfig-linux-amd64** will be created. 

## Configuration
A sample configuration file can be found at config/sample_webconfig.conf

### Setup an encryption key
If we want to encrypt the data in db for security, we can set a security key through an environment variable. A command like this generate a random key in base64
```shell
$ head -c 32 /dev/random | base64
ABCDEF...
```
Specify the environ variable in the config
```shell
webconfig {
    security {
        encryption_key_env_name = "WEBCONFIG_KEY"
    }
```
The subdocs that need encryption can be specified in the config
```shell
    encrypted_subdoc_ids = [ "privatessid", "homessid", "telcovoip", "voiceservice" ]
```

### Configurations for connecting to mqtt server
Webconfig is designed to work with an "http-collector" service. It includes full MQTT broker capabilities and a REST api interface. The endpoint needs to be properly configure in the "mqtt" section of the config

### Configurations for connecting to webpa server and JWT authentication
Webconfig works with rdk devices through webpa. "webpa" and "jwt" sections need to be properly configured. If this route is not used, then the jwt should be disabled.

### Configuration for kafka
Devices use mqtt or webpa route send message to webconfig cloud. Webconfig supports 3 types of kafka message
1. webpa state
2. mqtt get
3. mqtt state
The actual kafka topic names can be any. Below is just an example. "topics" should include all topics. "ratelimit" can be tuned to users env.
```shell
    kafka {
        enabled = false
        brokers = "localhost:9092"
        topics = "config-version-report"
        use_random_consumer_group = false
        consumer_group = "webconfig"
        assignor = "roundrobin"
        oldest = false
        ratelimit {
            messages_per_second = 10
        }
        clusters {
            mesh {
                enabled = false
                brokers = "localhost:19092"
                topics = "staging-chi-onewifi-from-device"
                use_random_consumer_group = false
                consumer_group = "webconfig"
                assignor = "roundrobin"
                oldest = false
                ratelimit {
                    messages_per_second = 10
                }
            }
        }
```

### Configuration for database
The main database operations are defined as an interface. Any driver that implements the interface should work. We has implemented using sqlite, cassandra and yugabytedb. After the db is properly configured, the dbinit.cql can be used to create the tables for cassandra.



## Run the application
```shell
$ export WEBCONFIG_KEY='ABCDEF...'
$ mkdir -p /app/logs/webconfig
$ cd $HOME/go/src/github.com/rdkcentral/webconfig
$ bin/webconfig-linux-amd64 -f config/sample_webconfig.conf
```

## APIs
### Version API
This display the build date and code commit info
```shell
curl http://localhost:9000/api/v1/version
{"status":200,"message":"OK","data":{"code_git_commit":"2ac7ff4","build_time":"Thu Feb 14 01:57:26 2019 UTC","binary_version":"317f2d4","binary_branch":"develop","binary_build_time":"2021-02-10_18:26:49_UTC"}}
```

### Write data into DB
The POST API is designed to accept binary input "Content-type: application/msgpack". A "group_id" is mandatory in the query parameter to specify the subdoc the input is meant for. Most programming languages supports HTTP would accept binary data as POST body. The example uses curl and reads the binary data from a file.
```shell
curl -s -i "http://localhost:9000/api/v1/device/010203040506/document?group_id=privatessid" -H 'Content-type: application/msgpack' --data-binary @privatessid.bin -X POST
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 14 Oct 2020 22:58:21 GMT
Content-Length: 29

{"status":200,"message":"OK"}
```

### Verify data in DB
The GET API read binary data in the response. The "group_id" is mandatory in the query parameter. For simplicity, the binary output is saved as a file. We can compare the 2 files to verify.
```shell
curl -s -i "http://localhost:9000/api/v1/device/010203040506/document?group_id=privatessid" > result.bin

cmp privatessid.bin result.bin
```

### Poke RDK device to download the configuration
When data are prepared in DB, users are expected to call this poke API. Webconfig uses the RDK webpa service to prompt the webconfig client on the devices to download the prepared configurations.
```shell
curl -s "http://localhost:9009/api/v1/device/010203040506/poke" -X POST

```

### RDK devices downloads the configuration
RDK devices use this API to fetch data. The response is in HTTP multipart. Each part maps to a subdoc, or a logical group of configurations encoded in msgpack.
```shell
curl -s "http://localhost:9000/api/v1/device/010203040506/config"
HTTP/1.1 200 OK
Content-Type: multipart/mixed; boundary=2xKIxjfJuErFW+hmNCwEoMoY8I+ECM9efrV6EI4efSSW9QjI
Etag: 2484248953
Date: Sun, 18 Oct 2020 19:19:34 GMT
Content-Length: 578

--2xKIxjfJuErFW+hmNCwEoMoY8I+ECM9efrV6EI4efSSW9QjI
Content-type: application/msgpack
Etag: 1737259797
Namespace: privatessid

��parameters���name�Device.WiFi.Private�value�^AE��private_ssid_2g��SSID�garf_private_2g�EnableøSSIDAdvertisementEnabledïprivate_ssid_5g��SSID�garf_private_5g�EnableøSSIDAdvertisementEnabledóprivate_security_2g��Passphrase�garf_pw_2g�EncryptionMethod�AES�ModeEnabled�WPA2-Personal�private_security_5g��Passphrase�garf_pw_5g�EncryptionMethod�AES�ModeEnabled�WPA2-Personal�dataType�

--2xKIxjfJuErFW+hmNCwEoMoY8I+ECM9efrV6EI4efSSW9QjI--
```

### Send new configurations to device through mqtt
When data are prepared in DB, users are expected to call this poke API. Webconfig uses the RDK webpa service to prompt the webconfig client on the devices to download the prepared configurations.
```shell
curl -s "http://localhost:9009/api/v1/device/010203040506/poke?route=mqtt" -X POST
```


