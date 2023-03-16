/*
Copyright 2023 The Kubernetes Authors.

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

package cli

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	"github.com/kubernetes-sigs/community-images/pkg/community_images"
	"github.com/kubernetes-sigs/community-images/pkg/logger"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "community-images",
		Short:         "",
		Long:          `.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewLogger()
			log.Info("")

			//s := spin.New()
			finishedCh := make(chan bool, 1)
			foundImageName := make(chan string, 1)
			go func() {
				//lastImageName := ""
				for {
					select {
					case <-finishedCh:
						fmt.Printf("\r")
						return
					case <-foundImageName:
						//lastImageName = i
						//case <-time.After(time.Millisecond * 100):
						//	if lastImageName == "" {
						//		fmt.Printf("\r  \033[36mSearching for images\033[m %s", s.Next())
						//	} else {
						//		fmt.Printf("\r  \033[36mSearching for images\033[m %s (%s)", s.Next(), lastImageName)
						//	}
					}
				}
			}()
			defer func() {
				finishedCh <- true
			}()

			contexts := v.GetStringSlice("contexts")
			if len(contexts) <= 0 {
				cfg, _ := KubernetesConfigFlags.ToRawKubeConfigLoader().RawConfig()
				contexts = []string{cfg.CurrentContext}
			}

			ns := "kube-system"
			for i := range contexts {
				KubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
				KubernetesConfigFlags.Namespace = &ns
				KubernetesConfigFlags.ClusterName = &contexts[i]
				KubernetesConfigFlags.AuthInfoName = &contexts[i]

				config, _ := KubernetesConfigFlags.ToRESTConfig()
				log.Header(headerLine(contexts[i], config.Host))

				imagesList, err := community_images.ListImages(KubernetesConfigFlags, foundImageName, v.GetStringSlice("ignore-ns"))
				if err != nil {
					log.Error(err)
					log.Info("")
					os.Exit(1)
					return nil
				}
				//finishedCh <- true

				re := regexp.MustCompile(`^k8s\.gcr\.io/|^gcr\.io/google-containers`)
				for _, runningImage := range imagesList {
					image := imageWithTag(runningImage)
					if re.MatchString(image) {
						log.StartImageLine(image)
						log.ImageRedLine(image)
					} else {
						if !v.GetBool("only-red-line") {
							log.StartImageLine(image)
							log.ImageGreenLine(image)
						}
					}
				}

				log.Info("")
			}

			fmt.Printf("\nImages in \033[91mred ❌ \033[mare being pulled from \033[1m*outdated*\033[0m Kubernetes community registries.\n" +
				"The others marked in \033[92mgreen ✅ \u001B[mare good as they do not use the outdated registries.\n" +
				"Please copy these images to your own registry and change your manifest(s)\nto point to the new location.\n\n")
			fmt.Printf(
				"If you are unable to do so, as a short term fix please use \033[92m`registry.k8s.io`\033[m " +
					"\ninstead of \033[91m`k8s.gcr.io`\033[m until you have your own registry.\n\n")
			fmt.Printf("This simple change on your part will help the Kubernetes community immensely as it\n" +
				"reduces the cost of us serving these container images.\n")

			fmt.Printf("\n\033[1mWhy you should do this as soon as possible? Read more in the following blog\n" +
				"posts by the Kubernetes community:\033[m\n" +
				"- https://kubernetes.io/blog/2022/11/28/registry-k8s-io-faster-cheaper-ga/\n" +
				"- https://kubernetes.io/blog/2023/02/06/k8s-gcr-io-freeze-announcement/\n")

			log.Info("")
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	KubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	KubernetesConfigFlags.AddFlags(cmd.Flags())

	cmd.Flags().StringSlice("ignore-ns", []string{}, "optional list of namespaces to exclude from searching")
	cmd.Flags().Bool("only-red-line", false, "print only red lines")
	cmd.Flags().StringSlice("contexts", []string{}, "optional list of contexts to searching")

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("OUTDATED")
	viper.AutomaticEnv()
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
