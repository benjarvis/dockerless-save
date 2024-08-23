# dockerless-save

It is possible to deploy k3s kubernetes in an airgapped environment by providing
one or more airgap image tarballs which will be preloaded when k3s starts so 
that when containers are subsequently started it is not necessary to pull the
images.

k3s expects these tarballs to be in the format created by the 'docker save',
command.  So one would normally do something like:

```
$ docker pull ubuntu:22.04
22.04: Pulling from library/ubuntu
857cc8cb19c0: Pull complete
Digest: sha256:adbb90115a21969d2fe6fa7f9af4253e16d45f8d4c1e930182610c4731962658
Status: Downloaded newer image for ubuntu:22.04
docker.io/library/ubuntu:22.04

$ docker pull nginx:latest
latest: Pulling from library/nginx
e4fff0779e6d: Pull complete
2a0cb278fd9f: Pull complete
7045d6c32ae2: Pull complete
03de31afb035: Pull complete
0f17be8dcff2: Pull complete
14b7e5e8f394: Pull complete
23fa5a7b99a6: Pull complete
Digest: sha256:447a8665cc1dab95b1ca778e162215839ccbb9189104c79d7ec3a81e14577add
Status: Downloaded newer image for nginx:latest
docker.io/library/nginx:latest

$ docker save -o images.tar ubuntu:22.04 nginx:latest
```

This is an unnecessarily heavy process.

First, we need to have docker running, which is itself objectionable, especially
if we are performing this operation inside a container in the first place as it
requires DIND or DINK.

Secondly, every image must be pulled first, if not already present, which means
the layers all need to be decompressed and extracted.   Then when we save them
they are all repacked and compressed again.

Finally, if we're doing this exercise on a large number of images, we have to
pass them all on the command line to the save command which has a limited 
maximum length.

This little go program does the same job, but just fetches the image manifests
and blobs natively in golang and streams them straight into a tarball.   No
unpacking and repacking of content is required, and no docker daemon is required.

I only implemented this to the minimum level of fidelity needed to satisfy my
own immediate needs, so chances are you'll find some problems trying to use it 
in a different environment.  It should at least provide a good example of how
to get close.

It would be nicer if a non-docker open source utility like skopeo could do this.
Unfortunately, I was not able to get that route to work.

Using this program we can instead do something like this:

