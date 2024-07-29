package RDMAGO

import "C"
import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"strings"
	"unsafe"
)

type GoQPInfo struct {
	QpNum uint32   `json:"qp_num"`
	Lid   uint16   `json:"lid"`
	Gid   [16]byte `json:"gid"`
}

func ConvertToGoQPInfo(qpInfo QPInfo) GoQPInfo {
	var gid [16]byte
	copy(gid[:], C.GoBytes(unsafe.Pointer(&qpInfo.Gid), 16))

	return GoQPInfo{
		QpNum: uint32(qpInfo.QpNum),
		Lid:   uint16(qpInfo.Lid),
		Gid:   gid,
	}
}

func ConvertToCQPInfo(goQPInfo GoQPInfo) QPInfo {
	var qpInfo QPInfo
	qpInfo.QpNum = C.uint(goQPInfo.QpNum)
	qpInfo.Lid = C.ushort(goQPInfo.Lid)

	var gid C.union_ibv_gid
	copy((*[16]byte)(unsafe.Pointer(&gid))[:], goQPInfo.Gid[:])

	qpInfo.Gid = gid
	return qpInfo
}

// StartServer start server
func StartServer(port string, info QPInfo) (error, *QPInfo) {
	listener, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return errors.New("[Socket] Error starting server: " + err.Error()), nil
	}
	defer listener.Close()

	conn, err := listener.Accept()
	if err != nil {
		return errors.New("[Socket] Error accepting connection: " + err.Error()), nil
	}

	err, QPInfo := handleConnection(conn, info)
	if err != nil {
		return err, nil
	}

	return nil, QPInfo
}

func handleConnection(conn net.Conn, info QPInfo) (error, *QPInfo) {
	defer conn.Close()
	message, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return errors.New("[Socket] Error reading message: " + err.Error()), nil
	}

	jsonData, err := json.Marshal(ConvertToGoQPInfo(info))
	if err != nil {
		return errors.New("[Socket] Error marshalling QP info: " + err.Error()), nil
	}

	_, err = conn.Write(append(jsonData, '\n'))
	if err != nil {
		return errors.New("[Socket] Error writing response: " + err.Error()), nil
	}

	message = strings.TrimSuffix(message, "\n")

	var goQPInfo GoQPInfo

	err = json.Unmarshal([]byte(message), &goQPInfo)
	if err != nil {
		return err, nil
	}

	QPInfo := ConvertToCQPInfo(goQPInfo)

	return nil, &QPInfo
}

// StartClient start client
func StartClient(address string, info QPInfo) (error, *QPInfo) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return errors.New("[Socket] Error connecting to server: " + err.Error()), nil
	}
	defer conn.Close()

	jsonData, err := json.Marshal(ConvertToGoQPInfo(info))
	if err != nil {
		return errors.New("[Socket] Error marshalling QP info: " + err.Error()), nil
	}

	_, err = conn.Write(append(jsonData, '\n'))
	if err != nil {
		return errors.New("[Socket] Error writing message: " + err.Error()), nil
	}

	response, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return errors.New("Error reading response: " + err.Error()), nil
	}

	response = strings.TrimSuffix(response, "\n")

	var goQPInfo GoQPInfo

	err = json.Unmarshal([]byte(response), &goQPInfo)
	if err != nil {
		return err, nil
	}

	QPInfo := ConvertToCQPInfo(goQPInfo)

	return nil, &QPInfo
}
