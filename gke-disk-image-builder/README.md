# GKE Disk Image Builder
**[gke-disk-image-builder]([https://github.com/GoogleCloudPlatform/ai-on-gke/tools/gke-disk-image-builder](https://github.com/ai-on-gke/tools/tree/main/gke-disk-image-builder))**
is a Go module that pulls a given list of container images and creates a disk
image from the unpacked snapshots of the images. The generated disk image be
consumed by GKE at the time of node pool creation via secondary-boot-disk
feature. The goal of this module is preloading a disk image with the container
images to improve the startup latency of k8s pods on GKE nodes using the GKE
secondary-boot-disk feature.

## Usage
This is a Go module that can be used as a library as well as a CLI . Find the
examples below to understand how this tool work better.

### Preconditions

1.  For GCP authentication, either:
    1.  `~/.config/gcloud/application_default_credentials.json` must exit. If it
        does not exist, create it by running `gcloud auth application-default
        login`.
    1.  You must provide a `credentials.json` file to the tool via the
        `--gcp-oauth` flag. The path to `credentials.json` must be absolute.
1.  If a container image resides in a private registry, the tool runner must
    have access to it (See examples below).
1.  Compute Engine API must be enabled.
    (https://console.developers.google.com/apis/api/compute.googleapis.com/overview?project=$PROJECT_NAME)
1.  The *--gcs-path* flag is optional. If you omit it, the tool falls back to a
    `$PROJECT_NAME-daisy-bkt` bucket, creating it in the project on first use, so
    you don't have to provision a bucket yourself.
1.  If you *do* provide a *--gcs-path*, verify that
    `$PROJECT_NUMBER-compute@developer.gserviceaccount.com` has
    `storage.objectCreator` and `storage.objectViewer` permissions to that
    bucket for the logs and startup script.
    You can run the following command to grant proper permissions for this:

    ```shell
      gcloud storage buckets add-iam-policy-binding gs://$GCS_PATH \
      --project=$PROJECT_NAME \
      --member=serviceAccount:$PROJECT_NUMBER-compute@developer.gserviceaccount.com \
      --role=roles/storage.objectCreator

      gcloud storage buckets add-iam-policy-binding gs://$GCS_PATH \
      --project=$PROJECT_NAME \
      --member=serviceAccount:$PROJECT_NUMBER-compute@developer.gserviceaccount.com \
      --role=roles/storage.objectViewer
    ```

1.  If a disk image with the given name (via the **--image-name** flag) already
    exists, the tool will error out. Please provide a new name for the image.

### Available Flags

Flag                | Required | Default | Description
------------------- | -------- | ------- | -----------
*--project-name*    | Yes      | nil     | Name of a gcp project where the script will be run
*--image-name*      | Yes      | nil     | Name of the image that will be generated
*--image-family-name* | No | secondary-disk-image | Name of the image family associated with the created disk image
*--image-labels* | No | nil | Labels tagged to the disk image. This flag can be specified multiple times. The accepted format is `--image-labels=key=val`.
*--job-name*        | No       | secondary-disk-image | Name of the workflow. This name is used to provision some of the intermediate resources (disks, VMs) needed by the workflow. The maximum length is 50 characters
*--zone*            | Yes      | nil     | Zone where the resources will be used to create the image creator resources
*--gcs-path*        | No       | nil     | GCS path prefix to dump the logs and upload the startup script. If omitted, a `<project>-daisy-bkt` bucket is auto-created (or reused) in the project.
*--container-image* | Yes      | nil     | Container image with tag to include in the disk image. Tag are required for image like `latest` in this example: `docker.io/library/python:latest`. This flag can be specified multiple times
*--gcp-oauth*       | No       | nil     | Path to GCP service account credential file
*--disk-size-gb*    | No       | 10      | Size of a disk that will host the unpacked images
*--image-pull-auth* | No       | 'None'  | Auth mechanism to pull the container image, valid values: [None, ServiceAccountToken, RegistryCredentials]. None means that the images are publically available and no authentication is required to pull them. ServiceAccountToken means the service account oauth token will be used to pull the images. RegistryCredentials means the `user:password` string provided via *--registry-credentials* will be used to pull the images. For more information refer to https://cloud.google.com/compute/docs/access/authenticate-workloads#applications
*--registry-credentials* | No  | nil     | A `user:password` credential passed to `ctr image pull --user` when *--image-pull-auth=RegistryCredentials*. Use this for private registries (Docker Hub, Quay, Harbor, etc.) that do not accept a GCP service account OAuth token. **Note:** this value is embedded in the startup script uploaded to the GCS bucket (either *--gcs-path* or the auto-created `<project>-daisy-bkt`), so restrict access to that bucket.
*--timeout*         | No       | '20m'   | Default timeout for each step. Must be set to a proper value if the image is large to account for the pull and image creation time step.
*--network*         | No       | 'default'   | VPC network used by the GCE resources used for creating the disk image.
*--subnet*          | No       | 'default'   | Subnet used by the GCE resources used for creating the disk image.
*--service-account* | No       | 'default'   | Service Account email assigned to the GCE instance used for creating the disk image.
*--k8s-manifests-filepath* | No  | nil     | Specifies the file path where the tool will save the generated Kubernetes VolumeSnapshot and VolumeSnapshotContent manifests.
*--k8s-namespace*   | No       | 'default' | Specifies the Kubernetes namespace for the generated VolumeSnapshot manifest.


### Go

Run the tool by simply running it as a Go file:

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --disk-size-gb=$DISK_SIZE_GB \
    --container-image=$IMAGE_NAME
```

### Cloud Build

Cloud Build API must be enabled.
(https://console.cloud.google.com/apis/library/cloudbuild.googleapis.com?project=$PROJECT_NAME)

The Cloud Build service account must have the `compute.admin` and
`roles/compute.serviceAgent` roles:

```shell
gcloud projects add-iam-policy-binding $PROJECT_NAME \
--member serviceAccount:$PROJECT_NUMBER@cloudbuild.gserviceaccount.com \
--role roles/compute.admin \
--role roles/compute.serviceAgent
```

Create a Cloud Build config file, named **cloudbuild.yaml**:

```yaml
steps:
- name: 'gcr.io/cloud-builders/git'
  args: ['clone', 'https://github.com/GoogleCloudPlatform/ai-on-gke.git']
- name: 'gcr.io/cloud-builders/go:1.21'
  env: ['GOPATH=./ai-on-gke/tools/gke-disk-image-builder']
  dir: './ai-on-gke/tools/gke-disk-image-builder'
  args:
    - 'run'
    - './cli'
    - --project-name={project-name}
    - --image-name={image-name}
    - --zone={zone}
    - --gcs-path=gs://{gcs-path}/
    - --disk-size-gb={disk-size-gb}
    - --container-image={image-name}
```

And then submit it:

```shell
gcloud builds submit --config cloudbuild.yaml --no-source
```

### CLI

You can compile the module and use the generated binary:

```shell
go build -o image-builder ./cli
./image-builder \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --disk-size-gb=$DISK_SIZE_GB \
    --container-image=$IMAGE_NAME
```

## Examples

Here are some examples on how to use the tool.

### Environment Variable

**Note:** Setting this environment variables are not necessary to use the tool.
This only simplify presenting the examples.

```shell
PROJECT_NAME=my-project-name
IMAGE_NAME=example-image
ZONE=us-west1-b
GCS_PATH=gke-disk-image-builder-logs
```

### Pull an image from a public registry

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --container-image=docker.io/library/nginx:latest
```

### Pull multiple images from a public registry

All the images will be pulled, unpacked and put on the same disk image. The
generated disk image is consumable by GKE the same as a single image one.

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --container-image=docker.io/library/python:latest \
    --container-image=docker.io/library/nginx:latest
```

### Pull a large image from a public registry

Use the **--disk-size-gb** flag to create a larger disk as follows. This value
will be the size of output disk image.

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --disk-size-gb=50 \
    --container-image=nvcr.io/nvidia/tritonserver:23.09-py3
```

### Pull an image from a private registry

The tool uses OAuth token of the GCE service account to authenticate to the
private registry. First, you must grant it read permission on the registry.

Use the **--image-pull-auth** flag as follows:

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --image-pull-auth=ServiceAccountToken \
    --container-image=$CONTAINER_IMAGE_FROM_PRIVATE_REGISTRY
```

### Pull an image from a private registry with a username/password

For private registries that do not accept a GCP service account OAuth token
(e.g. Docker Hub, Quay, a self-hosted Harbor), provide a static `user:password`
credential. It is passed verbatim to `ctr image pull --user`.

**Note:** the credential is embedded in the startup script that the tool uploads
to the GCS bucket (either **--gcs-path** or, if omitted, the auto-created
`<project>-daisy-bkt`). Restrict access to that bucket accordingly, and prefer a
scoped, read-only token over a long-lived password.

`--gcs-path` is omitted below to show that no bucket needs to be configured:

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --image-pull-auth=RegistryCredentials \
    --registry-credentials="$REGISTRY_USER:$REGISTRY_PASSWORD" \
    --container-image=$CONTAINER_IMAGE_FROM_PRIVATE_REGISTRY
```

### Run the tool with a GCP OAuth token

By default the tool uses the OAuth credentials stored in
`~/.config/gcloud/application_default_credentials.json`.

**Note:** The path of the `--gcp-oauth` must be absolute.

**Note:** Run `gcloud auth application-default login` to create
`~/.config/gcloud/application_default_credentials.json` file.

```shell
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --gcp-oauth=/usr/local/google/home/username/another_credentials.json \
    --container-image=docker.io/library/nginx:latest
```
### Customize the image family name and attach labels to the built image
```
go run ./cli \
    --project-name=$PROJECT_NAME \
    --image-name=$IMAGE_NAME \
    --zone=$ZONE \
    --gcs-path=gs://$GCS_PATH/ \
    --container-image=docker.io/library/python:latest \
    --container-image=docker.io/library/nginx:latest \
    --image-family-name=customize-image-family \
    --image-labels=pokemon=pikachu \
    --image-labels=band=ac-dc \
    --image-labels=theory=ramsey_theory
```

### Generate Kubernetes manifests derived from the disk image

Use the `--k8s-manifests-filepath` flag to specify a file path where the tool will save the generated Kubernetes `VolumeSnapshot` and `VolumeSnapshotContent` manifests. You can also specify a custom namespace with the `--k8s-namespace` flag:

```shell
go run ./cli \
    --project-name=my-gke-project \
    --image-name=my-secondary-disk-image \
    --zone=us-central1-c \
    --gcs-path=gs://my-gke-project-logs \
    --disk-size-gb=20 \
    --container-image=docker.io/library/python:3.11.7-slim-bookworm \
    --container-image=docker.io/library/nginx:1.25.3 \
    --timeout=30m \
    --k8s-manifests-filepath=./image_handles.yaml \
    --k8s-namespace=my-namespace
```

