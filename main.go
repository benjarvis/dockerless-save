package main

import (
    "encoding/json"
    "archive/tar"
    "io/ioutil"
    "net/http"
    "bufio"
    "bytes"
    "fmt"
    "log"
    "io"
    "os"
    "strings"
    "time"
)

type Manifest struct {
    SchemaVersion int `json:"schemaVersion"`
    MediaType     string `json:"mediaType"`
    Config        struct {
        MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
    } `json:"config"`
    Layers []struct {
        MediaType string `json:"mediaType"`
        Size      int    `json:"size"`
        Digest    string `json:"digest"`
    } `json:"layers"`
}

type ManifestList struct {
    SchemaVersion int `json:"schemaVersion"`
    MediaType     string `json:"mediaType"`
    Manifests     []struct {
        MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
	Platform  struct {
	    Architecture string `json:"architecture"`
	    OS           string `json:"os"`
	} `json:"platform"`
    } `json:"manifests"`
}

type LayerSource struct {
	MediaType string `json:"mediaType"`
	Size      int64  `json:"size"`
	Digest    string `json:"digest"`
}

type ManifestEntry struct {
	Config       string                 `json:"Config"`
	RepoTags     []string               `json:"RepoTags"`
	Layers       []string               `json:"Layers"`
	LayerSources map[string]LayerSource `json:"LayerSources"`
}

type ImageManifest []ManifestEntry

func addBlob(client *http.Client, registry string, repository string, digest string, tarWriter *tar.Writer) error {

    hash_parts := strings.SplitN(digest, ":", 2)

    hashType := hash_parts[0]
    hash     := hash_parts[1]

    url := fmt.Sprintf("https://%s/v2/%s/blobs/%s", registry, repository, digest)

    filename := fmt.Sprintf("blobs/%s/%s", hashType, hash)

    resp, err := client.Get(url)

    if err != nil {
        return fmt.Errorf("error fetching file: %v", err)
    }

    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("received non-OK HTTP status: %s", resp.Status)
    }

    header := &tar.Header{
        Name: filename,
	Size: resp.ContentLength,
    }

    err = tarWriter.WriteHeader(header)
    if err != nil {
        return fmt.Errorf("error writing tar header: %v", err)
    }

    _, err = io.Copy(tarWriter, resp.Body)
    if err != nil {
        return fmt.Errorf("error copying file to tarball: %v", err)
    }

    return nil
}

func fetchManifest(client *http.Client, registry, repository, tag string) (string, []byte, error) {

    url := fmt.Sprintf("https://%s/v2/%s/manifests/%s", registry, repository, tag)

    resp, err := client.Get(url)

    if err != nil {
        return "", nil, err
    }

    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return "", nil, fmt.Errorf("failed to fetch manifest: status %s", resp.Status)
    }

    body, err := ioutil.ReadAll(resp.Body)

    if err != nil {
        return "", nil, err
    }

    mediaType := resp.Header.Get("Content-Type")

    return mediaType, body, nil
}

func addManifest(
    client *http.Client,
    registry string,
    repository string,
    tag string,
    manifest *Manifest,
    body []byte,
    tarWriter *tar.Writer,
    imageManifest *ImageManifest) error {

    log.Printf("Adding %s/%s\n", repository, tag)

    hash_parts := strings.SplitN(manifest.Config.Digest, ":", 2)

    hashType := hash_parts[0]
    hash     := hash_parts[1]

    filename := fmt.Sprintf("blobs/%s/%s", hashType, hash)

    for i, layer := range manifest.Layers {
        log.Printf(   "Layer %d: %s (size: %d bytes)\n", i+1, layer.Digest, layer.Size)
        err := addBlob(client, registry, repository, layer.Digest, tarWriter)

        if err != nil {
            return err
        }
    }

    err := addBlob(client, registry, repository, manifest.Config.Digest, tarWriter)

    if err != nil {
        return err
    }

    repoTag := fmt.Sprintf("%s:%s", repository, tag)

    entry := ManifestEntry{
        Config:       filename,
	RepoTags:     []string{repoTag},
	Layers:       make([]string, len(manifest.Layers)),
	LayerSources: make(map[string]LayerSource),
    }

    for i, layer := range manifest.Layers {

        hash_parts := strings.SplitN(layer.Digest, ":", 2)

        hashType := hash_parts[0]
        hash     := hash_parts[1]

        filename := fmt.Sprintf("blobs/%s/%s", hashType, hash)
	entry.Layers[i] = filename
	entry.LayerSources[layer.Digest] = LayerSource{
	    MediaType: layer.MediaType,
	    Size:      int64(layer.Size),
	    Digest:    layer.Digest,
	    }
    }

    *imageManifest = append(*imageManifest, entry)

    return nil
}

