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
