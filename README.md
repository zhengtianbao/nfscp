## About

nfscp is a tool to transfer files between host and a NFS(Network File System) directly without mount.

## Features

- Copy file from local to NFS
- * Copy file from NFS to local
- Recursively copy directory from local to NFS
- * Recursively copy directory from NFS to local
- Transfer speed limit

Note: start with * means not implemented yet.

## Installation

To install type (in the folder that contains the Makefile):

``` bash
make build
```

The target binary will be at `_output/bin/$ARCH/nfscp`.

## Usage

Copy local directory to remote NFS server with speed limit 10KB/s:

example:
```
nfscp -l 10 -r /path/to/dir 192.168.1.100:/nfs/
```

## Help

If you need any help, feel free to open a [new Issue](https://github.com/zhengtianbao/nfscp/issues/new).