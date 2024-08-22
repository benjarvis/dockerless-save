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
