package RDMAGO

/*
#cgo CFLAGS: -I/usr/include/infiniband
#cgo LDFLAGS: -libverbs -L. -lwrapper

#include "wrapper.h"

#include <stdlib.h>
#include <infiniband/verbs.h>
*/
import "C"
import (
	"bufio"
	"errors"
	"fmt"
	"unsafe"
)

type IBRes struct {
	Ctx       *C.struct_ibv_context
	Pd        *C.struct_ibv_pd
	Mr        *C.struct_ibv_mr
	Cq        *C.struct_ibv_cq
	Qp        *C.struct_ibv_qp
	Srq       *C.struct_ibv_srq
	PortAttr  *C.struct_ibv_port_attr
	DevAttr   C.struct_ibv_device_attr
	Gid       C.union_ibv_gid
	NumQps    int
	IbBuf     *C.char
	IbBufSize C.size_t

	DeviceIndex C.int
}

type QPInfo struct {
	QpNum C.uint
	Lid   C.ushort
	Gid   C.union_ibv_gid
}

type IBVQPAttr struct {
	QPState         int
	pathMTU         int
	qkey            C.uint
	RQPsn           C.uint
	SQPsn           C.uint
	destQPNum       C.uint
	QPAccessFlages  C.uint
	pkeyIndex       uint16
	portNum         C.uchar
	maxDestRDAtomic C.uchar
	minRNRTimer     C.uchar
	timeout         C.uchar
	retryCount      C.uchar
	rnrRetry        C.uchar
	maxRdAtomic     C.uchar
	ahAttr          IBVAHAttr
}
type IBVAHAttr struct {
	grhDgid      C.union_ibv_gid
	grhHopLimit  C.uchar
	grhSgidIndex C.uchar

	dlid        C.ushort
	SL          C.uchar
	srcPathBits C.uchar
	portNum     C.uchar
	isGlobal    C.uchar
}

const (
	IB_PORT      = 1
	IBV_PORT_NUM = 1

	TOT_NUM_OPS   = 1
	IB_WR_ID_STOP = 0xE000000000000000
)

const (
	MSG_CLIENT_START = 100
	MSG_CLIENT_STOP  = 101
)

func InitIBRes() (*IBRes, error) {
	ibRes := (*IBRes)(C.malloc(C.size_t(unsafe.Sizeof(IBRes{}))))
	if ibRes == nil {
		return nil, fmt.Errorf("Failed to allocate memory for IBRes")
	}

	//clear memory
	C.memset(unsafe.Pointer(ibRes), 0, C.size_t(unsafe.Sizeof(*ibRes)))
	return ibRes, nil
}

func (ibRes *IBRes) FreeIBRes() {
	C.free(unsafe.Pointer(ibRes))
	return
}

// InitRCQP: init RC
func (ibRes *IBRes) InitRCQP(deviceName string, MRSize int) (*QPInfo, error) {
	// get device context
	deviceAttr, err := GetIbvDeviceContext(ibRes, deviceName)
	if err != nil {
		return nil, errors.New("[InitRCQP] get IBV device context failed")
	}
	ibRes.Ctx = deviceAttr.Ctx

	// alloc PD
	err = IbvAllocPD(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] alloc PD failed")
	}

	// query port
	err = IbvQueryPort(ibRes, 1)
	if err != nil {
		return nil, errors.New("[InitRCQP] query port failed")
	}

	// query gid
	err = IbvQueryGid(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] query gid failed")
	}

	// alloc MR
	err = IbvRegMR(ibRes, MRSize)
	if err != nil {
		return nil, errors.New("[InitRCQP] regist MR failed")
	}

	// query device attr
	err = IbvQueryDevice(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] query device failed")
	}

	// create CQ
	err = IbvCreateCQ(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] create CQ failed")
	}

	// create SRQ
	err = IbvCreateSRQ(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] create SRQ failed")
	}

	// create QP
	err = IbvCreateQP(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] create QP failed")
	}

	//get QP info
	qpInfo, err := GetQPInfo(ibRes)
	if err != nil {
		return nil, errors.New("[InitRCQP] get QP info failed")
	}

	return qpInfo, nil
}

