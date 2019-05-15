<p align="center">
  <img src="https://github.com/mgren/ogive/raw/master/logo/logo.png" alt="ogive logo">
  <br>
  Secure backups with AWS S3 Glacier Deep Archive
  <br>
  <br>
  <p align="center">
    <a href="https://goreportcard.com/report/github.com/mgren/ogive">
      <img src="https://goreportcard.com/badge/github.com/mgren/ogive" alt="goreportcard badge">
    </a>
  </p>
</p>
<br>
----

Ogive is a simple commandline tool for storing and retrieving cryptographically secure backups from [AWS S3 Glacier Deep Archive](https://aws.amazon.com/blogs/aws/new-amazon-s3-storage-class-glacier-deep-archive/).

Ogive encrypts all data and metadata (original filename) before uploading the file to S3 and decrypts it upon retrieval. Each file upload has its own, unique encryption key derived from the master key. Only the non-secret nonce used for key derivation is stored together with the encrypted file. Neither the master key nor the derived key are ever uploaded to any AWS service. The master key together with AWS credentials used for S3 operations is stored locally, in portable profile files. Those files are in turn indirectly (using [Argon2](https://www.argon2.com) KDF) secured with an user-provided password. Why not just use KMS? [No reason.](https://i.imgur.com/T5cKGDr.jpg)

## Installation
This project requires go 1.11.0 or newer. Assuming the go binary is available in $PATH and a valid $GOPATH exists:

```sh
$ go get -d github.com/mgren/ogive
$ cd $GOPATH/src/github.com/mgren/ogive
$ make
$ make install
```

## Configuring AWS
* A separate, dedicated bucket for ogive is recommended, but not necessary. The list command will skip any files whose Content-Type is not application/x-ogive.
* Ogive uploads objects with private ACLs. Nevertheless, bucket configuration should block uploading public objects and remove public access (those are the default and recommended settings when creating an S3 bucket in the AWS Console).
* Enabling bucket encryption is not necessary, as stored data is already encrypted. There are, however, no arguments against doing it - the locally stored key and the key used by S3 will be different.
* **It is essential to configure a lifecycle rule that automatically cancels incomplete multipart uploads.** Ogive will not keep track of failed uploads in case a critical network failure or other issues prevent upload completion.

The following is a minimal IAM Policy for ogive:
```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "s3:GetObject",
                "s3:RestoreObject"
            ],
            "Resource": "arn:aws:s3:::BUCKET_NAME/*"
        },
        {
            "Effect": "Allow",
            "Action": "s3:ListBucket",
            "Resource": "arn:aws:s3:::BUCKET_NAME"
        }
    ]
}
```

## Examples
#### Basic Example
```sh
$ ogive init
...
# enter profile information
$ ogive put example.dat
...
$ ogive list
SIZE       DATE         STATUS STORAGE ID FILENAME
---------- ------------ ------ ---------- --------
123.4 MiB  2019-May-05  DEEPS  5PqBqHILQoIckevFn5EbXX1yGrgXIgAY2UvWT5ruYD-YOCGNMGEM Example1.dat
123.4 GiB  2019-May-03  READY  5hC4jbOhHGpF-j5wIO9aLkgRAAWw9wzpvpN9pvGdbjX.1wPAHJQe Example2.dat
123.4 KiB  2019-May-04  RECOV  liSROji4FYcW6MVr0fzrdeNJnOHeOL7qRsHHY88cpTmnDQDRZ99M Example3.dat

# find example.dat on the list to determine its storage_id
$ ogive restore <storage_id>
...
$ ogive head <storage_id>
READY
# Bulk Restore usually completes within 48 hours
$ ogive get <storage_id> /directory/to/save-in
...
```

#### Writing To/From a Block Device
```sh
$ ogive put /dev/sdf
...
$ ogive restore <storage_id> /dev # --output=sdg to write to sdg instead of sdf
```

#### Supplying a Password From Stdin
```sh
$ bash securely-retrieve-password-and-write-to-stdout.sh | ogive put example.dat
```

#### Restore All Archives
```sh
$ bash securely-retrieve-password-and-write-to-stdout.sh | ogive list | \
> awk 'NR>2 && $4 == "DEEPS" {print $5}' | while read id;
> do bash securely-retrieve-password-and-write-to-stdout.sh | ogive restore $id;
> done
```

## Usage
All of the following information is also available as a manpage.

Global flags available for any subcommand:
```
  -h, --help             help for ogive
  -p, --profile string   Location of Ogive profile file. (default "$HOME/.ogive")
```

### get
Download and decrypt file, saving it under its original filename.

```sh
$ ogive get <source_file> <destination_directory> [flags]
```

##### flags
```
  -o, --output string   Override destination filename.
```

### head
Head a specific Ogive file on S3 and retrieve its current archival status. 

```sh
$ ogive head <storage_id> [flags]
```

Following exit codes and file statuses are possible for this command: 

| Code | Description |
| ------ | ------ |
| 0 | file available for download |
| 1 | error occured |
| 2 | file not available for download |

| Status | Description |
| ------ | ------ |
| IDEEPS | file is in DEEP_ARCHIVE state with no restore job pending |
| RECOV | file is currently being restored |
| READY | file has been restored into STANDARD storage and is ready for downloading |
| \?\?\?\?\? | file state is unrecognized |

### init
Set up an Ogive profile, including generating the master key and providing the S3 bucket location.

```sh
$ ogive init [flags]
```

##### flags
```
  -r, --reinit   Reinitialize an existing profile to change password and/or AWS keys. Old profile is stored as "<name>.bak".
```

### list
Lists all Ogive archives in an S3 bucket. Lists entire bucket and HEADs each file to retrieve metadata.

```sh
$ ogive list [flags]
```

### put
Encrypt and upload file to S3 Glacier Deep Archive.

```sh
$ ogive put <source_file> [flags]
```

### restore
Initiate file recovery from Deep Archive. Bulk Restore is used. Use _head_ command to verify when the file becomes ready for download.

```sh
$ ogive restore <storage_id> [flags]
```

##### flags
```
  -t, --lifetime int   Specifies the number of days to retain the restored object before returning it to Deep Archive. (default 1)
```

## Notes
#### Progress Reporting
When running the _get_ or _put_ commands, ogive will report an approximate progress. For file uploads this is highly inaccurate for objects smaller than 550 MiB. This is because aws-sdk-go lacks progress reporting in its s3manager, so this program relies on the amount of bytes read by the manager instead. Users should always wait for the program to exit gracefully instead of relying solely on the progress bar.

#### Multiple Backup Versions
Since each _put_ generates an unique nonce and derives an unique name, the probability of name collision in storage is basically zero. This allows to _put_ the same file multiple times at different points in time to create multiple backups.

#### About the profile file
Since the profile file stores the master key, its loss or corruption renders all backups created with it unrecoverable. A copy of the profile file on a separate medium is essential. An additional, physical backup of the profile file such as [PaperBack](http://ollydbg.de/Paperbak/) is suggested.

#### Broken Downloads/uploads
Currently ogive does not support any form of download/upload resumption. One file operation must complete in one run. This is unlikely to change, at least until [sio supports WriteAt and ReadAt](https://github.com/minio/sio/issues/13). With that in place, the upload/download code would need to be rewritten to replace s3manager with manual control of part download/upload in order to ensue all operations are aligned to the underlying cipher block size.

## Built With
* [sio](https://github.com/minio/sio) - Go implementation of the Data At Rest Encryption (DARE) format
* [memguard](https://github.com/awnumar/memguard) -  Secure software enclave for storage of sensitive information in memory
* [cobra](https://github.com/spf13/cobra) - A Commander for modern Go CLI interactions
* [aws-sdk-go](https://aws.amazon.com/sdk-for-go/) - The official AWS SDK for the Go programming language

## Contributing
Please refer to [CONTRIBUTING.md](CONTRIBUTING.md)

## License
This software is released under [the Unlicense](https://unlicense.org).