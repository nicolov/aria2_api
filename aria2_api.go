package aria2_api

import (
	"fmt"
	"github.com/ybbus/jsonrpc"
)

type AriaClient struct {
	c *jsonrpc.RPCClient
}

func NewAriaClient(endpoint string) *AriaClient {
	return &AriaClient{
		c: jsonrpc.NewRPCClient(endpoint),
	}
}

//

type GlobalStat struct {
	DownloadSpeed string `json:"downloadSpeed"`
	UploadSpeed   string `json:"uploadSpeed"`
	NumActive     string `json:"numActive"`
	NumWaiting    string `json:"numWaiting"`
	NumStopped    string `json:"numStopped"`
}

func (aria *AriaClient) GetGlobalStat() (stat GlobalStat, err error) {
	resp, err := aria.c.Call("aria2.getGlobalStat")

	if err != nil {
		fmt.Println(err)
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf(resp.Error.Message)
		return
	}

	err = resp.GetObject(&stat)
	return
}

//

type DownloadFile struct {
	Index           string `json:"index"`
	Path            string `json:"path"`
	Length          string `json:"length"`
	CompletedLength string `json:"completedLength"`
	Selected        string `json:"selected"`
}

type DownloadStatus struct {
	Gid             string `json:"string"`
	Status          string `json:"status"`
	TotalLength     string `json:"totalLength"`
	CompletedLength string `json:"completedLength"`
	UploadedLength  string `json:"uploadedLength"`
	DownloadSpeed   string `json:"downloadSpeed"`
	UploadSpeed     string `json:"uploadSpeed"`
	InfoHash        string `json:"infoHash"`
	Dir             string `json:"dir"`
	Files           []DownloadFile `json:"files"`
}

func (aria *AriaClient) TellStatus(downloadId DownloadId) (status DownloadStatus, err error) {
	resp, err := aria.c.Call("aria2.tellStatus", string(downloadId))

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf(resp.Error.Message)
		return
	}

	err = resp.GetObject(&status)
	return
}

//

type DownloadId string

func (aria *AriaClient) AddUri(uri string) (downloadId DownloadId, err error) {
	uris := [1]string{uri}

	resp, err := aria.c.Call("aria2.addUri", uris)

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf(resp.Error.Message)
		return
	}

	s, err := resp.GetString()
	if err != nil {
		return
	}

	downloadId = DownloadId(s)
	return
}
