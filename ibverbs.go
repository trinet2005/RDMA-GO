package RDMAGO

import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

/*
#cgo CFLAGS: -I/usr/include/infiniband
#cgo LDFLAGS: -libverbs -L. -lwrapper

#include "wrapper.h"

#include <stdlib.h>
#include <infiniband/verbs.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <malloc.h>

int ibv_query_port_wrapper(struct ibv_context *context,
       uint8_t port_num, struct ibv_port_attr *port_attr);
*/
import "C"

type IbvDeviceContext struct {
	Ctx *C.struct_ibv_context
}

func GetIbvDeviceContext(ibRes *IBRes, deviceName string) (IbvDeviceContext, error) {
	var res IbvDeviceContext

	// get device list
	var numDevices C.int
	devList := C.ibv_get_device_list(&numDevices)
	if devList == nil {
		return res, errors.New("no RDMA devices found")
	}
	defer C.ibv_free_device_list(devList)

	// find target device
	var targetDevice *C.struct_ibv_device
	for i := 0; i < int(numDevices); i++ {
		devices := (*[1 << 30]*C.struct_ibv_device)(unsafe.Pointer(devList))[:numDevices:numDevices]
		device := devices[i]
		if device == nil {
			continue
		}
		if C.GoString(C.ibv_get_device_name(device)) == deviceName {
			targetDevice = device
			break
		}
	}

	if targetDevice == nil {
		return res, errors.New(fmt.Sprintf("No RDMA device found with name %s\n", deviceName))
	}

	// get device context
	context := C.ibv_open_device(targetDevice)
	if context == nil {
		return res, errors.New("failed to open device")
	}

	res.Ctx = context

	// query device attributes
	var deviceAttr C.struct_ibv_device_attr
	if err := C.ibv_query_device(context, &deviceAttr); err != 0 {
		return res, errors.New("failed to query device")
	}

	return res, nil
}

func IbvCloseDevice(ibRes *IBRes) error {
	_, err := C.ibv_close_device(ibRes.Ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to close device: %v", err))
	}
	return nil
}

func IbvAllocPD(ibRes *IBRes) error {
	pd, err := C.ibv_alloc_pd(ibRes.Ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to allocate protection domain: %v", err))
	}
	if pd == nil {
		return errors.New("failed to allocate protection domain")
	}
	ibRes.Pd = pd
	return nil
}

func IbvDeallocPD(ibRes *IBRes) error {
	_, err := C.ibv_dealloc_pd(ibRes.Pd)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to deallocate protection domain: %v", err))
	}
	return nil
}

// IbvQueryGid queries the GID of the specified port.
// context：指向 struct ibv_context 结构体的指针，表示 InfiniBand 设备的上下文。
// port_num：要查询的端口号。通常从 1 开始编号。
// index：用于多个 GID 的索引值，表示要查询的具体 GID。
// gid：用于存储查询结果的 union ibv_gid 结构体或其指针。
func IbvQueryGid(ibRes *IBRes) error {
	//index 固定为1选择index的值
	_, err := C.ibv_query_gid(ibRes.Ctx, IB_PORT, 1, &ibRes.Gid)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to query device: %v", err))
	}
	return nil
}

func IbvRegMR(ibRes *IBRes, IbBufSize int) error {
	ibRes.IbBufSize = C.ulong(IbBufSize)
	ptr := C.malloc(C.size_t(ibRes.IbBufSize))

	ibRes.IbBuf = (*C.char)(ptr)

	if ibRes.IbBuf == nil {
		return errors.New("failed to allocate memory")
	}

	mr, err := C.ibv_reg_mr(ibRes.Pd, unsafe.Pointer(ibRes.IbBuf), ibRes.IbBufSize,
		C.IBV_ACCESS_LOCAL_WRITE|
			C.IBV_ACCESS_REMOTE_WRITE|
			C.IBV_ACCESS_REMOTE_READ)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to allocate memory region: %v", err))
	}
	if mr == nil {
		return errors.New("failed to allocate memory region")
	}
	ibRes.Mr = mr
	return nil
}

func IbvDeregMR(ibRes *IBRes) error {
	_, err := C.ibv_dereg_mr(ibRes.Mr)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to deallocate memory region: %v", err))
	}
	return nil
}

