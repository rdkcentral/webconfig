# webconfig

This project is to implement a configuration management server. RDK devices download configurations from this server during bootup or notified when updates are available.

## Transport route
Webconfig supports 2 types of transport between cloud and devices
1. http / webpa
2. mqtt

## Install go
This project is written and tested with Go **1.21**.

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

#### Kafka TLS/SSL Configuration

Webconfig supports secure TLS/SSL connections to Kafka brokers for both consumers and producers. This is recommended for production environments to ensure data encryption in transit and proper authentication.

**TLS Configuration Options:**

- `tls.enabled` - Enable/disable TLS for Kafka connections (default: false)
- `tls.cert_file` - Path to client certificate file for mTLS authentication (optional)
- `tls.key_file` - Path to client private key file for mTLS authentication (optional)
- `tls.ca_cert_file` - Path to CA certificate file for broker verification (optional)
- `tls.insecure_skip_verify` - Skip certificate verification (insecure, for testing only, default: false)

**Consumer TLS Configuration Example:**

```shell
    kafka {
        enabled = true
        brokers = "kafka-broker:9093"  # Use secure port
        topics = "config-version-report"
        consumer_group = "webconfig"

        tls {
            enabled = true
            cert_file = "/etc/webconfig/kafka/client.crt"
            key_file = "/etc/webconfig/kafka/client.key"
            ca_cert_file = "/etc/webconfig/kafka/ca.crt"
            insecure_skip_verify = false
        }

        # Per-cluster TLS configuration
        clusters {
            mesh {
                enabled = true
                brokers = "kafka-mesh:9093"
                topics = "staging-chi-onewifi-from-device"

                tls {
                    enabled = true
                    cert_file = "/etc/webconfig/kafka/mesh-client.crt"
                    key_file = "/etc/webconfig/kafka/mesh-client.key"
                    ca_cert_file = "/etc/webconfig/kafka/mesh-ca.crt"
                }
            }
        }
    }
```

**Producer TLS Configuration Example:**

```shell
    kafka_producer {
        enabled = true
        brokers = "kafka-broker:9093"
        topic = "webconfig_downstream"

        tls {
            enabled = true
            cert_file = "/etc/webconfig/kafka/producer-client.crt"
            key_file = "/etc/webconfig/kafka/producer-client.key"
            ca_cert_file = "/etc/webconfig/kafka/ca.crt"
        }
    }
```

**Certificate Requirements:**

1. **Client Certificate (mTLS)**: If `cert_file` and `key_file` are provided, mutual TLS authentication is enabled. The certificate and key must be in PEM format.

2. **CA Certificate**: If `ca_cert_file` is provided, it will be used to verify the Kafka broker's certificate. This is useful when using self-signed certificates or internal CAs.

3. **Certificate Validation**: All certificate files are validated at startup. The application will fail to start with clear error messages if:
   - Certificate files are missing or unreadable
   - Certificates are in invalid format
   - Certificate and key don't match

**TLS Security Best Practices:**

1. **Use TLS in Production**: Always enable TLS for production Kafka connections to encrypt data in transit
2. **Use mTLS**: Provide client certificates (`cert_file` and `key_file`) for mutual authentication
3. **Verify Certificates**: Never use `insecure_skip_verify = true` in production - it disables certificate verification and is insecure
4. **Protect Certificate Files**: Set appropriate file permissions (0600) on certificate and key files
5. **Use Secure Ports**: Configure Kafka brokers to listen on secure ports (typically 9093 for TLS)
6. **Certificate Rotation**: Plan for certificate rotation - the service must be restarted to pick up new certificates

**Troubleshooting TLS Issues:**

- Check logs for TLS-related errors during startup
- Verify certificate file paths are correct and files are readable
- Ensure Kafka brokers are configured to accept TLS connections
- Test certificate validity: `openssl x509 -in client.crt -text -noout`
- Verify certificate and key match: `openssl x509 -noout -modulus -in client.crt | openssl md5` vs `openssl rsa -noout -modulus -in client.key | openssl md5`

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
curl -s -i "http://localhost:9000/api/v1/device/010203040506/document/privatessid" -H 'Content-type: application/msgpack' --data-binary @privatessid.bin -X POST
HTTP/1.1 200 OK
Content-Type: application/json
Date: Wed, 14 Oct 2020 22:58:21 GMT
Content-Length: 29

{"status":200,"message":"OK"}
```

### Verify data in DB
The GET API read binary data in the response. The "group_id" is mandatory in the query parameter. For simplicity, the binary output is saved as a file. We can compare the 2 files to verify.
```shell
curl -s -i "http://localhost:9000/api/v1/device/010203040506/document/privatessid" > result.bin

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