func (ibRes *IBRes) FreeRCQP() error {

	err := IbvDestroyQP(ibRes)
	if err != nil {
		return errors.New("[DestroyRCQP] destroy QP failed")
	}
	LogDebug("QP destroyed")

	err = IbvDestroySRQ(ibRes)
	if err != nil {
		return errors.New("[DestroyRCQP] destroy SRQ failed")
	}
	LogDebug("SRQ destroyed")

	err = IbvDestroyCQ(ibRes)
	if err != nil {
		return errors.New("[DestroyRCQP] destroy CQ failed")
	}
	LogDebug("CQ destroyed")

	err = IbvDeregMR(ibRes)
	if err != nil {
		return errors.New("[DestroyRCQP] dereg MR failed")
	}
	LogDebug("MR deregistered")

	err = IbvDeallocPD(ibRes)
	if err != nil {
		return errors.New(fmt.Sprintf("[DestroyRCQP] dealloc PD failed %v", err))
	}
	LogDebug("PD deallocated")

	err = IbvCloseDevice(ibRes)
	if err != nil {
		return errors.New("[DestroyRCQP] close device failed")
	}
	LogDebug("Device closed")

	return nil
}

func ConmunicateQPInfo(socketConfig *Config, info *QPInfo) (*QPInfo, error) {
	var QPInfo *QPInfo
	var err error

	switch socketConfig.Mode {
	case "server":
		err, QPInfo = StartServer(socketConfig.Port, *info)
		if err != nil {
			return nil, errors.New("[ConmunicateQPInfo] start server failed with error: " + err.Error())
		}
	case "client":
		err, QPInfo = StartClient(socketConfig.Address, *info)
		if err != nil {
			return nil, errors.New("[ConmunicateQPInfo] start client failed" + err.Error())
		}
	default:
		return nil, errors.New("[ConmunicateQPInfo] invalid mode")
	}

	return QPInfo, nil
}

func (ibRes *IBRes) ModifyQPRTS(qpInfo *QPInfo) error {
	err := IbvModifyQPInit(ibRes.Qp)
	if err != nil {
		return errors.New("[ModifyQPRTS] modify QP to Init failed")
	}

	err = IbvModifyQPRTRDefault(ibRes.Qp, qpInfo.QpNum, qpInfo.Lid, qpInfo.Gid)
	if err != nil {
		return errors.New("[ModifyQPRTS] modify QP to RTR failed")
	}

	err = IbvModifyQPRTSDefault(ibRes.Qp)
	if err != nil {
		return errors.New("[ModifyQPRTS] modify QP to RTS failed")
	}

	return nil
}

func (ibRes *IBRes) SetIbBuf(buf string) {
	C.strcpy(ibRes.IbBuf, C.CString(buf))
}

func (ibRes *IBRes) SetIbBufWithBytes(buf []byte) {
	C.strcpy(ibRes.IbBuf, (*C.char)(unsafe.Pointer(&buf[0])))
}

func (ibRes *IBRes) GetIbBuf() string {
	return C.GoString(ibRes.IbBuf)
}

func (ibRes *IBRes) ListenServer(peerNum, cuurentMsgNum int) error {
	LogDebug("ListenServer start")

	var err error
	/* pre-post recvs */
	for i := 0; i < peerNum; i++ {
		for j := 0; j < 2; j++ {
			wrID := uintptr(unsafe.Pointer(ibRes.IbBuf))
			err = IbvPostSRQRecvRes(ibRes, uint64(wrID))
			if err != nil {
				return errors.New("post SRQ recv failed")
			}
			LogDebug(fmt.Sprintf("--- [SERVER] SRQ recv posted wrID = %v", wrID))
		}
	}
	LogDebug("pre-post recvs done")

	for i := 0; i < peerNum; i++ {
		wrID := uintptr(unsafe.Pointer(ibRes.IbBuf))
		err = IbvPostSendRes(ibRes, MSG_CLIENT_START, uint64(wrID))
		if err != nil {
			return errors.New("post start send failed")
		}
	}
	LogDebug("post start send done")

	wc, err := CreateWC(10)
	if err != nil {
		return errors.New("create WC failed")
	}
	defer DestroyWC(wc)

	LogDebug("start to poll CQ")
	var stop bool
	for stop != true {

		var num, opsCount int
		for num < 1 {
			num, err = IbvPollCQ(ibRes.Cq, 10, wc)
			if err != nil {
				return errors.New("poll CQ failed")
			}
		}

		for i := 0; i < num; i++ {
			offset := uintptr(i * int(unsafe.Sizeof(C.struct_ibv_wc{})))
			newPtr := unsafe.Pointer(uintptr(unsafe.Pointer(wc)) + offset)
			wcPtr := (*C.struct_ibv_wc)(newPtr)
			LogDebug(fmt.Sprintf("wcStatus:%v wcOPcode: %v \n", wcPtr.status, wcPtr.opcode))

			if wcPtr.status != C.IBV_WC_SUCCESS {
				if wcPtr.opcode == C.IBV_WC_RECV {
					return errors.New("server recv failed")
				} else if wcPtr.opcode == C.IBV_WC_SEND {
					return errors.New("server send failed")
				} else {
					return errors.New("server unknown wc failed")
				}
			} else {
				if wcPtr.opcode == C.IBV_WC_RECV {
					opsCount++
					msgPtr := unsafe.Pointer(uintptr(wcPtr.wr_id))
					LogInfo(fmt.Sprintf("WC RECV msg:%v\n", C.GoString((*C.char)(msgPtr))))
					if opsCount == TOT_NUM_OPS {
						stop = true
						break
					}
				}
			}
		}
	}
	LogDebug("stop pull CQ")

	LogDebug("start to send stop")
	for i := 0; i < peerNum; i++ {
		err = IbvPostSendRes(ibRes, MSG_CLIENT_STOP, IB_WR_ID_STOP)
		if err != nil {
			return errors.New("post stop send failed ")
		}
		LogDebug(fmt.Sprintf("[SEND] IB_WR_ID_STOP ,immData = %v \n", MSG_CLIENT_STOP))
	}
	LogDebug("stop send done")

	LogDebug("start to poll CQ")
	stop = false
	for stop != true {
		var num, numAckedPeers int
		for num < 1 {
			num, err = IbvPollCQ(ibRes.Cq, 10, wc)
			if err != nil {
				return errors.New("poll CQ failed")
			}
		}

		for i := 0; i < num; i++ {
			offset := uintptr(i * int(unsafe.Sizeof(C.struct_ibv_wc{})))
			newPtr := unsafe.Pointer(uintptr(unsafe.Pointer(wc)) + offset)
			wcPtr := (*C.struct_ibv_wc)(newPtr)
			LogDebug(fmt.Sprintf("wcStatus:%v wcOPcode: %v \n", wcPtr.status, wcPtr.opcode))

			if wcPtr.status != C.IBV_WC_SUCCESS {
				if wcPtr.opcode == C.IBV_WC_RECV {
					return errors.New("server recv failed")
				} else if wcPtr.opcode == C.IBV_WC_SEND {
					return errors.New("server send failed")
				} else {
					return errors.New("server unknown wc failed")
				}
			} else {
				if wcPtr.opcode == C.IBV_WC_SEND {
					if wcPtr.wr_id == IB_WR_ID_STOP {
						numAckedPeers++
						if numAckedPeers == peerNum {
							LogDebug("stop")
							stop = true
							break
						}
					}
				}
			}
		}
	}
	return nil
}

