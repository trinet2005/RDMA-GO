package main

/*
#cgo LDFLAGS: -libverbs
*/
import "C"
import (
	"fmt"
	RDMA "github.com/trinet2005/RDMA-GO"
)

func main() {

	config, err := RDMA.LoadConfig("config.json")
	if err != nil {
		fmt.Println("LoadConfig Error: ", err)
		return
	}

	RDMA.InitLog(config.Debug)

	RDMA.LogInfo("InitIBRes start")

	ibRes, err := RDMA.InitIBRes()
	if err != nil {
		RDMA.LogError("InitIBRes Error: ", err)
		return
	}
	defer ibRes.FreeIBRes()

	qpInfo, err := ibRes.InitRCQP(config.DeviceName, config.MrSize)
	if err != nil {
		RDMA.LogError("InitRCQP Error: ", err)
		return
	}
	defer func(ibRes *RDMA.IBRes) {
		err := ibRes.FreeRCQP()
		if err != nil {
			RDMA.LogDebug(fmt.Sprintf("FreeRCQP Error: ", err))
		}
	}(ibRes)

	info, err := RDMA.ConmunicateQPInfo(config, qpInfo)
	if err != nil {
		RDMA.LogError("ConmunicateQPInfo Error: ", err)
		return
	}

	err = ibRes.ModifyQPRTS(info)
	if err != nil {
		RDMA.LogError("ModifyQPRTS Error: ", err)
		return
	}

	RDMA.LogInfo("ModifyQPRTS success")
	RDMA.LogDebug("ModifyQPRTS debug message")

	switch config.Mode {
	case "server":
		err := ibRes.ListenServer(1, 1)
		if err != nil {
			RDMA.LogError("ListenServer Error: ", err)
			return
		}
	case "client":
		err := ibRes.StartClient(1, 1, config.FileName)
		if err != nil {
			RDMA.LogError("StartClient Error: ", err)
			return
		}
	}

}
