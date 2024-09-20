![Build Status]

# Velero Plugins for Scaleway

## Overview

This repository contains plugins to support running Velero on Scaleway:

- **Object Store Plugin**: Persists and retrieves backups on Scaleway Object Storage. The content of backups includes Kubernetes resources, metadata for CSI objects, and progress of asynchronous operations.
- It also stores result data from backups and restores, including log files, and warning/error files.

- **Volume Snapshotter Plugin**: Creates snapshots from volumes during a backup and restores volumes from snapshots during a restore using Scaleway Block Storage.
    - The snapshotter plugin supports volumes provisioned by the CSI driver `sbs-default.csi.scaleway.com`.

## Environment

You can configure your config with environment variables.

# Documentation for `scw config`
Config management engine is common across all Scaleway developer tools (CLI, terraform, SDK, ... ). It allows to handle Scaleway config through two ways: environment variables and/or config file.
Default path for configuration file is based on the following priority order:

- $SCW_CONFIG_PATH
- $XDG_CONFIG_HOME/scw/config.yaml
- $HOME/.config/scw/config.yaml
- $USERPROFILE/.config/scw/config.yaml

In this plugin, environment variables have priority over the configuration file.

The following environment variables are supported:

|Environment Variable| Description                                                                                         |
|--|-----------------------------------------------------------------------------------------------------|
|SCW_ACCESS_KEY| The access key of a token (create a token at https://console.scaleway.com/iam/api-keys)             |
|SCW_SECRET_KEY| The secret key of a token (create a token at https://console.scaleway.com/iam/api-keys)             |
|SCW_S3_ENDPOINT| URL of a custom S3 endpoint API                                                                     |
|SCW_DEFAULT_ORGANIZATION_ID| The default organization ID (get your organization ID at https://console.scaleway.com/iam/api-keys) |
|SCW_DEFAULT_PROJECT_ID| The default project ID (get your project ID at https://console.scaleway.com/iam/api-keys)           |
|SCW_DEFAULT_REGION| The default region                                                                                  |
|SCW_DEFAULT_ZONE| The default availability zone                                                                       |
|SCW_API_URL| URL of the API                                                                                      |
|SCW_PROFILE| Set the config profile to use                                                                       |

Read more about the config management engine at https://github.com/scaleway/scaleway-sdk-go/tree/master/scw#scaleway-config

## Compatibility

Below is a listing of plugin versions and respective Velero versions that are compatible:

| Plugin Version | Velero Version |
|----------------|----------------|
| v1.0.x         | v1.14.x        |

## Filing Issues

If you would like to file a GitHub issue for the plugin, please open the issue on the [core Velero repo][103].

## Setup

To set up Velero on Scaleway, follow these steps:

* [Create a Scaleway Object Storage Bucket][1]
* [Set permissions for Velero][2]
* [Install and start Velero][3]

You can also use this plugin to [migrate PVs across clusters][5] or create an additional [Backup Storage Location][12].

If you do not have the `scw` CLI locally installed, follow the [Scaleway CLI guide][6] to set it up.

You must use the env variables from the SDK [scaleway](https://github.com/scaleway/scaleway-sdk-go)

## Create Scaleway Object Storage Bucket

Velero requires an object storage bucket to store backups in, preferably unique to a single Kubernetes cluster. Create an Object Storage bucket, replacing placeholders appropriately:

## Create S3 bucket

Velero requires an object storage bucket to store backups in, preferably unique to a single Kubernetes cluster (see the [FAQ][11] for more details). Create an S3 bucket, replacing placeholders appropriately:

```bash
BUCKET=<YOUR_BUCKET>
REGION=<YOUR_REGION>
aws s3api create-bucket \
    --bucket $BUCKET \
    --region $REGION \
    --create-bucket-configuration LocationConstraint=$REGION
```

[1]: #Create-S3-bucket
[2]: #Set-permissions-for-Velero
[3]: #Install-and-start-Velero
[4]: https://velero.io/docs/install-overview/
[5]: #Migrating-PVs-across-clusters
[6]: https://www.scaleway.com/en/cli/
[7]: backupstoragelocation.md
[8]: volumesnapshotlocation.md
[9]: https://velero.io/docs/customize-installation/
[10]: https://www.scaleway.com/en/docs/identity-and-access-management/iam/quickstart/
[11]: https://velero.io/docs/faq/
[12]: #Create-an-additional-Backup-Storage-Location
[13]: https://velero.io/docs/latest/api-types/backupstoragelocation/
[14]: #option-2-set-permissions-using-kube2iam
[15]: #create-s3-bucket
[16]: #option-1-set-permissions-with-an-iam-user
[17]: https://kubernetes.io/docs/concepts/configuration/secret/
[103]: https://github.com/vmware-tanzu/velero/issues/new/choose