func main() {

    if len(os.Args) < 4 {
        log.Fatalf("Usage: %s <registry> <image list> <output tarball>", os.Args[0])
    }

    registry       := os.Args[1]
    image_list     := os.Args[2]
    output_tarball := os.Args[3]

    image_list_file, err := os.Open(image_list)
    if err != nil {
	log.Fatalf("Failed to open image list file: %s", err)
    }

    defer image_list_file.Close()

    tarball, err := os.Create(output_tarball)

    if err != nil {
        log.Fatalf("Error creating tarball file: %v\n", err)
    }
    defer tarball.Close()

    tarWriter := tar.NewWriter(tarball)

    defer tarWriter.Close()

    client := &http.Client{
        Transport: &http.Transport{
	    MaxIdleConns:        8,
	    MaxIdleConnsPerHost: 8,
	    IdleConnTimeout:     30 * time.Second,
	},
    }

    var imageManifest ImageManifest

    scanner := bufio.NewScanner(image_list_file)

    for scanner.Scan() {
        image_path := scanner.Text()

        log.Printf("Scanning %s\n", image_path)

        parts := strings.SplitN(image_path, ":", 2)

	repository := parts[0]
	tag := parts[1]

        mediaType, body, err := fetchManifest(client, registry,repository,tag)

        if err != nil {
	    log.Fatalf("Error fetching manifest: %v\n", err)
	}

        if strings.Contains(mediaType, "application/vnd.docker.distribution.manifest.v2+json") ||
           strings.Contains(mediaType, "application/vnd.oci.image.manifest.v1+json") {

            var manifest Manifest

            if err := json.Unmarshal(body, &manifest); err != nil {
                log.Fatalf("Error parsing manifest: %v\n", err)
            }
    
            addManifest(client, registry, repository, tag, &manifest, body, tarWriter, &imageManifest)

        } else if strings.Contains(mediaType, "application/vnd.docker.distribution.manifest.list.v2+json") {

            var manifestList ManifestList

            if err := json.Unmarshal(body, &manifestList); err != nil {
                log.Fatalf("Error parsing manifest list: %v\n", err)
            }

            for _, m := range manifestList.Manifests {
                _, arch_manifest, err := fetchManifest(client, registry, repository, m.Digest)

                if err != nil {
                    log.Fatalf("Error fetching manifest: %v\n", err)
                }

                var manifest Manifest

                if err := json.Unmarshal(arch_manifest, &manifest); err != nil {
                    log.Fatalf("Error parsing manifest: %v\n", err)
                }

                if strings.Contains(m.Platform.Architecture,"amd64") {
                    addManifest(client, registry, repository, tag, &manifest, body, tarWriter, &imageManifest)
                }
            }

        } else {
            log.Fatalf("Unhandled media type %s\n", mediaType) 
        }
    }

    if err := scanner.Err(); err != nil {
        log.Fatalf("Error scanning image list: %s", err)
    }

    json, err := json.MarshalIndent(imageManifest, "", "  ")

    if err != nil {
        log.Fatalf("Error serializing manifest to JSON: %v", err)
    }

    header := &tar.Header{
        Name: "manifest.json",
        Size: int64(len(json)),
    }

    err = tarWriter.WriteHeader(header)

    if err != nil {
        log.Fatalf("error writing tar header: %v", err)
    }

    _, err = io.Copy(tarWriter,  bytes.NewReader(json))

    if err != nil {
        log.Fatalf("error copying file to tarball: %v", err)
    }
}
