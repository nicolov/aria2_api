package main

import (
	"github.com/spf13/cobra"
	"fmt"
	"os"
	"aria2_api"
	"log"
	"math"
	"encoding/json"
	"strings"
)

const endpointUrl = "http://127.0.0.1:6801/jsonrpc"

func logn(n, b float64) float64 {
	return math.Log(n) / math.Log(b)
}

func humanizeHelper(s uint64, base float64, sizes []string) string {
	if s == 0 {
		return " -"
	}
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(logn(float64(s), base))
	suffix := sizes[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.1f%s"

	return fmt.Sprintf(f, val, suffix)
}

func humanizeBytes(s uint64) string {
	sizes := []string{"B", "k", "M", "G", "T", "P", "E"}
	return humanizeHelper(s, 1000, sizes)
}

func main() {
	var rootCmd = &cobra.Command{
		Use: "aria2_cli",
	}

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List torrents",
		Run: func(cmd *cobra.Command, args [] string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			stats, err := client.TellActive()

			if err != nil {
				log.Fatal(err)
			}

			for _, dStatus := range stats {
				// Try to determine display name
				var displayName string
				displayName = "n/a"
				if dStatus.Bittorrent != nil && dStatus.Bittorrent.Info.Name != "" {
					displayName = dStatus.Bittorrent.Info.Name
				}

				// Percent completion
				pctComplete := "100.0%"
				if dStatus.TotalLength > 0 {
					pctComplete = fmt.Sprintf("%0.1f%%",
						100.0*float64(dStatus.CompletedLength)/float64(dStatus.TotalLength))
				}
				if pctComplete == "100.0%" {
					pctComplete = "done"
				}

				fmt.Printf("%s  %20s  %5s  %6s  %6s  %6s  %6s\n",
					dStatus.Gid[:6],
					displayName,
					pctComplete,
					humanizeBytes(dStatus.CompletedLength),
					humanizeBytes(dStatus.TotalLength),
					humanizeBytes(dStatus.DownloadSpeed),
					humanizeBytes(dStatus.UploadSpeed))
			}
		},
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Get/set global configuration",
		Run: func(cmd *cobra.Command, args [] string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			config, err := client.GetGlobalOption()
			if err != nil {
				log.Fatal(err)
			}

			jsonConfig, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				log.Fatal(err)
			}

			fmt.Printf("%s\n", jsonConfig)
		},
	}

	var peersCmd = &cobra.Command{
		Use: "peers [gid]",
		Short: "Get peer information for a torrent",
		Run: func(cmd *cobra.Command, args [] string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			printPeersForDownload := func(gid string) {
				peers, err := client.GetPeers(gid)
				if err != nil {
					log.Fatal(err)
				}

				if len(peers) > 0 {
					fmt.Println(gid)
					fmt.Println(strings.Repeat("-", 37))

					for _, peer := range(peers) {
						fmt.Printf("%15s:%5s  %6s  %6s\n",
							peer.Ip,
							peer.Port,
							humanizeBytes(peer.DownloadSpeed),
							humanizeBytes(peer.UploadSpeed))
					}

					fmt.Printf("\n")
				}
			}

			var gids []string

			if len(args) == 0 {
				downloads, err := client.TellActive("gid")
				if err != nil {
					log.Fatal(err)
				}

				for _, dwn := range downloads {
					gids = append(gids, dwn.Gid)
				}
			} else {
				gids = args
			}

			for _, gid := range gids  {
				printPeersForDownload(gid)
			}
		},
	}

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(peersCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
