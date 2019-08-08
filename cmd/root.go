// Copyright © 2018 ERNO AAPA <ERNO.AAPA@GMAIL.COM>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/ernoaapa/kubectl-warp/pkg/cert"
	"github.com/ernoaapa/kubectl-warp/pkg/kubectl"
	"github.com/ernoaapa/kubectl-warp/pkg/sync"
	"github.com/ernoaapa/kubectl-warp/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apiv1 "k8s.io/api/core/v1"
)

type runOptions struct {
	Image     string
	Stdin     bool
	TTY       bool
	RsyncArgs string
	Includes  []string
	Excludes  []string
	ServiceAccountName string
	NodeSelectorKeys []string
	NodeSelectorValues []string
}

var configFlags = genericclioptions.NewConfigFlags()
var opt = runOptions{}
var workDir = "/work-dir"
var devNull = utils.DevNull(0)

var rootCmd = &cobra.Command{
	Use:   "warp",
	Short: "Transfer local files and run command in container",
	Long: `Start Pod and syncs local files to Pod and executes command
along with the synchronized files.`,
	RunE: func(_ *cobra.Command, args []string) error {
		stopChannel := make(chan struct{}, 1)
		// ctrl+c signal
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, os.Interrupt)
		defer signal.Stop(signals)

		go func() {
			<-signals
			close(stopChannel)
		}()
		if len(args) < 1 {
			return errors.New("NAME is required for warp")
		}
		var (
			name          = args[0]
			cmd           = args[1:]
			stdin         = os.Stdin
			stdout        = os.Stdout
			stderr        = os.Stderr
			containerName = "exec" // TODO
		)

		privateKey, publicKey, err := cert.Create()
		if err != nil {
			return err
		}

		privateKeyFile, err := utils.CreateTempFile(privateKey)
		if err != nil {
			return err
		}
		defer os.Remove(privateKeyFile)

		if !opt.Stdin {
			stdin = nil
		}

		ns, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
		if err != nil {
			return err
		}

		config, err := configFlags.ToRESTConfig()
		if err != nil {
			return err
		}
		kubectl.SetKubernetesDefaults(config)

		c := kubectl.NewClient(config)

		nodeSelectors := make(map[string]string)
		if len(opt.NodeSelectorKeys) > 0 {
			if len(opt.NodeSelectorKeys) != len(opt.NodeSelectorValues) {
				fmt.Fprintln(stderr, "Node selector keys and values don't match, dropping node selector. %d != %d", len(opt.NodeSelectorKeys), len(opt.NodeSelectorValues))
			} else {
				for index, value := range opt.NodeSelectorKeys {
					nodeSelectors[value] = opt.NodeSelectorValues[index]
				}
			}
		}

		fmt.Fprintln(stderr, "Create the Pod")
		_, err = c.CreatePod(ns, name, opt.Image, cmd, workDir, opt.TTY, opt.Stdin, publicKey, opt.ServiceAccountName, nodeSelectors)
		if err != nil {
			return err
		}
		defer c.DeletePod(ns, name)

		_, err = c.WaitForPod(ns, name, kubectl.PodInitReady)
		if err != nil && err != kubectl.ErrPodCompleted {
			return err
		}

		// Because init container doesn't support readinessProbe, we must wait a small moment so sshd is listening the port
		// otherwise sometimes we get error "Connection refused" from the port 22
		time.Sleep(100 * time.Millisecond)

		// Until this bug is fixed, we cannot use 0 to make the PortForwarder to pick random port
		// https://github.com/kubernetes/kubernetes/pull/71575
		randomPort := utils.MustResolveRandomPort()
		readyChannel := make(chan struct{}, 1)

		fmt.Fprintln(stderr, "Open connection to the Pod")
		f, err := kubectl.PreparePortForward(config, ns, name, []string{fmt.Sprintf("%d:%d", randomPort, 22)}, stopChannel, readyChannel, devNull, stderr)
		if err != nil {
			return err
		}
		go f.ForwardPorts()

		// Wait until port forwarding is ready
		<-readyChannel

		fmt.Fprintln(stderr, "Sync initial files to the Pod")
		s := sync.NewRsync(randomPort, strings.Split(opt.RsyncArgs, " "), privateKeyFile, devNull, devNull)
		if err := s.Sync(fmt.Sprintf("root@localhost:%s", workDir), opt.Includes, opt.Excludes); err != nil {
			return err
		}

		pod, err := c.WaitForPod(ns, name, kubectl.ContainerRunning(containerName))
		if err != nil {
			if err == kubectl.ErrPodCompleted {
				fmt.Fprintf(stderr, "Pod %s execution container were already completed. Print logs out\n", name)
				return logOutput(c, ns, name, containerName, stdout)
			}
			return err
		}
		if pod.Status.Phase == apiv1.PodSucceeded || pod.Status.Phase == apiv1.PodFailed {
			fmt.Fprintf(stderr, "Pod %s were already completed. Print logs to stdout\n", name)
			return logOutput(c, ns, name, containerName, stdout)
		}

		go func() {
			if _, err := c.WaitForPod(ns, name, kubectl.ContainerRunning("sync")); err != nil {
				fmt.Fprintf(stderr, "Error while waiting sync container to be started: %s\n", err)
				return
			}

			fmt.Fprintln(stderr, "Start background file sync")
			for {
				select {
				case <-time.After(1 * time.Second):
					if err := s.Sync(fmt.Sprintf("root@localhost:%s", workDir), opt.Includes, opt.Excludes); err != nil {
						fmt.Fprintf(stderr, "sync Failed: %s\n", err)
					}
				case <-stopChannel:
					fmt.Fprintf(stderr, "sync: Stop %s syncing\n", name)
					return
				}
			}
		}()

		return c.Attach(ns, name, containerName, stdin, stdout, stderr, opt.TTY)
	},
	// We handle errors at root.go
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	configFlags.AddFlags(rootCmd.Flags())

	rootCmd.Flags().StringVar(&opt.Image, "image", opt.Image, "The image for the container to run.")
	rootCmd.MarkFlagRequired("image")
	rootCmd.Flags().StringVar(&opt.RsyncArgs, "rsync-args", "--recursive --times --links --devices --specials", "Space separated arguments for the rsync command")
	rootCmd.Flags().BoolVarP(&opt.Stdin, "stdin", "i", opt.Stdin, "Pass stdin to the container")
	rootCmd.Flags().BoolVarP(&opt.TTY, "tty", "t", opt.TTY, "Stdin is a TTY")
	rootCmd.Flags().StringSliceVar(&opt.Includes, "include", []string{}, "Include only specific paths from current directory for syncing")
	rootCmd.Flags().StringSliceVar(&opt.Excludes, "exclude", []string{}, "Exclude only specific paths from current directory for syncing")
	rootCmd.Flags().StringVar(&opt.ServiceAccountName, "service-account-name", opt.ServiceAccountName, "The service account name that you want the pod to use")
	rootCmd.Flags().StringSliceVar(&opt.NodeSelectorKeys, "node-selector-keys", []string{}, "The node selector keys that you want applied")
	rootCmd.Flags().StringSliceVar(&opt.NodeSelectorValues, "node-selector-values", []string{}, "The node selector values that you want applied")
}

// Execute run the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if err.Error() == "interrupted" {
			fmt.Println("Cancelling...")
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
}