func (ibRes *IBRes) StartClient(peerNum, cuurentMsgNum int, fileName string) error {
	LogDebug("start connection")
	var err error
	/* pre-post recvs */
	for i := 0; i < peerNum; i++ {
		for j := 0; j < 2; j++ {
			wrID := uintptr(unsafe.Pointer(ibRes.IbBuf))
			err = IbvPostSRQRecvRes(ibRes, uint64(wrID))
			if err != nil {
				return errors.New("post SRQ recv failed")
			}
			LogDebug(fmt.Sprintf("--- [CLIENT] SRQ recv posted wrID = %v", wrID))
			//ibRes.IbBuf = (*C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(ibRes.IbBuf)) + unsafe.Sizeof(C.struct_ibv_wc{})))
		}
	}
	LogDebug("pre-post recvs done")

	wc, err := CreateWC(10)
	if err != nil {
		return errors.New("create WC failed")
	}

	LogDebug("start polling CQ")
	var startSend bool
	for startSend != true {
		var num int
		for num < 1 {
			num, err = IbvPollCQ(ibRes.Cq, 10, wc)
			if err != nil {
				return errors.New("poll CQ failed")
			}
		}
		LogDebug(fmt.Sprintf("  Poll CQ num: %v ", num))

		var currentReady int
		for i := 0; i < num; i++ {
			offset := uintptr(i * int(unsafe.Sizeof(C.struct_ibv_wc{})))
			newPtr := unsafe.Pointer(uintptr(unsafe.Pointer(wc)) + offset)
			wcPtr := (*C.struct_ibv_wc)(newPtr)
			LogDebug(fmt.Sprintf("wcStatus:%v wcOPcode: %v \n", wcPtr.status, wcPtr.opcode))

			if wcPtr.status == C.IBV_WC_SUCCESS && wcPtr.opcode == C.IBV_WC_RECV {

				//wrID := uintptr(unsafe.Pointer(ibRes.IbBuf))
				//err = IbvPostSRQRecvRes(ibRes, uint64(wrID))
				//if err != nil {
				//	return errors.New("post SRQ recv failed")
				//}
				//LogDebug(fmt.Sprintf("--- [CLIENT] SRQ recv posted wrID = %v", wrID))
				//ibRes.IbBuf = (*C.char)(unsafe.Pointer(uintptr(unsafe.Pointer(ibRes.IbBuf)) + unsafe.Sizeof(C.struct_ibv_wc{})))

				immData, err := C.ibv_get_imm_data(wcPtr)
				if err != nil {
					return errors.New("get imm data failed")
				}

				if immData == MSG_CLIENT_START {
					currentReady++
					//ready to send
					if currentReady == peerNum {
						startSend = true
						LogDebug("startSend ture")
						break
					}
				}
			}
		}
	}
	LogDebug("ready to send")

	err = QPSendData(peerNum, ibRes, wc, fileName, int64(ibRes.IbBufSize))
	if err != nil {
		return err
	}

	return nil
}

