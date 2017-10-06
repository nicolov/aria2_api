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

const rpcPort = 6802
var endpointUrl = fmt.Sprintf("http://localhost:%d/jsonrpc", rpcPort)

const cfgDefaultMaxConcurrentDownloads = "7"

// Wrap aria2 daemon setup/teardown code
func testWithAriaDaemon(t *testing.T, f func(*testing.T)) {
	ariaCmd := exec.Command("aria2c",
		"--enable-rpc",
		"--rpc-listen-all",
		fmt.Sprintf("--rpc-listen-port=%d", rpcPort),
		"--max-concurrent-downloads=" + cfgDefaultMaxConcurrentDownloads)

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
	mux := http.NewServeMux()
	mux.HandleFunc("/test_file/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello world!")
	})
	mux.HandleFunc("/slow_resp/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		fmt.Fprint(w, "(Slow) hello world!")
	})

	go func() {
		http.ListenAndServe(":7575", mux)
	}()

	// Actually run the test now
	f(t)

	// Tear down
	ariaCmd.Process.Kill()
}

// Test that we can successfully download a single file
func TestDownload(t *testing.T) {
	testWithAriaDaemon(t, func(t *testing.T) {
		r := require.New(t)

		const sampleUri = "http://localhost:7575/test_file"

		client := NewAriaClient(endpointUrl)

		// No download in the queue
		stats, err := client.GetGlobalStat()
		r.NoError(err)
		r.Equal(stats.NumActive, 0)
		r.Equal(stats.NumWaiting, 0)
		r.Equal(stats.NumStopped, 0)

		// Add an URI
		downloadId, err := client.AddUri(sampleUri)
		r.NoError(err)
		r.NotEqual("", downloadId)

		time.Sleep(20 * time.Millisecond)

		stats, _ = client.GetGlobalStat()
		r.Equal(stats.NumActive, 0)
		r.Equal(stats.NumStopped, 1)

		downloadStatus, err := client.TellStatus(downloadId)
		r.NoError(err)
		r.Equal("complete", downloadStatus.Status)

		r.Equal(len(downloadStatus.Files), 1)
		downloadPath := downloadStatus.Files[0].Path
		_, err = os.Stat(downloadPath)
		r.False(os.IsNotExist(err), "Downloaded file not found")

		// Clean up
		_ = os.Remove(downloadPath)
	})
}

// Test getting/setting global options
func TestGlobalOption(t *testing.T) {
	testWithAriaDaemon(t, func (t* testing.T) {
		r := require.New(t)
		client := NewAriaClient(endpointUrl)

		// Check that it's set to the initial value
		cfg, err := client.GetGlobalOption()
		r.NoError(err)
		r.Equal(cfg["max-concurrent-downloads"], cfgDefaultMaxConcurrentDownloads)

		// Set a new value
		err = client.ChangeGlobalOption(
			map[string]string{"max-concurrent-downloads": "5"})
		r.NoError(err)

		// Check again
		cfg, err = client.GetGlobalOption()
		r.NoError(err)
		r.Equal(cfg["max-concurrent-downloads"], "5")

	})
}

// Test queue
func TestQueue(t *testing.T) {
	testWithAriaDaemon(t, func(t *testing.T) {
		r := require.New(t)
		client := NewAriaClient(endpointUrl)

		// No download in the queue
		stats, err := client.GetGlobalStat()
		r.NoError(err)
		r.Equal(stats.NumActive, 0)
		r.Equal(stats.NumStopped, 0)
		r.Equal(stats.NumWaiting, 0)

		// Set concurrency to one
		err = client.ChangeGlobalOption(
			map[string]string{"max-concurrent-downloads": "1"})
		r.NoError(err)

		// Add the first download that starts right away
		idRunning, err := client.AddUri("http://localhost:7575/slow_resp/1")
		r.NoError(err)

		time.Sleep(20 * time.Millisecond)

		// One download running
		stats, _ = client.GetGlobalStat()
		r.Equal(stats.NumActive, 1)
		r.Equal(stats.NumStopped, 0)
		r.Equal(stats.NumWaiting, 0)

		// Add a second that will be queued
		idWaiting, err := client.AddUri("http://localhost:7575/slow_resp/2")
		r.NoError(err)

		// One running, one queued
		stats, _ = client.GetGlobalStat()
		r.Equal(stats.NumActive, 1)
		r.Equal(stats.NumStopped, 0)
		r.Equal(stats.NumWaiting, 1)

		// Check status
		downloadStatus, err := client.TellStatus(idRunning)
		r.NoError(err)
		r.Equal("active", downloadStatus.Status)

		downloadStatus, err = client.TellStatus(idWaiting)
		r.NoError(err)
		r.Equal("waiting", downloadStatus.Status)
	})
}