func IbvQueryDevice(ibRes *IBRes) error {
	_, err := C.ibv_query_device(ibRes.Ctx, &ibRes.DevAttr)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to query device: %v", err))
	}
	return nil
}

func IbvCreateCQ(ibRes *IBRes) error {
	cq, err := C.ibv_create_cq(ibRes.Ctx, ibRes.DevAttr.max_cqe, nil, nil, 0)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create completion queue: %v", err))
	}
	if cq == nil {
		return errors.New("failed to create completion queue")
	}
	ibRes.Cq = cq
	return nil
}

func IbvDestroyCQ(ibRes *IBRes) error {
	_, err := C.ibv_destroy_cq(ibRes.Cq)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to destroy completion queue: %v", err))
	}
	return nil
}

func IbvCreateSRQ(ibRes *IBRes) error {
	attr := C.struct_ibv_srq_init_attr{
		attr: C.struct_ibv_srq_attr{
			max_wr:    (C.uint)(ibRes.DevAttr.max_srq_wr),
			max_sge:   0,
			srq_limit: 0,
		},
	}

	srq, err := C.ibv_create_srq(ibRes.Pd, &attr)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create shared receive queue: %v", err))
	}
	if srq == nil {
		return errors.New("failed to create shared receive queue")
	}
	ibRes.Srq = srq

	return nil
}

func IbvDestroySRQ(ibRes *IBRes) error {
	_, err := C.ibv_destroy_srq(ibRes.Srq)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to destroy shared receive queue: %v", err))
	}
	return nil
}

func IbvCreateQP(ibRes *IBRes) error {
	attr := C.struct_ibv_qp_init_attr{
		send_cq: ibRes.Cq,
		recv_cq: ibRes.Cq,
		srq:     ibRes.Srq,
		cap: C.struct_ibv_qp_cap{
			max_send_wr:  (C.uint)(ibRes.DevAttr.max_qp_wr),
			max_recv_wr:  (C.uint)(ibRes.DevAttr.max_qp_wr),
			max_send_sge: 1,
			max_recv_sge: 1,
		},
		qp_type: C.IBV_QPT_RC,
	}

	qp, err := C.ibv_create_qp(ibRes.Pd, &attr)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to create queue pair: %v", err))
	}
	if qp == nil {
		return errors.New("failed to create queue pair")
	}
	ibRes.Qp = qp

	return nil
}

func IbvDestroyQP(ibRes *IBRes) error {
	_, err := C.ibv_destroy_qp(ibRes.Qp)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to destroy queue pair: %v", err))
	}
	return nil
}

func IbvModifyQP(qp *C.struct_ibv_qp, attr *C.struct_ibv_qp_attr, mask C.int) error {
	res, err := C.ibv_modify_qp(qp, attr, mask)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to modify qp: %v ,res :%v", err, res))
	}
	if res != 0 {
		return errors.New("failed to modify qp")
	}
	return nil
}

func IbvModifyQPInit(qp *C.struct_ibv_qp) error {
	attr := (*C.struct_ibv_qp_attr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_qp_attr{}))))
	if attr == nil {
		return errors.New("failed to allocate memory")
	}
	defer C.free(unsafe.Pointer(attr))

	C.memset(unsafe.Pointer(attr), 0, C.size_t(unsafe.Sizeof(*attr)))

	// init attributes
	attr.qp_state = C.IBV_QPS_INIT
	attr.pkey_index = 0
	attr.port_num = IBV_PORT_NUM
	attr.qp_access_flags = C.IBV_ACCESS_LOCAL_WRITE |
		C.IBV_ACCESS_REMOTE_WRITE |
		C.IBV_ACCESS_REMOTE_READ |
		C.IBV_ACCESS_REMOTE_ATOMIC

	err := IbvModifyQP(qp, attr, C.IBV_QP_STATE|
		C.IBV_QP_PKEY_INDEX|
		C.IBV_QP_PORT|
		C.IBV_QP_ACCESS_FLAGS)
	if err != nil {
		return err
	}

	return nil
}

