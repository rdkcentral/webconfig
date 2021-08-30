# webconfig
Webconfig is a solution to pass configuration data from the server to networking devices. This makes the cloud based server the master for all device configurations.


## Install go

This project is written and tested with Go **1.15**.

## Build the binary
```shell
cd $HOME/go/src/github.com/rdkcentral/webconfig
make
```
**bin/webconfig-linux-amd64** will be created. 

## Run the application
The application includes an API to notify RDK devices to download updated configurations from this server. A JWT token is required to communicate with RDK devices. Valid credentials are needed to generate JWT tokens from Comcast codebig2 service. The credentials are passed to the application through environment variables. A configuration file can be passed as an argument when the application starts. config/sample_webconfig.conf is an example. 


```shell
export SAT_CLIENT_ID='xxxxxx'
export SAT_CLIENT_SECRET='yyyyyy'
cd $HOME/go/src/github.com/rdkcentral/webconfig
bin/webconfig-linux-amd64 -f config/sample_webconfig.conf
```

## security
A pair of public/private keys can be created like this
```shell
$ openssl genrsa -out /tmp/webconfig_key.pem 2048
$ openssl rsa -in /tmp/webconfig_key.pem -pubout -outform PEM -out /tmp/webconfig_key_pub.pem
```

## API examples
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

��parameters���name�Device.WiFi.Private�value�E��private_ssid_2g��SSID�garf_private_2g�EnableøSSIDAdvertisementEnabledïprivate_ssid_5g��SSID�garf_private_5g�EnableøSSIDAdvertisementEnabledóprivate_security_2g��Passphrase�garf_pw_2g�EncryptionMethod�AES�ModeEnabled�WPA2-Personal�private_security_5g��Passphrase�garf_pw_5g�EncryptionMethod�AES�ModeEnabled�WPA2-Personal�dataType�

--2xKIxjfJuErFW+hmNCwEoMoY8I+ECM9efrV6EI4efSSW9QjI--
```

