package main

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/docker/distribution"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	digest "github.com/opencontainers/go-digest"

	ctxu "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver/filesystem"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

var (
	commandMount = cli.Command{
		Name:   "mount",
		Usage:  "Mount the image at path",
		Action: mount,
	}
)

func mount(c *cli.Context) {
	ctx := context.Background()
	image := c.Args().First()
	mountPath := c.Args().Get(1)

	if mountPath == "" {
		ctxu.GetLogger(ctx).Fatalln("must specify mount path")
	}

	fi, err := os.Stat(mountPath)
	if err != nil {
		ctxu.GetLogger(ctx).Fatalln(err)
	}

	if !fi.IsDir() {
		ctxu.GetLogger(ctx).Fatalln("mount path should be a directory")
	}

	storageDriver := filesystem.New(filesystem.DriverParameters{
		RootDirectory: "/var/lib/registry",
		MaxThreads:    1,
	})

	local, err := storage.NewRegistry(ctx, storageDriver)
	if err != nil {
		ctxu.GetLogger(ctx).Fatalf("could not create a local registry: %v\n", err)
	}

	namedRepo, err := reference.WithName(image)
	if err != nil {
		ctxu.GetLogger(ctx).Fatalf("could not get a name: %v\n", err)
	}

	repo, err := local.Repository(ctx, namedRepo)
	if err != nil {
		ctxu.GetLogger(ctx).Fatalln(err)
	}

	manifests, err := repo.Manifests(ctx)
	if err != nil {
		ctxu.GetLogger(ctx).Fatalf("could not get manifests: %v\n", err)
	}

	manifestEnumerator, ok := manifests.(distribution.ManifestEnumerator)
	if !ok {
		ctxu.GetLogger(ctx).Fatalln("unable to convert ManifestService into ManifestEnumerator")
	}

	err = manifestEnumerator.Enumerate(ctx, func(dgst digest.Digest) error {

		manifest, err := manifests.Get(ctx, dgst)
		if err != nil {
			return fmt.Errorf("failed to retrieve manifest for digest %v: %v", dgst, err)
		}

		descriptors := manifest.References()
		for _, descriptor := range descriptors {
			// skip media types which are not part of the layer
			if descriptor.MediaType != schema2.MediaTypeLayer {
				continue
			}
			blob, err := repo.Blobs(ctx).Open(ctx, descriptor.Digest)
			if err != nil {
				return fmt.Errorf("could not open blob with digest %s: %v", dgst, err)
			}
			defer blob.Close()

			if err := extractTarFile(ctx, mountPath, blob); err != nil {
				return fmt.Errorf("error extracting tar: %v", err)
			}
		}
		return nil
	})

	if err != nil {
		ctxu.GetLogger(ctx).Fatalln(err)
	}
	fmt.Println("done")
}

func extractTarFile(ctx context.Context, path string, rd io.Reader) error {
	cmd := exec.Command("tar", "-x", "-z", "-C", path) // may need some extra options for users/permissions
	cmd.Stdin = rd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeTarFile(ctx context.Context, path string, hdr *tar.Header, rd io.Reader) error {
	target := filepath.Join(path, hdr.Name)

	switch hdr.Typeflag {
	case tar.TypeReg, tar.TypeRegA:
		fp, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer fp.Close()

		if _, err := io.Copy(fp, rd); err != nil {
			return err
		}

		fp.Chmod(hdr.FileInfo().Mode())
		// fp.Chmod(hdr.Uid, hdr.Gid)
		os.Chtimes(target, hdr.AccessTime, hdr.ModTime)
	case tar.TypeDir:
		if err := os.MkdirAll(target, hdr.FileInfo().Mode()); err != nil {
			return err
		}
	default:
		ctxu.GetLogger(ctx).Infof("skip %q %v -> %v", hdr.Typeflag, hdr.Name, hdr.Linkname)
		return fmt.Errorf("unsupported file: %v", hdr)
	}

	return nil
}