```
$ cat airgap_images.txt
rancher/local-path-provisioner:v0.0.26
rancher/mirrored-coredns-coredns:1.10.1
rancher/mirrored-library-busybox:1.36.1
rancher/mirrored-library-traefik:2.10.7
rancher/mirrored-metrics-server:v0.7.0
rancher/mirrored-pause:3.6
rancher/klipper-helm:v0.8.3-build20240228
rancher/klipper-lb:v0.4.7
tigera/operator:v1.34.0
calico/cni:v3.28.0
calico/ctl:v3.28.0
calico/apiserver:v3.28.0
calico/kube-controllers:v3.28.0
calico/typha:v3.28.0
calico/pod2daemon-flexvol:v3.28.0
calico/node:v3.28.0
calico/node-driver-registrar:v3.28.0
calico/dikastes:v3.28.0
calico/csi:v3.28.0
squat/generic-device-plugin:latest

$ ./dockerless-save myrepository.company.com airgap_images.txt images.tar
2024/08/22 15:12:44 Adding rancher/local-path-provisioner/v0.0.26
2024/08/22 15:12:44 Layer 1: sha256:c926b61bad3b94ae7351bafd0c184c159ebf0643b085f7ef1d47ecdc7316833c (size: 3402422 bytes)
2024/08/22 15:12:44 Layer 2: sha256:d9dea7184c18b9fd1eda122162a298424d19837f382ad406689de67f8da449be (size: 1878924 bytes)
2024/08/22 15:12:44 Layer 3: sha256:85343b2d3ec51fce11468bc17ef81a6eb7d5d51869d1c0ff415dd4e978cb6de2 (size: 7958 bytes)
2024/08/22 15:12:44 Layer 4: sha256:0b46a5af7972c235f1aadf4ecd2b319d66ca653314d8a461db6e0542cc1f1c15 (size: 11886679 bytes)
2024/08/22 15:12:44 Adding rancher/mirrored-coredns-coredns/1.10.1
2024/08/22 15:12:44 Layer 1: sha256:25b7032c281a433b92d09930f3a03c0f7382c27eb69ae7f35addf2e3853dbba7 (size: 115220 bytes)
2024/08/22 15:12:44 Layer 2: sha256:3799eae1a077913c39123d0f4fe4e16243237e7ee94cd84874ea755ec930805a (size: 16071754 bytes)
2024/08/22 15:12:44 Adding rancher/mirrored-library-busybox/1.36.1
2024/08/22 15:12:44 Layer 1: sha256:ec562eabd705d25bfea8c8d79e4610775e375524af00552fe871d3338261563c (size: 2152663 bytes)
2024/08/22 15:12:44 Adding rancher/mirrored-library-traefik/2.10.7
2024/08/22 15:12:44 Layer 1: sha256:619be1103602d98e1963557998c954c892b3872986c27365e9f651f5bc27cab8 (size: 3402542 bytes)
2024/08/22 15:12:44 Layer 2: sha256:987f790ee1434532d04fff8e0ff09f57d177ab515cdad8c0c3797d839f186a98 (size: 622798 bytes)
2024/08/22 15:12:44 Layer 3: sha256:c6d80f829c660d4ca9f0cf5c29678d30cbb7922ae43f9b783d15a2fcbce9f786 (size: 39207537 bytes)
2024/08/22 15:12:44 Layer 4: sha256:5300d181735bfc4dbb1dcaefae56d132d04c53b7e951c6d23139c55b6ea5cb25 (size: 368 bytes)
2024/08/22 15:12:44 Adding rancher/mirrored-metrics-server/v0.7.0
2024/08/22 15:12:44 Layer 1: sha256:07a64a71e01156f8f99039bc246149925c6d1480d3957de78510bbec6ec68f7a (size: 103742 bytes)
2024/08/22 15:12:44 Layer 2: sha256:fe5ca62666f04366c8e7f605aa82997d71320183e99962fa76b3209fdfbb8b58 (size: 21202 bytes)
2024/08/22 15:12:44 Layer 3: sha256:280126c0e181aba326fc843e7f17918dc9d54ddbfd917f5a3e0b346cec57fb70 (size: 717051 bytes)
2024/08/22 15:12:44 Layer 4: sha256:fcb6f6d2c9986d9cd6a2ea3cc2936e5fc613e09f1af9042329011e43057f3265 (size: 317 bytes)
2024/08/22 15:12:44 Layer 5: sha256:e8c73c638ae9ec5ad70c49df7e484040d889cca6b4a9af056579c3d058ea93f0 (size: 198 bytes)
2024/08/22 15:12:44 Layer 6: sha256:1e3d9b7d145208fa8fa3ee1c9612d0adaac7255f1bbc9ddea7e461e0b317805c (size: 113 bytes)
2024/08/22 15:12:44 Layer 7: sha256:4aa0ea1413d37a58615488592a0b827ea4b2e48fa5a77cf707d0e35f025e613f (size: 385 bytes)
2024/08/22 15:12:44 Layer 8: sha256:7c881f9ab25e0d86562a123b5fb56aebf8aa0ddd7d48ef602faf8d1e7cf43d8c (size: 355 bytes)
2024/08/22 15:12:44 Layer 9: sha256:5627a970d25e752d971a501ec7e35d0d6fdcd4a3ce9e958715a686853024794a (size: 130562 bytes)
2024/08/22 15:12:44 Layer 10: sha256:12d3002873f59d403aa658d1ccd12b291a116ccbcd012afdfac7db04778f66a1 (size: 18455127 bytes)
2024/08/22 15:12:44 Adding rancher/mirrored-pause/3.6
2024/08/22 15:12:44 Layer 1: sha256:fbe1a72f5dcd08ba4ca3ce3468c742786c1f6578c1f6bb401be1c4620d6ff705 (size: 296517 bytes)
2024/08/22 15:12:44 Adding rancher/klipper-helm/v0.8.3-build20240228
2024/08/22 15:12:44 Layer 1: sha256:4abcf20661432fb2d719aaf90656f55c287f8ca915dc1c92ec14ff61e67fbaf8 (size: 3408729 bytes)
2024/08/22 15:12:44 Layer 2: sha256:c8f65b19222d1a249d3ade42072f2ef00ee6f3811ddac59d45b0ee0403671a09 (size: 1546624 bytes)
2024/08/22 15:12:44 Layer 3: sha256:8cf9443d78431facaf0b1e9275962c05c76b89e52c0114f045dd4273cec68be4 (size: 44493065 bytes)
2024/08/22 15:12:44 Layer 4: sha256:c4f37b85adb94d0d17dc3ba7b753643661db09a1e8d5eadc4cbf8b24688deefa (size: 41706901 bytes)
2024/08/22 15:12:44 Adding rancher/klipper-lb/v0.4.7
2024/08/22 15:12:44 Layer 1: sha256:4abcf20661432fb2d719aaf90656f55c287f8ca915dc1c92ec14ff61e67fbaf8 (size: 3408729 bytes)
2024/08/22 15:12:44 Layer 2: sha256:11b5df3687c6c87e365fc66824b198ee1c01eca7b8a97f55c6789792b2c37428 (size: 1362286 bytes)
2024/08/22 15:12:44 Layer 3: sha256:52da6df503a6858574525446b4978ff3c0176a346cd677ff0791167440176d74 (size: 882 bytes)
2024/08/22 15:12:45 Adding squat/generic-device-plugin/latest
2024/08/22 15:12:45 Layer 1: sha256:29d721ecba621fd548d4778672db15fb35e2bdbe9c1728cd9c20f7feb78517b2 (size: 10074188 bytes)

Then just to prove the output is in a format docker will accept:

$ docker load -i images.tar
Loaded image: rancher/local-path-provisioner:v0.0.26
Loaded image: rancher/mirrored-coredns-coredns:1.10.1
Loaded image: rancher/mirrored-library-busybox:1.36.1
Loaded image: rancher/mirrored-library-traefik:2.10.7
Loaded image: rancher/mirrored-metrics-server:v0.7.0
Loaded image: rancher/mirrored-pause:3.6
Loaded image: rancher/klipper-helm:v0.8.3-build20240228
Loaded image: rancher/klipper-lb:v0.4.7
Loaded image: squat/generic-device-plugin:latest
```

The way I am using this, all my images come from the same local repository, but
the program could be modified easily enough to allow the images to come from
different places in a different situation.