func IbvModifyQPRTRDefault(qp *C.struct_ibv_qp, targetQPNum C.uint, targetLid C.ushort, rGid C.union_ibv_gid) error {
	qpAttr := IBVQPAttr{
		destQPNum:       targetQPNum,
		pathMTU:         C.IBV_MTU_4096,
		RQPsn:           0,
		maxDestRDAtomic: 1,
		minRNRTimer:     12,

		ahAttr: IBVAHAttr{
			dlid:         targetLid,
			grhDgid:      rGid,
			isGlobal:     1,
			grhHopLimit:  1,
			grhSgidIndex: 1,
			SL:           0,
			srcPathBits:  0,
			portNum:      IBV_PORT_NUM,
		},
	}
	return IbvModifyQPRTR(qp, qpAttr)

}

func IbvModifyQPRTR(qp *C.struct_ibv_qp, qpAttr IBVQPAttr) error {
	attr := (*C.struct_ibv_qp_attr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_qp_attr{}))))
	if attr == nil {
		return errors.New("failed to allocate memory")
	}
	defer C.free(unsafe.Pointer(attr))

	C.memset(unsafe.Pointer(attr), 0, C.size_t(unsafe.Sizeof(*attr)))

	// init attributes
	attr.qp_state = C.IBV_QPS_RTR
	attr.path_mtu = C.IBV_MTU_4096
	attr.dest_qp_num = qpAttr.destQPNum
	attr.rq_psn = qpAttr.RQPsn
	attr.max_dest_rd_atomic = qpAttr.maxDestRDAtomic
	attr.min_rnr_timer = qpAttr.minRNRTimer
	attr.ah_attr.is_global = qpAttr.ahAttr.isGlobal

	attr.ah_attr.grh.hop_limit = qpAttr.ahAttr.grhHopLimit
	attr.ah_attr.grh.dgid = qpAttr.ahAttr.grhDgid
	attr.ah_attr.grh.sgid_index = qpAttr.ahAttr.grhSgidIndex

	attr.ah_attr.dlid = qpAttr.ahAttr.dlid
	attr.ah_attr.sl = qpAttr.ahAttr.SL
	attr.ah_attr.src_path_bits = qpAttr.ahAttr.srcPathBits
	attr.ah_attr.port_num = qpAttr.ahAttr.portNum

	_, err := C.ibv_modify_qp_wrapper(qp, attr, C.IBV_QP_STATE|
		C.IBV_QP_AV|
		C.IBV_QP_PATH_MTU|
		C.IBV_QP_DEST_QPN|
		C.IBV_QP_RQ_PSN|
		C.IBV_QP_MAX_DEST_RD_ATOMIC|
		C.IBV_QP_MIN_RNR_TIMER)
	if err != nil {
		return err
	}

	return nil
}

func IbvModifyQPRTSDefault(qp *C.struct_ibv_qp) error {
	qpAttr := IBVQPAttr{
		timeout:     14,
		retryCount:  7,
		rnrRetry:    7,
		SQPsn:       0,
		maxRdAtomic: 1,
	}
	return IbvModifyQPRTS(qp, qpAttr)
}

// IbvModifyQPRTS modifies the queue pair to the ready to send state.
// qp: pointer to the queue pair to modify.
// timeOut: the time(ms) to wait for a response.
// retryCnt: the number of times to retry.
// RNRRetry: the number of times to retry after a receiver not ready error.
// sqPsn: the starting packet sequence number.
// maxRdAtomic: the maximum number of atomic operations each receive work request can handle.

func IbvModifyQPRTS(qp *C.struct_ibv_qp, qpAttr IBVQPAttr) error {
	attr := (*C.struct_ibv_qp_attr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_qp_attr{}))))
	if attr == nil {
		return errors.New("failed to allocate memory")
	}
	defer C.free(unsafe.Pointer(attr))

	// 初始化结构体
	attr.qp_state = C.IBV_QPS_RTS
	attr.timeout = qpAttr.timeout
	attr.retry_cnt = qpAttr.retryCount
	attr.rnr_retry = qpAttr.rnrRetry
	attr.sq_psn = qpAttr.SQPsn
	attr.max_rd_atomic = qpAttr.maxRdAtomic

	mask := (C.int)(C.IBV_QP_STATE | C.IBV_QP_TIMEOUT | C.IBV_QP_RETRY_CNT | C.IBV_QP_RNR_RETRY |
		C.IBV_QP_SQ_PSN | C.IBV_QP_MAX_QP_RD_ATOMIC)

	err := IbvModifyQP(qp, attr, mask)
	if err != nil {
		return err
	}

	return nil
}

