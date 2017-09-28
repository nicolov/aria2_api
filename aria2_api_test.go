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

func TestMain(m *testing.M) {
	errs := make(chan int, 1)

	ariaCmd := exec.Command("aria2c",
		"--enable-rpc",
		"--rpc-listen-all")

	// Kill aria even if the golang code panics
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

		time.Sleep(200 * time.Millisecond)
		retries += 1
	}

	// HTTP server for test files
	http.HandleFunc("/test_file", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello world")
	})
	go func() {
		err := http.ListenAndServe(":7575", nil)

		if err != nil {
			errs <- -1
		}
	}()

	// test
	go func() {
		errs <- m.Run()
	}()

	res := <-errs

	os.Exit(res)
}

// Test that we can successfully download a single file
func TestDownload(t *testing.T) {
	require := require.New(t)

	const sampleUri = "http://localhost:7575/test_file"

	client := NewAriaClient(endpointUrl)

	// No download in the queue
	stats, _ := client.GetGlobalStat()
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
}

// Test queue
func TestQueue(t *testing.T) {
	//r := require.New(t)
	//client := NewAriaClient(endpointUrl)

}