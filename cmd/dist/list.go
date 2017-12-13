package main

import (
	"fmt"
	"io"
	"os"

	"github.com/docker/distribution/reference"

	ctxu "github.com/docker/distribution/context"
	"github.com/docker/distribution/registry/storage"
	"github.com/docker/distribution/registry/storage/driver/filesystem"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

var (
	commandList = cli.Command{
		Name:   "images",
		Usage:  "List available images",
		Action: imageList,
	}
)

func imageList(c *cli.Context) {
	ctx := context.Background()

	driver := filesystem.New(filesystem.DriverParameters{
		RootDirectory: "/var/lib/registry",
		MaxThreads:    1,
	})

	local, err := storage.NewRegistry(ctx, driver)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"REPOSITORY", "TAG", "DIGEST"})

	// limit ourselves to showing 100 images
	repos := make([]string, 100)

	n, err := local.Repositories(ctx, repos, "")
	if err != nil {
		if err != io.EOF {
			ctxu.GetLogger(ctx).Fatalf("error getting repositories: %v", err)
		}
	}

	for i := 0; i < n; i++ {
		namedRepo, err := reference.WithName(repos[i])
		if err != nil {
			ctxu.GetLogger(ctx).Fatalf("error getting a named reference: %v", err)
		}
		repo, err := local.Repository(ctx, namedRepo)
		if err != nil {
			ctxu.GetLogger(ctx).Fatalf("error getting repository %s: %v", namedRepo.Name(), err)
		}

		tagService := repo.Tags(ctx)

		tags, err := tagService.All(ctx)
		if err != nil {
			ctxu.GetLogger(ctx).Fatalf("error reading tags: %v", err)
		}

		for _, tag := range tags {
			descriptor, err := tagService.Get(ctx, tag)
			if err != nil {
				ctxu.GetLogger(ctx).Errorf("error retrieving tag descriptor for %s: %v", tag, err)
				continue
			}
			table.Append([]string{namedRepo.Name(), tag, descriptor.Digest.String()})
		}
	}

	table.Render()
}
