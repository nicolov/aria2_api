package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/nicolov/aria2_api"
	"github.com/spf13/cobra"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
)

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

// Return the percentage, or "done"
func toPercentage(done uint64, total uint64) string {
	switch {
	case total == 0:
		return "-"
	case done == total:
		return "done"
	default:
		return fmt.Sprintf("%0.1f%%",
			100.0*float64(done)/float64(total))
	}
}

func main() {
	var rootCmd = &cobra.Command{
		Use: "aria2_cli",
	}

	var endpointUrl string
	rootCmd.PersistentFlags().StringVarP(&endpointUrl,
		"endpoint_url", "u",
		"http://127.0.0.1:6800/jsonrpc",
		"Endpoint url")

	var listCmd = &cobra.Command{
		Use:   "list",
		Short: "List torrents",
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			stats, err := client.TellActive()

			if err != nil {
				log.Fatal(err)
			}

			// Compute summary statistics
			var summaryStats struct {
				completedLength uint64
				totalLength     uint64
				downloadSpeed   uint64
				uploadSpeed     uint64
			}

			const lineFormatStr = "%4s  %20s  %5s  %1s  %6s  %6s  %6s  %6s\n"

			for _, dStatus := range stats {
				summaryStats.completedLength += dStatus.CompletedLength
				summaryStats.totalLength += dStatus.TotalLength
				summaryStats.downloadSpeed += dStatus.DownloadSpeed
				summaryStats.uploadSpeed += dStatus.UploadSpeed
			}

			// Print summary line
			totalCount := fmt.Sprintf("total (%d)", len(stats))
			fmt.Printf(lineFormatStr,
				"", totalCount, "", "",
				humanizeBytes(summaryStats.completedLength),
				humanizeBytes(summaryStats.totalLength),
				humanizeBytes(summaryStats.downloadSpeed),
				humanizeBytes(summaryStats.uploadSpeed))
			fmt.Printf("\n")

			for _, dStatus := range stats {
				// Try to determine display name
				var displayName string
				displayName = "n/a"
				if dStatus.Bittorrent != nil && dStatus.Bittorrent.Info.Name != "" {
					displayName = dStatus.Bittorrent.Info.Name
				}

				// Percent completion
				var pctComplete string
				if dStatus.VerifiedLength > 0 {
					pctComplete = toPercentage(dStatus.VerifiedLength, dStatus.TotalLength)
				} else {
					pctComplete = toPercentage(dStatus.CompletedLength, dStatus.TotalLength)
				}

				// Default to the first letter of the status
				var statusLabel string
				switch {
				case dStatus.VerifyIntegrityPending:
					statusLabel = "q" // waiting in the hash check queue
				case dStatus.VerifiedLength > 0:
					statusLabel = "k" // checking
				default:
					// Just use the first letter of the status word
					statusLabel = dStatus.Status[:1]
				}

				fmt.Printf(lineFormatStr,
					dStatus.Gid[:4],
					displayName,
					pctComplete,
					statusLabel,
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
		Args: func(cmd *cobra.Command, args []string) error {
			if !(len(args) == 0 || len(args) == 2) {
				return errors.New("config requires either 0, or 2 arguments")
			}
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			if len(args) == 0 {
				// Print configuration
				config, err := client.GetGlobalOption()
				if err != nil {
					log.Fatal(err)
				}

				jsonConfig, err := json.MarshalIndent(config, "", "  ")
				if err != nil {
					log.Fatal(err)
				}

				fmt.Printf("%s\n", jsonConfig)
			} else {
				// Set configuration
				configChange := map[string]string{
					args[0]: args[1],
				}

				err := client.ChangeGlobalOption(configChange)

				if err != nil {
					log.Fatal(err)
				}
			}
		},
	}

	var peersCmd = &cobra.Command{
		Use:   "peers [gid]",
		Short: "Get peer information for a torrent",
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			printPeersForDownload := func(gid string) {
				peers, err := client.GetPeers(gid)
				if err != nil {
					log.Fatal(err)
				}

				if len(peers) > 0 {
					fmt.Println(gid)
					fmt.Println(strings.Repeat("-", 44))

					for _, peer := range peers {
						complPieces, totalPieces := peer.PiecesCompletedTotal()
						fmt.Printf("%15s:%5s  %6s  %6s  %.1f%%\n",
							peer.Ip,
							peer.Port,
							humanizeBytes(peer.DownloadSpeed),
							humanizeBytes(peer.UploadSpeed),
							100*float64(complPieces)/float64(totalPieces))
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

			for _, gid := range gids {
				printPeersForDownload(gid)
			}
		},
	}

	var addCmd = &cobra.Command{
		Use:   "addU [url]",
		Short: "Add URLs to the download queue",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			var gids []string

			for _, uri := range args {
				gid, err := client.AddUri(uri)
				if err != nil {
					log.Printf("%s: %v", uri, err)
				} else {
					gids = append(gids, gid)
				}
			}

			fmt.Println(gids)
		},
	}

	var addTorrentCmd = &cobra.Command{
		Use:   "addT [torrent file path]",
		Short: "Add .torrent file to the download queue",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			var gids []string

			for _, torrentPath := range args {
				// Open and base64 encode the .torrent file
				contents, err := ioutil.ReadFile(torrentPath)
				if err != nil {
					log.Printf("FAIL %s: %v", torrentPath, err)
				}
				b64Contents := base64.StdEncoding.EncodeToString(contents)

				gid, err := client.AddTorrent(b64Contents)
				if err != nil {
					log.Printf("FAIL %s: %v", torrentPath, err)
				} else {
					gids = append(gids, gid)
				}
			}

			fmt.Println(gids)
		},
	}

	var pauseCmd = &cobra.Command{
		Use:   "pause [gid, ...]",
		Short: "Pause torrent",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			var gids []string

			for _, gid := range args {
				gidReply, err := client.Pause(gid)
				if err != nil || gid != gidReply {
					log.Printf(gidReply)
					log.Printf("FAIL %s: %v", gid, err)
				} else {
					gids = append(gids, gid)
				}
			}

			fmt.Println(gids)
		},
	}

	var forcePauseCmd = &cobra.Command{
		Use:   "forcePause [gid, ...]",
		Short: "Force pause torrent",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			var gids []string

			for _, gid := range args {
				gidReply, err := client.ForcePause(gid)
				if err != nil || gid != gidReply {
					log.Printf(gidReply)
					log.Printf("FAIL %s: %v", gid, err)
				} else {
					gids = append(gids, gid)
				}
			}

			fmt.Println(gids)
		},
	}

	var removeCmd = &cobra.Command{
		Use:   "remove [gid, ...]",
		Short: "Remove torrent",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			var gids []string

			for _, gid := range args {
				gidReply, err := client.Remove(gid)
				if err != nil || gid != gidReply {
					log.Printf(gidReply)
					log.Printf("FAIL %s: %v", gid, err)
				} else {
					gids = append(gids, gid)
				}
			}

			fmt.Println(gids)
		},
	}

	var forceRemoveCmd = &cobra.Command{
		Use:   "forceRemove [gid, ...]",
		Short: "Force remove torrent",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			client := aria2_api.NewAriaClient(endpointUrl)

			var gids []string

			for _, gid := range args {
				gidReply, err := client.ForceRemove(gid)
				if err != nil || gid != gidReply {
					log.Printf(gidReply)
					log.Printf("FAIL %s: %v", gid, err)
				} else {
					gids = append(gids, gid)
				}
			}

			fmt.Println(gids)
		},
	}

	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(peersCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(addTorrentCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(forcePauseCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(forceRemoveCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
