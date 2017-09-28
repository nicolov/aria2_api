package aria2_api

import (
	"testing"
	"os"
	"os/exec"
	"log"
	"net/http"
	"time"
	"syscall"
	"github.com/stretchr/testify/require"
	"fmt"
)

const endpointUrl = "http://localhost:6800/jsonrpc"

// Wrap aria2 daemon setup/teardown code
func testWithAriaDaemon(t *testing.T, f func(*testing.T)) {
	ariaCmd := exec.Command("aria2c",
		"--enable-rpc",
		"--rpc-listen-all")

	// Set up a process group so that aria is killed even if the golang
	// code panics.
	ariaCmd.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	err := ariaCmd.Start()
	if err != nil {
		log.Panic(err)
	}

	// wait for Aria to be ready
	retries := 0
	for {
		_, err := http.Get(endpointUrl)

		if err == nil || retries > 10 {
			break
		}

		time.Sleep(50 * time.Millisecond)
		retries += 1
	}

	// HTTP server for test files
	http.HandleFunc("/test_file", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello world")
	})
	go func() {
		http.ListenAndServe(":7575", nil)
	}()

	// Actually run the test now
	f(t)

	// Tear down
	ariaCmd.Process.Kill()
}

// Test that we can successfully download a single file
func TestDownload(t *testing.T) {
	testWithAriaDaemon(t, func(t *testing.T) {
		require := require.New(t)

		const sampleUri = "http://localhost:7575/test_file"

		client := NewAriaClient(endpointUrl)

		// No download in the queue
		stats, err := client.GetGlobalStat()
		require.NoError(err)
		require.Equal(stats.NumActive, "0")
		require.Equal(stats.NumStopped, "0")

		// Add an URI
		downloadId, err := client.AddUri(sampleUri)
		require.NoError(err)
		require.NotEqual("", downloadId)

		time.Sleep(20 * time.Millisecond)

		stats, _ = client.GetGlobalStat()
		require.Equal(stats.NumActive, "0")
		require.Equal(stats.NumStopped, "1")

		downloadStatus, err := client.TellStatus(downloadId)
		require.NoError(err)
		require.Equal("complete", downloadStatus.Status)

		require.Equal(len(downloadStatus.Files), 1)
		downloadPath := downloadStatus.Files[0].Path
		_, err = os.Stat(downloadPath)
		require.False(os.IsNotExist(err), "Downloaded file not found")

		// Clean up
		_ = os.Remove(downloadPath)
	})
}

//// Test queue
//func TestQueue(t *testing.T) {
//	r := require.New(t)
//	client := NewAriaClient(endpointUrl)
//
//	// No download in the queue
//	stats, _ := client.GetGlobalStat()
//	r.Equal(stats.NumActive, "0")
//	r.Equal(stats.NumStopped, "0")
//}
