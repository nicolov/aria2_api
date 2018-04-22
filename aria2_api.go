package aria2_api

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ybbus/jsonrpc"
	"math/bits"
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
	"bittorrent"}

//

func (aria *AriaClient) makeCall(methodName string, params ...interface{}) (resp *jsonrpc.RPCResponse, err error) {
	fullMethodName := "aria2." + methodName
	resp, err = aria.c.Call(fullMethodName, params...)

	if err != nil {
		return
	}

	if resp.Error != nil {
		err = fmt.Errorf("aria2: %s", resp.Error.Message)
	}

	return
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
	resp, err := aria.makeCall("getGlobalStat")

	if err != nil {
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
	Comment      string      `json:"comment"`
	CreationDate uint64      `json:"creationDate"`
	Mode         string      `json:"mode"`
	Info         TorrentInfo `json:"info"`
}

type DownloadStatus struct {
	Gid                    string         `json:"gid"`
	Status                 string         `json:"status"`
	TotalLength            uint64         `json:"totalLength,string"`
	CompletedLength        uint64         `json:"completedLength,string"`
	UploadLength           uint64         `json:"uploadLength,string"`
	DownloadSpeed          uint64         `json:"downloadSpeed,string"`
	UploadSpeed            uint64         `json:"uploadSpeed,string"`
	InfoHash               string         `json:"infoHash"`
	NumSeeders             uint64         `json:"numSeeders,string"`
	Dir                    string         `json:"dir"`
	Files                  []DownloadFile `json:"files"`
	Bittorrent             *TorrentStatus `json:"bittorrent"`
	VerifiedLength         uint64         `json:"verifiedLength,string"`
	VerifyIntegrityPending bool           `json:"verifyIntegrityPending,string"`
}

const StatusActive = "active"
const StatusWaiting = "waiting"
const StatusPaused = "paused"
const StatusError = "error"
const StatusComplete = "complete"
const StatusRemoved = "removed"

func (aria *AriaClient) TellStatus(downloadId string) (status DownloadStatus, err error) {
	resp, err := aria.makeCall("tellStatus", downloadId)
	if err != nil {
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

	resp, err := aria.makeCall("tellActive", keys)
	if err != nil {
		return
	}

	err = resp.GetObject(&list)
	return
}

//

type BtPeer struct {
	PeerId        string  `json:"peerId"`
	Ip            string  `json:"ip"`
	Port          string  `json:"port"`
	Bitfield      []uint8 `json:"bitfield"`
	AmChoking     bool    `json:"amChoking,string"`
	PeerChoking   bool    `json:"peerChoking,string"`
	DownloadSpeed uint64  `json:"downloadSpeed,string"`
	UploadSpeed   uint64  `json:"uploadSpeed,string"`
	Seeder        bool    `json:"seeder,string"`
}

// Custom marshalling for peer piece completion bitfield
// http://choly.ca/post/go-json-marshalling/
func (p *BtPeer) UnmarshalJSON(data []byte) error {
	type AliasedBtPeer BtPeer
	aux := &struct {
		Bitfield string `json:"bitfield""`
		*AliasedBtPeer
	}{
		AliasedBtPeer: (*AliasedBtPeer)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	bytes, err := hex.DecodeString(aux.Bitfield)
	if err != nil {
		return err
	}

	p.Bitfield = bytes
	return nil
}

// Return the number of pieces the peer has, and the total.
func (p *BtPeer) PiecesCompletedTotal() (completed int, total int) {
	for _, b := range p.Bitfield {
		completed += bits.OnesCount8(b)
	}
	total = 8 * len(p.Bitfield)
	return
}

func (aria *AriaClient) GetPeers(gid string) (peers []BtPeer, err error) {
	resp, err := aria.makeCall("getPeers", gid)
	if err != nil {
		return
	}

	err = resp.GetObject(&peers)
	return
}

//

func (aria *AriaClient) AddUri(uri string) (downloadId string, err error) {
	// aria2 expects an array of uris pointing to the same resource.
	// This is confusing and could cause corruption, so this
	// function takes in a single uri.
	uris := [...]string{uri}

	resp, err := aria.makeCall("addUri", uris)

	if err != nil {
		return
	}

	downloadId, err = resp.GetString()
	return
}

func (aria *AriaClient) AddTorrent(torrentPath string) (downloadId string, err error) {
	resp, err := aria.makeCall("addTorrent", torrentPath)

	if err != nil {
		return
	}

	downloadId, err = resp.GetString()
	return
}

//

func (aria *AriaClient) Pause(gid string) (downloadId string, err error) {
	resp, err := aria.makeCall("pause", gid)
	if err != nil {
		return
	}
	downloadId, err = resp.GetString()
	return
}

func (aria *AriaClient) ForcePause(gid string) (downloadId string, err error) {
	resp, err := aria.makeCall("forcePause", gid)
	if err != nil {
		return
	}
	downloadId, err = resp.GetString()
	return
}

//

func (aria *AriaClient) Remove(gid string) (downloadId string, err error) {
	resp, err := aria.makeCall("remove", gid)
	if err != nil {
		return
	}
	downloadId, err = resp.GetString()
	return
}

func (aria *AriaClient) ForceRemove(gid string) (downloadId string, err error) {
	resp, err := aria.makeCall("forceRemove", gid)
	if err != nil {
		return
	}
	downloadId, err = resp.GetString()
	return
}

//

func (aria *AriaClient) ListMethods() (methods []string, err error) {
	resp, err := aria.makeCall("listMethods")
	if err != nil {
		return
	}

	err = resp.GetObject(&methods)
	return
}

func (aria *AriaClient) ListNotifications() (notifications []string, err error) {
	resp, err := aria.makeCall("listNotifications")
	if err != nil {
		return
	}

	err = resp.GetObject(&notifications)
	return
}

func (aria *AriaClient) GetGlobalOption() (options map[string]string, err error) {
	resp, err := aria.makeCall("getGlobalOption")
	if err != nil {
		return
	}

	err = resp.GetObject(&options)
	return
}

func (aria *AriaClient) ChangeGlobalOption(options map[string]string) (err error) {
	resp, err := aria.makeCall("changeGlobalOption", options)
	if err != nil {
		return
	}

	s, err := resp.GetString()

	if err != nil {
		return
	}

	if s != "OK" {
		err = fmt.Errorf(s)
	}

	return
}