func IbvQueryPort(ibRes *IBRes, portNum int) error {

	portAttr := (*C.struct_ibv_port_attr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_port_attr{}))))

	res, err := C.ibv_query_port_wrapper(ibRes.Ctx, C.uint8_t(portNum), portAttr)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to query port: %v", err))
	}
	if res != 0 {
		return errors.New("failed to query port")
	}

	ibRes.PortAttr = portAttr
	return nil
}

func IbvPostSRQRecvRes(ibRes *IBRes, wrID uint64) error {
	return IbvPostSRQRecv(ibRes.Srq, C.ulong(wrID), ibRes.Mr.lkey, C.uint(ibRes.IbBufSize), ibRes.IbBuf)
}

func IbvPostSRQRecv(srq *C.struct_ibv_srq, wrID C.ulong, lkey, bufSize C.uint, buf *C.char) error {
	badRecvWr := (*C.struct_ibv_recv_wr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_recv_wr{}))))
	defer C.free(unsafe.Pointer(badRecvWr))

	cstrAddr := uintptr(unsafe.Pointer(buf))
	addrBuf := C.ulong(cstrAddr)

	list := C.struct_ibv_sge{
		addr:   addrBuf,
		length: bufSize,
		lkey:   lkey,
	}

	recvWr := (*C.struct_ibv_recv_wr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_recv_wr{}))))
	recvWr.wr_id = wrID
	recvWr.sg_list = &list
	recvWr.num_sge = 1

	defer C.free(unsafe.Pointer(recvWr))

	_, err := C.ibv_post_srq_recv(srq, recvWr, &badRecvWr)
	if err != nil {
		return errors.New(fmt.Sprintf("[IbvPostSRQRecv] failed to post recv: %v", err))
	}
	return nil
}

func IbvPostSendRes(ibRes *IBRes, immData int, wrID uint64) error {
	return IbvPostSend(C.uint(ibRes.IbBufSize), ibRes.Mr.lkey, C.ulong(wrID), C.uint(immData), ibRes.Qp, ibRes.IbBuf)
}

func IbvPostSend(reqSize C.uint, lkey C.uint, wrID C.ulong, immData C.uint, qp *C.struct_ibv_qp, buf *C.char) error {
	badSendWr := (*C.struct_ibv_send_wr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_send_wr{}))))
	defer C.free(unsafe.Pointer(badSendWr))

	cstrAddr := uintptr(unsafe.Pointer(buf))
	addrBuf := C.ulong(cstrAddr)

	list := C.struct_ibv_sge{
		addr:   addrBuf,
		length: reqSize,
		lkey:   lkey,
	}

	sendWr := (*C.struct_ibv_send_wr)(C.malloc(C.size_t(unsafe.Sizeof(C.struct_ibv_send_wr{}))))
	sendWr.wr_id = wrID
	sendWr.sg_list = &list
	sendWr.num_sge = 1
	sendWr.opcode = C.IBV_WR_SEND_WITH_IMM
	sendWr.send_flags = C.IBV_SEND_SIGNALED

	_, err := C.ibv_post_send_wrapper(qp, sendWr, &badSendWr, immData)
	if err != nil {
		return errors.New(fmt.Sprintf("[IbvPostSend] failed to post send: %v", err))
	}
	return nil
}

// IbvPollCQ polls the completion queue for completion events.
// numEntries specifies the maximum number of completion events to poll.
// wc is a pointer to an array of completion queue work completion structs.
// Returns the number of completion events polled.
// if numEntries is greater than the returned num, means the CQ is empty
func IbvPollCQ(cq *C.struct_ibv_cq, numEntries C.int, wc *C.struct_ibv_wc) (int, error) {
	num, err := C.ibv_poll_cq(cq, numEntries, wc)
	if err != nil {
		return 0, errors.New(fmt.Sprintf("failed to poll completion queue: %v", err))
	}
	return int(num), nil
}

func CreateWC(numWC int) (*C.struct_ibv_wc, error) {
	wc := (*C.struct_ibv_wc)(C.calloc(C.ulong(numWC), C.sizeof_struct_ibv_wc))
	if wc == nil {
		return nil, errors.New("[CreateWC] failed to allocate memory")
	}
	return wc, nil
}

func DestroyWC(wc *C.struct_ibv_wc) {
	C.free(unsafe.Pointer(wc))
}
