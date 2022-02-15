/*
 * @File:    utils.go
 * @Date:    2022.02.15 21:58
 * @Author:  Max Xu
 * @Contact: xuhuan@live.cn
 */

package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type ZookeeperStat struct {
	Version     string `json:"version"`
	ReadOnly    bool   `json:"read_only"`
	ServerStats struct {
		PacketsSent               int `json:"packets_sent"`
		PacketsReceived           int `json:"packets_received"`
		FsyncThresholdExceedCount int `json:"fsync_threshold_exceed_count"`
		ClientResponseStats       struct {
			LastBufferSize int `json:"last_buffer_size"`
			MinBufferSize  int `json:"min_buffer_size"`
			MaxBufferSize  int `json:"max_buffer_size"`
		} `json:"client_response_stats"`
		ServerState               string  `json:"server_state"`
		ProviderNull              bool    `json:"provider_null"`
		DataDirSize               int     `json:"data_dir_size"`
		LogDirSize                int     `json:"log_dir_size"`
		LastProcessedZxid         int     `json:"last_processed_zxid"`
		OutstandingRequests       int     `json:"outstanding_requests"`
		AvgLatency                float64 `json:"avg_latency"`
		MaxLatency                int     `json:"max_latency"`
		MinLatency                int     `json:"min_latency"`
		NumAliveClientConnections int     `json:"num_alive_client_connections"`
		AuthFailedCount           int     `json:"auth_failed_count"`
		NonMtlsremoteConnCount    int     `json:"non_mtlsremote_conn_count"`
		NonMtlslocalConnCount     int     `json:"non_mtlslocal_conn_count"`
		Uptime                    int     `json:"uptime"`
	} `json:"server_stats"`
	ClientResponse struct {
		LastBufferSize int `json:"last_buffer_size"`
		MinBufferSize  int `json:"min_buffer_size"`
		MaxBufferSize  int `json:"max_buffer_size"`
	} `json:"client_response"`
	NodeCount         int           `json:"node_count"`
	Connections       []interface{} `json:"connections"`
	SecureConnections []interface{} `json:"secure_connections"`
	Command           string        `json:"command"`
	Error             interface{}   `json:"error"`
}

func GetZookeeperStat(podIP string) (*ZookeeperStat, error) {
	response, err := http.Get(fmt.Sprintf("http://%s:8080/commands/stat", podIP))
	if err != nil {
		return nil, err
	}

	zs := &ZookeeperStat{}
	if err = json.NewDecoder(response.Body).Decode(zs); err != nil {
		return nil, err
	}

	return zs, nil
}