func QPSendData(peerNum int, ibRes *IBRes, wc *C.struct_ibv_wc, fileName string, chunkSize int64) error {

	chunkCount, fileSize, file, err := GetFileMeta(fileName, chunkSize)
	if err != nil {
		return err
	}
	defer file.Close()

	for i := 0; i < peerNum; i++ {
		reader := bufio.NewReader(file)
		for j := 0; j < chunkCount; j++ {

			curChunkSize := chunkSize
			if j == chunkCount-1 {
				curChunkSize = fileSize - int64(j)*chunkSize
			}

			chunk := make([]byte, curChunkSize)
			wrID := uintptr(unsafe.Pointer(ibRes.IbBuf))

			_, err := reader.Read(chunk)
			if err != nil {
				return err
			}
			ibRes.SetIbBufWithBytes(chunk)

			err = IbvPostSendRes(ibRes, i, uint64(wrID))
			if err != nil {
				return errors.New("post stop send failed")
			}
			LogDebug(fmt.Sprintf("--- [CLIENT] IbvPostSendRes wrID = %v", wrID))

			newPtr := unsafe.Pointer(uintptr(unsafe.Pointer(wc)) + unsafe.Sizeof(C.struct_ibv_wc{}))
			ibRes.IbBuf = (*C.char)(newPtr)
		}
	}
	LogDebug("post send done")

	LogDebug("start to poll CQ")
	var numAckedPeers int
	var stop bool
	for stop != true {
		var num int
		for num < 1 {
			num, err = IbvPollCQ(ibRes.Cq, 10, wc)
			if err != nil {
				return errors.New("poll CQ failed")
			}
		}
		LogDebug(fmt.Sprintf("  Poll CQ num: %v ", num))

		for i := 0; i < num; i++ {
			offset := uintptr(i * int(unsafe.Sizeof(C.struct_ibv_wc{})))
			newPtr := unsafe.Pointer(uintptr(unsafe.Pointer(wc)) + offset)
			wcPtr := (*C.struct_ibv_wc)(newPtr)
			LogDebug(fmt.Sprintf("wcStatus:%v wcOPcode: %v \n", wcPtr.status, wcPtr.opcode))

			if wcPtr.status != C.IBV_WC_SUCCESS {
				if wcPtr.opcode == C.IBV_WC_RECV {
					LogDebug(fmt.Sprintf("wcPtr.wr_id = %v,content: %v", wcPtr.wr_id, C.GoString((*C.char)(unsafe.Pointer(uintptr(wcPtr.wr_id))))))
					return errors.New("server recv failed")
				} else if wcPtr.opcode == C.IBV_WC_SEND {
					LogDebug(fmt.Sprintf("wcPtr.wr_id = %v,content: %v", wcPtr.wr_id, C.GoString((*C.char)(unsafe.Pointer(uintptr(wcPtr.wr_id))))))
					return errors.New("server send failed")
				} else {
					LogDebug(fmt.Sprintf("wcPtr.wr_id = %v,content: %v", wcPtr.wr_id, C.GoString((*C.char)(unsafe.Pointer(uintptr(wcPtr.wr_id))))))
					return errors.New("server unknown wc failed")
				}
			} else {
				if wcPtr.opcode == C.IBV_WC_RECV {

					immData, err := C.ibv_get_imm_data(wcPtr)
					if err != nil {
						return errors.New("get imm data failed")
					}

					if immData == MSG_CLIENT_STOP {
						numAckedPeers++
					}
					msgPtr := unsafe.Pointer(uintptr(wcPtr.wr_id))
					LogDebug(fmt.Sprintf("[RECV] wcPtr.wr_id = %v ,immData = %v \n", wcPtr.wr_id, immData))
					LogInfo(fmt.Sprintf("WC RECV msg:%v ,immData:%v\n", C.GoString((*C.char)(msgPtr)), immData))
					if numAckedPeers == peerNum {
						LogDebug("stop")
						stop = true
						break
					}
				} else if wcPtr.opcode == C.IBV_WC_SEND {
					msgPtr := unsafe.Pointer(uintptr(wcPtr.wr_id))
					LogInfo(fmt.Sprintf("WC SEND msg:%v\n", C.GoString((*C.char)(msgPtr))))
				}
			}
		}
	}
	return nil
}
