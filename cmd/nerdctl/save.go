/*
   Copyright The containerd Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package main

import (
	"io"
	"os"

	"github.com/containerd/containerd/images/archive"
	"github.com/containerd/containerd/platforms"
	refdocker "github.com/containerd/containerd/reference/docker"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newSaveCommand() *cobra.Command {
	var saveCommand = &cobra.Command{
		Use:               "save",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Save one or more images to a tar archive (streamed to STDOUT by default)",
		Long:              "The archive implements both Docker Image Spec v1.2 and OCI Image Spec v1.0.",
		RunE:              saveAction,
		ValidArgsFunction: saveShellComplete,
		SilenceUsage:      true,
		SilenceErrors:     true,
	}
	saveCommand.Flags().StringP("output", "o", "", "Write to a file, instead of STDOUT")
	return saveCommand
}

func saveAction(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return errors.Errorf("requires at least 1 argument")
	}

	var (
		images   = args
		saveOpts = []archive.ExportOpt{}
	)

	if len(images) == 0 {
		return errors.Errorf("requires at least 1 argument")
	}

	out := cmd.OutOrStdout()
	output, err := cmd.Flags().GetString("output")
	if err != nil {
		return err
	}
	if output != "" {
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	} else {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			return errors.Errorf("cowardly refusing to save to a terminal. Use the -o flag or redirect")
		}
	}
	return saveImage(images, out, saveOpts, cmd)
}

func saveImage(images []string, out io.Writer, saveOpts []archive.ExportOpt, cmd *cobra.Command) error {
	client, ctx, cancel, err := newClient(cmd)
	if err != nil {
		return err
	}
	defer cancel()

	// Set the default platform
	saveOpts = append(saveOpts, archive.WithPlatform(platforms.DefaultStrict()))

	imageStore := client.ImageService()
	for _, img := range images {
		named, err := refdocker.ParseDockerRef(img)
		if err != nil {
			return err
		}
		saveOpts = append(saveOpts, archive.WithImage(imageStore, named.String()))
	}

	return client.Export(ctx, out, saveOpts...)
}

func saveShellComplete(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// show image names
	return shellCompleteImageNames(cmd)
}