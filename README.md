# Delegatio

Delegatio is a framework that can be used to manage homework of system security classes. The aim is to provide a infrastructure to let students work on problems independent of their hardware. 

# Installation
```bash
pacman -S libvirt qemu-full go mkosi make cmake 
```
`systemd 253` or newer is required to build the images, otherwise a local systemd tree is needed [mkosi issue](https://github.com/systemd/mkosi/issues/1290)

# Build
```bash
mkdir build
cd build
cmake ..
make
```

# Run
```bash
./cli --path=../images/image.qcow2
```
By default the ssh image will be pulled from Github, and deployed in Kubernetes. For testing you can also start the ssh binary locally with an exported kubeconfig `export KUBECONFIG=/path/to/admin.conf`.

Connecting is possible by sshing into the daemon, either on the kubernetes nodes or on localhost.
```bash
ssh testchallenge2@localhost -p 2200 -i ~/.ssh/id_rsa
```
You must provide your public keys in `./internal/config/global.go` (will be changed to read a config file soon) 

## TODO
* Unittests
* Abstract storage
* Webserver to deploy a website to generate ssh keys and sync them with the ssh daemon
* Support for multiple control planes
* Harden Kubernetes Pods
