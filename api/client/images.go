package client

import (
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/docker/distribution/reference"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	Cli "github.com/docker/docker/cli"
	"github.com/docker/docker/opts"
	flag "github.com/docker/docker/pkg/mflag"
	"github.com/docker/docker/pkg/stringid"
	"github.com/docker/docker/pkg/units"
)

// CmdImages lists the images in a specified repository, or all top-level images if no repository is specified.
//
// Usage: docker images [OPTIONS] [REPOSITORY]
func (cli *DockerCli) CmdImages(args ...string) error {
	cmd := Cli.Subcmd("images", []string{"[REPOSITORY[:TAG]]"}, Cli.DockerCommands["images"].Description, true)
	quiet := cmd.Bool([]string{"q", "-quiet"}, false, "Only show numeric IDs")
	all := cmd.Bool([]string{"a", "-all"}, false, "Show all images (default hides intermediate images)")
	noTrunc := cmd.Bool([]string{"-no-trunc"}, false, "Don't truncate output")
	showDigests := cmd.Bool([]string{"-digests"}, false, "Show digests")

	flFilter := opts.NewListOpts(nil)
	cmd.Var(&flFilter, []string{"f", "-filter"}, "Filter output based on conditions provided")
	cmd.Require(flag.Max, 1)

	cmd.ParseFlags(args, true)

	// Consolidate all filter flags, and sanity check them early.
	// They'll get process in the daemon/server.
	imageFilterArgs := filters.NewArgs()
	for _, f := range flFilter.GetAll() {
		var err error
		imageFilterArgs, err = filters.ParseFlag(f, imageFilterArgs)
		if err != nil {
			return err
		}
	}

	var matchName string
	if cmd.NArg() == 1 {
		matchName = cmd.Arg(0)
	}

	options := types.ImageListOptions{
		MatchName: matchName,
		All:       *all,
		Filters:   imageFilterArgs,
	}

	images, err := cli.client.ImageList(options)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cli.out, 20, 1, 3, ' ', 0)
	if !*quiet {
		if *showDigests {
			fmt.Fprintln(w, "REPOSITORY\tTAG\tDIGEST\tIMAGE ID\tCREATED\tSIZE")
		} else {
			fmt.Fprintln(w, "REPOSITORY\tTAG\tIMAGE ID\tCREATED\tSIZE")
		}
	}

	for _, image := range images {
		ID := image.ID
		if !*noTrunc {
			ID = stringid.TruncateID(ID)
		}

		repoTags := image.RepoTags
		repoDigests := image.RepoDigests

		if len(repoTags) == 1 && repoTags[0] == "<none>:<none>" && len(repoDigests) == 1 && repoDigests[0] == "<none>@<none>" {
			// dangling image - clear out either repoTags or repoDigsts so we only show it once below
			repoDigests = []string{}
		}

		// combine the tags and digests lists
		tagsAndDigests := append(repoTags, repoDigests...)
		for _, repoAndRef := range tagsAndDigests {
			// default repo, tag, and digest to none - if there's a value, it'll be set below
			repo := "<none>"
			tag := "<none>"
			digest := "<none>"

			if !strings.HasPrefix(repoAndRef, "<none>") {
				ref, err := reference.ParseNamed(repoAndRef)
				if err != nil {
					return err
				}
				repo = ref.Name()

				switch x := ref.(type) {
				case reference.Digested:
					digest = x.Digest().String()
				case reference.Tagged:
					tag = x.Tag()
				}
			}

			if !*quiet {
				if *showDigests {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s ago\t%s\n", repo, tag, digest, ID, units.HumanDuration(time.Now().UTC().Sub(time.Unix(int64(image.Created), 0))), units.HumanSize(float64(image.Size)))
				} else {
					fmt.Fprintf(w, "%s\t%s\t%s\t%s ago\t%s\n", repo, tag, ID, units.HumanDuration(time.Now().UTC().Sub(time.Unix(int64(image.Created), 0))), units.HumanSize(float64(image.Size)))
				}
			} else {
				fmt.Fprintln(w, ID)
			}
		}
	}

	if !*quiet {
		w.Flush()
	}
	return nil
}
