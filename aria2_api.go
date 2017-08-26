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

var defaultStatusKeys = [...]string{
	"gid",
	"status",
	"totalLength",
	"completedLength",
	"uploadLength",
	"downloadSpeed",
	"uploadSpeed",
	"infoHash",
	"numSeeders",
	"connections",
	"bittorrent",}

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

type TorrentInfo struct {
	Name string `json:"name"`
}

type TorrentStatus struct {
	//AnnounceList []string `json:"announceList"`
	Comment      string `json:"comment"`
	CreationDate uint64 `json:"creationDate"`
	Mode         string `json:"mode"`
	Info         TorrentInfo `json:"info"`
}

type DownloadStatus struct {
	Gid             string `json:"gid"`
	Status          string `json:"status"`
	TotalLength     uint64 `json:"totalLength,string"`
	CompletedLength uint64 `json:"completedLength,string"`
	UploadedLength  uint64 `json:"uploadedLength,string"`
	DownloadSpeed   uint64 `json:"downloadSpeed,string"`
	UploadSpeed     uint64 `json:"uploadSpeed,string"`
	InfoHash        string `json:"infoHash"`
	Dir             string `json:"dir"`
	Files           []DownloadFile `json:"files"`
	Bittorrent      *TorrentStatus `json:"bittorrent"`
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

type DownloadStatusList []DownloadStatus

func (aria *AriaClient) TellActive(keys ...string) (list DownloadStatusList, err error) {
	// Default response keys
	if len(keys) == 0 {
		keys = defaultStatusKeys[:]
	}

	resp, err := aria.c.Call("aria2.tellActive", keys)

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf(resp.Error.Message)
		return
	}

	err = resp.GetObject(&list)
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

//

func (aria *AriaClient) ListMethods() (methods []string, err error) {
	resp, err := aria.c.Call("aria2.listMethods")

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf(resp.Error.Message)
		return
	}

	err = resp.GetObject(&methods)
	return
}

func (aria *AriaClient) ListNotifications() (notifications []string, err error) {
	resp, err := aria.c.Call("aria2.listNotifications")

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf(resp.Error.Message)
		return
	}

	err = resp.GetObject(&notifications)
	return
}
